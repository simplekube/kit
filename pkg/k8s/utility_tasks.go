package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/simplekube/kit/pkg/k8sutil"

	"github.com/simplekube/kit/pkg/util"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// This file provides various structures that compose Task(s)
// to suit specific needs. In other words these showcases
// building things in a composable way with Task & ListingTask
// being the smallest units of Kubernetes work.

// Job are used to run multiple Runner instances together
type Job []Runner

// compile time check to AssertType if the structure
// Job implements the interface Runner
var _ Runner = (Job)(nil)

func (r Job) Run(ctx context.Context, opts ...RunOption) error {
	count := len(r)
	for idx, runner := range r {
		err := runner.Run(ctx, opts...)
		if err != nil {
			return errors.WithMessagef(err, "#%d/%d", idx+1, count)
		}
	}
	return nil
}

// Tasks is used to run more than one instances of Task
type Tasks []*Task

// compile time check to verify if the structure
// Tasks implements the interface Runner
var _ Runner = (Tasks)(nil)

func (t Tasks) Run(ctx context.Context, opts ...RunOption) error {
	count := len(t)
	for idx, task := range t {
		err := task.Run(ctx, opts...)
		if err != nil {
			return errors.WithMessagef(err, "#%d/%d", idx+1, count)
		}
	}
	return nil
}

// Lists is used to run more than one instance of ListingTask
type Lists []*ListingTask

// compile time check to verify if the structure
// Lists implements the interface Runner
var _ Runner = (Lists)(nil)

func (l Lists) Run(ctx context.Context, opts ...RunOption) error {
	count := len(l)
	for idx, listing := range l {
		err := listing.Run(ctx, opts...)
		if err != nil {
			return errors.WithMessagef(err, "#%d/%d", idx+1, count)
		}
	}
	return nil
}

// PodExecTask is used to execute command against a Pod
type PodExecTask struct {
	It            string
	PodName       string
	PodNamespace  string
	ContainerName string
	Command       []string
}

// compile time check to verify if the structure
// PodExecTask implements the interface Runner
var _ Runner = (*PodExecTask)(nil)

func (t *PodExecTask) Run(ctx context.Context, opts ...RunOption) error {
	if t.It == "" {
		return errors.New("missing description")
	}
	if t.PodNamespace == "" {
		return errors.New("missing pod namespace")
	}
	if t.PodName == "" {
		return errors.New("missing pod name")
	}
	// extract kubernetes client
	runOpts, err := FromRunOptions(opts...)
	if err != nil {
		return err
	}
	var klientset = runOpts.Clientset
	var conf *rest.Config
	if klientset == nil {
		var err error
		conf = config.GetConfigOrDie()
		klientset, err = kubernetes.NewForConfig(conf)
		if err != nil {
			return errors.Wrap(err, "failed to initialise clientset")
		}
	}
	req := klientset.CoreV1().RESTClient().Post().Resource("pods").
		Name(t.PodName).
		Namespace(t.PodNamespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: t.ContainerName,
		Command:   t.Command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)
	_, err = remotecommand.NewSPDYExecutor(conf, "POST", req.URL())

	return errors.Wrapf(err, "failed to exec into pod: url %q", req.URL().String())
}

// CustomTask provides the ability to execute any custom logic
// while adhering to Runner interface
type CustomTask struct {
	It     string
	Action func(ctx context.Context, opts ...RunOption) error
}

// compile time check to verify if the structure
// CustomTask implements the interface Runner
var _ Runner = (*CustomTask)(nil)

func (t *CustomTask) Run(ctx context.Context, opts ...RunOption) error {
	if t.It == "" {
		return errors.New("missing description")
	}
	if t.Action == nil {
		return errors.New("missing action")
	}
	return errors.Wrapf(t.Action(ctx, opts...), "task %q: action 'Custom'", fmt.Sprintf("It %s", t.It))
}

// AssertIsNotFoundTask is used to first fetch the provided resource and
// then assert the given state is not found in the Kubernetes cluster
type AssertIsNotFoundTask struct {
	It       string // long description of this task
	Resource client.Object
}

// compile time check to verify if the structure
// AssertIsNotFoundTask implements the interface Runner
var _ Runner = (*AssertIsNotFoundTask)(nil)

func (t *AssertIsNotFoundTask) Run(ctx context.Context, opts ...RunOption) error {
	var desc = "should assert the given state is not found in the cluster"
	if t.It != "" {
		desc = t.It
	}
	task := &Task{
		It:       desc,
		Action:   ActionTypeGet,
		Resource: t.Resource,
		Assert:   AssertTypeIsNotFound,
	}
	return task.Run(ctx, opts...)
}

// AssertIsNotEqualsTask is used to first fetch the provided resource and
// then assert the given state does not match the state observed in the
// Kubernetes cluster
type AssertIsNotEqualsTask struct {
	It       string // long description of this task
	Resource client.Object
}

// compile time check to verify if the structure
// AssertIsNotEqualsTask implements the interface Runner
var _ Runner = (*AssertIsNotEqualsTask)(nil)

func (t *AssertIsNotEqualsTask) Run(ctx context.Context, opts ...RunOption) error {
	var desc = "should assert the given state does not match the observed state"
	if t.It != "" {
		desc = t.It
	}
	task := &Task{
		It:       desc,
		Action:   ActionTypeGet,
		Resource: t.Resource,
		Assert:   AssertTypeIsNotEquals,
	}
	return task.Run(ctx, opts...)
}

// AssertIsEqualsTask is used to first fetch the provided resource and
// then assert the given state matches with the state observed in the
// Kubernetes cluster
type AssertIsEqualsTask struct {
	// [optional] long description of this task
	It string

	// Resource is the Kubernetes object against which
	// the API call in made
	Resource client.Object

	// [optional] callback that gets executed before making
	// the K8s API call
	PreAction func(object client.Object) error

	// [optional] callback that gets executed after making
	// the K8s API call
	PostAction func(object client.Object) error
}

// compile time check to verify if the structure
// AssertIsEqualsTask implements the interface Runner
var _ Runner = (*AssertIsEqualsTask)(nil)

func (t *AssertIsEqualsTask) Run(ctx context.Context, opts ...RunOption) error {
	var desc = "should assert the given state matches the observed state"
	if t.It != "" {
		desc = t.It
	}
	task := &Task{
		It:         desc,
		Action:     ActionTypeGet,
		Resource:   t.Resource,
		PreAction:  t.PreAction,
		PostAction: t.PostAction,
		Assert:     AssertTypeIsEquals,
	}
	return task.Run(ctx, opts...)
}

// CreateThenAssertIsEqualsTask is used to first create
// the provided resource and then assert the given state
// matches with the state observed in the Kubernetes cluster
type CreateThenAssertIsEqualsTask struct {
	// [optional] long description of this task
	It string

	// Resource is the Kubernetes object against which
	// the API call in made
	Resource client.Object

	// [optional] callback that gets executed before making
	// the K8s API call
	PreAction func(object client.Object) error

	// [optional] callback that gets executed after making
	// the K8s API call
	PostAction func(object client.Object) error
}

// compile time check to verify if the structure
// CreateThenAssertIsEqualsTask implements the interface Runner
var _ Runner = (*CreateThenAssertIsEqualsTask)(nil)

func (t *CreateThenAssertIsEqualsTask) Run(ctx context.Context, opts ...RunOption) error {
	var desc = "should create the resource and assert the given state matches the observed state"
	if t.It != "" {
		desc = t.It
	}
	task := &Task{
		It:         desc,
		Action:     ActionTypeCreate,
		Resource:   t.Resource,
		PreAction:  t.PreAction,
		PostAction: t.PostAction,
		Assert:     AssertTypeIsEquals,
	}
	return task.Run(ctx, opts...)
}

// UpsertThenAssertIsEqualsTask is used to first create or update
// the provided resource and then assert the given state
// matches the state observed in the Kubernetes cluster
type UpsertThenAssertIsEqualsTask struct {
	// [optional] long description of this task
	It string

	// Resource is the Kubernetes object against which
	// the API call in made
	Resource client.Object

	// [optional] callback that gets executed before making
	// the K8s API call
	PreAction func(object client.Object) error

	// [optional] callback that gets executed after making
	// the K8s API call
	PostAction func(object client.Object) error
}

// compile time check to verify if the structure
// UpsertThenAssertIsEqualsTask implements the interface Runner
var _ Runner = (*UpsertThenAssertIsEqualsTask)(nil)

func (t *UpsertThenAssertIsEqualsTask) Run(ctx context.Context, opts ...RunOption) error {
	var desc = "should upsert the resource and assert the given state matches the observed state"
	if t.It != "" {
		desc = t.It
	}
	task := &Task{
		It:         desc,
		Action:     ActionTypeCreateOrMerge,
		Resource:   t.Resource,
		PreAction:  t.PreAction,
		PostAction: t.PostAction,
		Assert:     AssertTypeIsEquals,
	}
	return task.Run(ctx, opts...)
}

// AssertPodListCountTask ensures the observed count of pods matches
// the expected count
type AssertPodListCountTask struct {
	It            string
	ListOptions   []client.ListOption
	ExpectedCount int
}

// compile time check to verify if the structure
// AssertPodListCountTask implements the interface Runner
var _ Runner = (*AssertPodListCountTask)(nil)

func (t *AssertPodListCountTask) Run(ctx context.Context, opts ...RunOption) error {
	pl := &ListingTask{
		It:          t.It,
		Resource:    &corev1.PodList{},
		ListOptions: t.ListOptions,
		PostAction: func(obj client.ObjectList) error {
			podList, _ := obj.(*corev1.PodList)
			if len(podList.Items) != t.ExpectedCount {
				return errors.Errorf("expected %d pod got %d", t.ExpectedCount, len(podList.Items))
			}
			return nil
		},
	}
	return pl.Run(ctx, opts...)
}

// EventualTask is used to run Task till name succeeds or times out
type EventualTask struct {
	Task      Runner
	Interval  *time.Duration
	Timeout   *time.Duration
	Immediate *bool
}

// compile time check to verify if the structure
// EventualTask implements the interface Runner
var _ Runner = (*EventualTask)(nil)

func (t *EventualTask) Run(ctx context.Context, opts ...RunOption) error {
	var (
		interval  = 3 * time.Second
		timeout   = 120 * time.Second
		immediate bool
	)

	if t.Interval != nil {
		interval = *t.Interval
	}
	if t.Timeout != nil {
		timeout = *t.Timeout
	}
	if t.Immediate != nil {
		immediate = *t.Immediate
	}
	rOpts := util.RetryOptions{
		Interval:  interval,
		Timeout:   timeout,
		Immediate: immediate,
	}
	return util.Retry(rOpts, func() (done bool, err error) {
		err = t.Task.Run(ctx, opts...)
		return err == nil, err
	})
}

// FinalizersRemovalTask is a utility task to remove all finalizers
type FinalizersRemovalTask struct {
	Resource client.Object
}

// compile time check to verify if the structure
// FinalizersRemovalTask implements the interface Runner
var _ Runner = (*FinalizersRemovalTask)(nil)

func (t *FinalizersRemovalTask) Run(ctx context.Context, opts ...RunOption) error {
	if t.Resource == nil {
		return nil
	}

	var options RunOptions
	err := ApplyRunOptionsToTarget(&options, opts...)
	if err != nil {
		return err
	}

	var rscheme = options.Scheme
	if rscheme == nil {
		rscheme = scheme.Scheme
	}

	// Since we are not sure of the type of the Resource we
	// build the generic one i.e. unstructured instance
	gvk, err := apiutil.GVKForObject(t.Resource, rscheme)
	if err != nil {
		return errors.Wrap(err, "failed to extract gvk")
	}
	unObj := &unstructured.Unstructured{}
	unObj.SetKind(gvk.Kind)
	unObj.SetAPIVersion(gvk.GroupVersion().String())
	unObj.SetNamespace(t.Resource.GetNamespace())
	unObj.SetName(t.Resource.GetName())

	var isSkipFinalizersRemoval bool

	var steps = Job{
		&Task{
			It:       "should fetch the unstructured resource from cluster if available",
			Action:   ActionTypeGet,
			Resource: unObj,
			PostAction: func(obj client.Object) error {
				if obj == nil {
					isSkipFinalizersRemoval = true
					return nil
				}
				un, _ := obj.(*unstructured.Unstructured)
				if len(un.GetFinalizers()) == 0 {
					isSkipFinalizersRemoval = true
				}
				return nil
			},
		},
		&Task{
			It:       "should remove all finalizers from unstructured resource if any",
			Action:   ActionTypeCreateOrMerge,
			Resource: unObj,
			Skip: func(_ client.Object) (bool, error) {
				return isSkipFinalizersRemoval, nil
			},
			PreAction: func(obj client.Object) error {
				un, _ := obj.(*unstructured.Unstructured)
				un.SetFinalizers([]string{})
				return nil
			},
			PostAction: func(obj client.Object) error {
				un, _ := obj.(*unstructured.Unstructured)
				if len(un.GetFinalizers()) != 0 {
					return errors.Errorf("expected 0 finalizers got %d", len(un.GetFinalizers()))
				}
				return nil
			},
		},
	}
	return errors.Wrap(steps.Run(ctx, opts...), "failed to remove finalizers")
}

// DeletingTask is a utility task to delete a Kubernetes resource
type DeletingTask struct {
	Resource client.Object
}

// compile time check to verify if the structure
// DeletingTask implements the interface Runner
var _ Runner = (*DeletingTask)(nil)

// compile time check to verify if the structure
// DeletingTask implements the interface RegistrarEntry
var _ RegistrarEntry = (*DeletingTask)(nil)

func (t *DeletingTask) Key() Key {
	return Key(k8sutil.ObjKey(t.Resource))
}

func (t *DeletingTask) Type() EntityType {
	return EntityTypeGarbageCollector
}

func (t *DeletingTask) Run(ctx context.Context, opts ...RunOption) error {
	if t.Resource == nil {
		return nil
	}

	var options RunOptions
	err := ApplyRunOptionsToTarget(&options, opts...)
	if err != nil {
		return err
	}

	var rscheme = options.Scheme
	if rscheme == nil {
		rscheme = scheme.Scheme
	}

	// Since we are not sure of the type of the Resource we
	// build the generic one i.e. unstructured instance
	gvk, err := apiutil.GVKForObject(t.Resource, rscheme)
	if err != nil {
		return errors.Wrap(err, "failed to extract gvk")
	}

	unObj := &unstructured.Unstructured{}
	unObj.SetKind(gvk.Kind)
	unObj.SetAPIVersion(gvk.GroupVersion().String())
	unObj.SetNamespace(t.Resource.GetNamespace())
	unObj.SetName(t.Resource.GetName())

	var (
		isSkipResetFinalizers bool
		isSkipDeletion        bool
	)

	var steps = Job{
		&Task{
			It:       "should fetch the resource from cluster if available",
			Action:   ActionTypeGet,
			Resource: unObj,
			PostAction: func(obj client.Object) error {
				if obj == nil {
					isSkipResetFinalizers = true
					isSkipDeletion = true
					return nil
				}
				un, _ := obj.(*unstructured.Unstructured)
				if len(un.GetFinalizers()) == 0 {
					isSkipResetFinalizers = true
				}
				if un.GetDeletionTimestamp() != nil {
					isSkipDeletion = true
				}
				return nil
			},
		},
		&Task{
			It:       "should remove the unstructured resource finalizers if any",
			Action:   ActionTypeCreateOrMerge,
			Resource: unObj,
			Skip: func(_ client.Object) (bool, error) {
				return isSkipResetFinalizers, nil
			},
			PreAction: func(obj client.Object) error {
				un, _ := obj.(*unstructured.Unstructured)
				un.SetFinalizers([]string{})
				return nil
			},
			PostAction: func(obj client.Object) error {
				un, _ := obj.(*unstructured.Unstructured)
				if len(un.GetFinalizers()) != 0 {
					return errors.Errorf("expected 0 finalizers got %d", len(un.GetFinalizers()))
				}
				return nil
			},
		},
		&Task{
			It:       "should delete the unstructured resource if not already",
			Action:   ActionTypeDelete,
			Resource: unObj,
			Skip: func(_ client.Object) (bool, error) {
				return isSkipDeletion, nil
			},
		},
		&EventualTask{
			Task: &Task{
				It:       "should eventually assert absence of the unstructured resource",
				Action:   ActionTypeGet,
				Resource: unObj,
				Assert:   AssertTypeIsNotFound,
			},
		},
	}
	return steps.Run(ctx, opts...)
}

// Teardown deletes the resources that were created by use of
// this package
func Teardown(ctx context.Context, opts ...RunOption) error {
	var result *multierror.Error

	runners := getDefaultGCRegistry().GetRunners()
	count := len(runners)

	// fmt.Printf("==> Teardown\n")

	// delete resources in reverse order of their creation
	for i := count - 1; i >= 0; i-- {
		result = multierror.Append(result, runners[i].Run(ctx, opts...))
	}
	return result.ErrorOrNil()
}
