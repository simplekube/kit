package k8s

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Task defines the task against a single Kubernetes
// resource. This defines one of the smallest unit of
// Kubernetes work.
type Task struct {
	// It describes the intention of this task
	//
	// e.g. It "should create a Pod"
	// e.g. It "should assert presence of a Deployment"
	// e.g. It "should assert absence of ConfigMap"
	It string

	// Action defines the operation i.e. create, delete, etc.
	Action ActionType

	// Resource represents the Kubernetes object against
	// which this task is supposed to get executed
	Resource client.Object

	// Assert defines the verification to be executed post
	// the execution of this task e.g. Equals, NotEquals,
	// NotFound, etc.
	Assert AssertType

	// Skip will skip run of this task if it returns true
	Skip func(object client.Object) (bool, error)

	// PostAction accepts a callback function that gets executed
	// against the resource found in the Kubernetes cluster
	// i.e. actual object (also known as observed state)
	PostAction func(object client.Object) error

	// PreAction accepts a callback function that gets executed
	// against the provided resource before invoking this task
	PreAction func(object client.Object) error

	// TODO (@amit.das)
	// IgnoreVersions can contain the Kubernetes versions
	// that should ignore this specification from getting
	// executed, this is optional
	// IgnoreVersions []string
}

func (t *Task) Build() Runner {
	return &runnableTask{
		task: t,
	}
}

func (t *Task) Run(ctx context.Context, opts ...RunOption) error {
	return t.Build().Run(ctx, opts...)
}

// runnableTask executes a Kubernetes task
type runnableTask struct {
	client    client.Client
	scheme    *runtime.Scheme
	task      *Task
	givenObj  client.Object
	actualObj client.Object
	isSkip    bool
}

// compile time check to AssertType if the structure
// runnableTask implements the interface Runner
var _ Runner = (*runnableTask)(nil)

func (r *runnableTask) Run(ctx context.Context, opts ...RunOption) error {
	var err error

	var errWrap = func(err error) error {
		if err == nil {
			return nil
		}
		var reporting = r.actualObj
		if reporting == nil {
			reporting = r.task.Resource
		}
		gvk, _ := apiutil.GVKForObject(reporting, scheme.Scheme)
		return errors.Wrapf(
			err,
			"task %q: action %q: assert %q: ns %q: name %q: gvk %q",
			fmt.Sprintf("It %s", r.task.It),
			r.task.Action,
			r.task.Assert,
			reporting.GetNamespace(),
			reporting.GetName(),
			gvk,
		)
	}

	// 0/ build the RunOptions instance
	runOpts, err := FromRunOptions(opts...)
	if err != nil {
		return errWrap(err)
	}

	// 1/ execute pre action logic
	err = r.preAction(ctx, *runOpts)
	if err != nil {
		return errWrap(err)
	}

	// 2/ verify if this task should be run
	if r.isSkip {
		// TODO (@amit.das) log this at V(2) level
		// will not execute this task
		return nil
	}

	// 3/ execute the action
	err = r.action(ctx, *runOpts)
	if err != nil {
		return errWrap(err)
	}

	// 4/ execute post action logic
	err = r.postAction(ctx, *runOpts)
	if err != nil {
		return errWrap(err)
	}

	// 5/ execute assertion logic
	return errWrap(r.assert(ctx, *runOpts))
}

func (r *runnableTask) preAction(ctx context.Context, opts RunOptions) error {
	// make copies of the given resource
	if r.task.Resource != nil {
		r.givenObj = r.task.Resource.DeepCopyObject().(client.Object)
		r.actualObj = r.task.Resource.DeepCopyObject().(client.Object)
	}

	if r.task.Skip != nil {
		isSkip, err := r.task.Skip(r.givenObj)
		if err != nil {
			return err
		}
		r.isSkip = isSkip
	}
	if r.isSkip {
		// no need to proceed further
		return nil
	}

	// ensure Kubernetes client is set
	r.client = opts.Client
	if r.client == nil {
		config := config.GetConfigOrDie()
		c, err := client.New(config, client.Options{})
		if err != nil {
			return errors.Wrap(err, "failed to initialise client")
		}
		r.client = c
	}

	// ensure Kubernetes scheme is set
	r.scheme = opts.Scheme
	if r.scheme == nil {
		// default to the scheme that understands all native Kubernetes API schemas
		r.scheme = scheme.Scheme
	}

	// run the callback if any against the given & actual objects
	//
	// Note: since given and actual objects are still same in pre-action
	// both of them are run against PreAction callback
	if r.task.PreAction != nil {
		err := r.task.PreAction(r.givenObj)
		if err != nil {
			return err
		}
		err = r.task.PreAction(r.actualObj)
		if err != nil {
			return err
		}
	}

	// assert can be optional if Task is only action based
	if r.task.Assert == "" {
		r.task.Assert = AssertTypeIsNoop
	}

	return nil
}

func (r *runnableTask) action(ctx context.Context, opts RunOptions) error {
	var err error

	switch r.task.Action {
	case ActionTypeCreate:
		err = r.create(ctx, opts)
	case ActionTypeGet:
		err = r.get(ctx, opts)
	case ActionTypeDelete:
		err = r.delete(ctx, opts)
	case ActionTypeCreateOrMerge:
		err = r.createOrMerge(ctx, opts)
	case ActionTypeUpdate:
		err = r.update(ctx, opts)
	default:
		err = errors.New("un-supported action")
	}
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		// IsNotFound error is not treated as an error since
		// observed object is set to nil
		err = nil
		r.actualObj = nil
	}
	return err
}

func (r *runnableTask) postAction(ctx context.Context, opts RunOptions) error {
	var err error
	if r.task.PostAction != nil {
		err = r.task.PostAction(r.actualObj)
	}

	return err
}

func (r *runnableTask) delete(ctx context.Context, opts RunOptions) error {
	dOpts := &client.DeleteOptions{
		GracePeriodSeconds: new(int64), // delete now
	}
	return r.client.Delete(context.Background(), r.actualObj, dOpts)
}

func (r *runnableTask) get(ctx context.Context, opts RunOptions) error {
	return r.client.Get(context.Background(), client.ObjectKeyFromObject(r.actualObj), r.actualObj)
}

func (r *runnableTask) create(ctx context.Context, opts RunOptions) error {
	err := r.client.Create(context.Background(), r.actualObj)
	if err == nil && r.actualObj != nil {
		// if this resource was created successfully
		// then push it to the garbage collection registry
		getDefaultGCRegistry().Register(&DeletingTask{
			Resource: r.actualObj.DeepCopyObject().(client.Object),
		})
	}
	return err
}

// createOrMerge merges the provided Resource in the Kubernetes cluster
//
// Note: Merge happens only if there is a difference between given
// and observed states. If there is no difference this operation
// becomes a noop.
func (r *runnableTask) createOrMerge(ctx context.Context, opts RunOptions) error {
	result, err := CreateOrMerge(context.Background(), r.client, r.scheme, r.actualObj)
	if result == OperationResultCreated && r.actualObj != nil {
		getDefaultGCRegistry().Register(&DeletingTask{
			Resource: r.actualObj.DeepCopyObject().(client.Object),
		})
	}
	return err
}

// update will update the provided resource in the Kubernetes cluster
func (r *runnableTask) update(ctx context.Context, opts RunOptions) error {
	return r.client.Update(context.Background(), r.actualObj)
}

func (r *runnableTask) assert(ctx context.Context, opts RunOptions) error {
	var matchOrErr = func(want bool) error {
		if r.actualObj == nil {
			return errors.Errorf("nil actual object: cannot run equality check")
		}
		if r.givenObj == nil {
			return errors.Errorf("nil given object: cannot run equality check")
		}
		isEqual, err := IsEqual(r.actualObj, r.givenObj)
		if err != nil {
			return errors.Wrap(err, "failed to verify object equality")
		}
		// we error if got is not equal to want
		//
		// NOTE: want & got variables are used to
		// either represent 'presence of a diff' or
		// 'absence of a diff'
		var (
			wantStr = "not equals"
			gotStr  = "not equals"
		)
		if want {
			wantStr = "equals"
		}
		if isEqual {
			gotStr = "equals"
		}
		if isEqual != want {
			return errors.Errorf("assert failed: want %q: got %q", wantStr, gotStr)
		}
		return nil
	}

	var err error
	switch r.task.Assert {
	case AssertTypeIsEquals:
		err = matchOrErr(true)
	case AssertTypeIsNotEquals:
		err = matchOrErr(false)
	case AssertTypeIsNotFound:
		if r.actualObj != nil {
			err = errors.New("assert failed: got a resource while expecting none")
		}
	case AssertTypeIsFound:
		if r.actualObj == nil {
			err = errors.New("assert failed: got no resource while expecting one")
		}
	case AssertTypeIsNoop:
		// do nothing since this task might be only an action
	default:
		err = errors.New("un-supported assert type")
	}

	return err
}

// ListingTask defines the structure to list Kubernetes resources
// of same type. This defines one of the smallest unit of Kubernetes work.
type ListingTask struct {
	// It describes the intention of this task
	//
	// e.g. It "should filter relevant Pods"
	// e.g. It "should list Deployments belonging to a namespace"
	It string

	// Resource represents the Kubernetes object against
	// which this task is supposed to get executed
	Resource client.ObjectList

	// ListOptions provide the filtering options if any
	// that are executed during the list operation
	ListOptions []client.ListOption

	// PostAction accepts a callback function that gets executed
	// against the resource(s) found in the Kubernetes cluster
	// i.e. actual objects (also known as observed states)
	PostAction func(object client.ObjectList) error

	// PreAction accepts a callback function that gets executed
	// against the provided resource before invoking this task
	PreAction func(object client.ObjectList) error
}

func (t *ListingTask) Build() Runner {
	return &listableTask{
		task: t,
	}
}

func (t *ListingTask) Run(ctx context.Context, opts ...RunOption) error {
	return t.Build().Run(ctx, opts...)
}

// listableTask executes a listing based Kubernetes operation
type listableTask struct {
	client    client.Client
	task      *ListingTask
	givenObj  client.ObjectList
	actualObj client.ObjectList
}

// compile time check to verify if the structure
// listableTask implements the interface Runner
var _ Runner = (*listableTask)(nil)

func (l *listableTask) Run(ctx context.Context, opts ...RunOption) error {
	var err error

	var errWrap = func(err error) error {
		if err == nil {
			return nil
		}
		var reporting = l.actualObj
		if reporting == nil {
			reporting = l.task.Resource
		}
		gvk, _ := apiutil.GVKForObject(reporting, scheme.Scheme)
		return errors.Wrapf(
			err,
			"task %q: gvk %q",
			fmt.Sprintf("It %s", l.task.It),
			gvk,
		)
	}

	// 0/ build the RunOptions instance
	runOpts, err := FromRunOptions(opts...)
	if err != nil {
		return errWrap(err)
	}

	// 1/ execute pre action logic
	err = l.preAction(ctx, *runOpts)
	if err != nil {
		return errWrap(err)
	}

	// 2/ execute the action
	err = l.action(ctx, *runOpts)
	if err != nil {
		return errWrap(err)
	}

	// 3/ execute post action logic
	return errWrap(l.postAction(ctx, *runOpts))
}

func (l *listableTask) preAction(ctx context.Context, opts RunOptions) error {
	// ensure Kubernetes client is set
	l.client = opts.Client
	if l.client == nil {
		config := config.GetConfigOrDie()
		c, err := client.New(config, client.Options{})
		if err != nil {
			return errors.Wrap(err, "failed to initialise client")
		}
		l.client = c
	}

	// make copies of the given resource
	if l.task.Resource != nil {
		l.givenObj = l.task.Resource.DeepCopyObject().(client.ObjectList)
		l.actualObj = l.task.Resource.DeepCopyObject().(client.ObjectList)
	}

	// run the callback if any against the given & actual objects
	//
	// Note: given and actual objects are still same
	if l.task.PreAction != nil {
		err := l.task.PreAction(l.givenObj)
		if err != nil {
			return err
		}
		err = l.task.PreAction(l.actualObj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *listableTask) action(ctx context.Context, opts RunOptions) error {
	err := l.list(ctx, opts)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		// IsNotFound error is not treated as an error since
		// observed object is set to nil
		err = nil
		l.actualObj = nil
	}
	return err
}

func (l *listableTask) postAction(ctx context.Context, opts RunOptions) error {
	var err error
	if l.task.PostAction != nil {
		err = l.task.PostAction(l.actualObj)
	}

	return err
}

func (l *listableTask) list(ctx context.Context, opts RunOptions) error {
	return l.client.List(context.Background(), l.actualObj, l.task.ListOptions...)
}
