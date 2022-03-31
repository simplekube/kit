package k8s

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/simplekube/kit/pkg/apply"
	"github.com/simplekube/kit/pkg/k8sutil"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetKindVersionForObject(object client.Object, rscheme *runtime.Scheme) (kind string, version string, err error) {
	gvk, err := apiutil.GVKForObject(object, rscheme)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to extract gvk")
	}

	return gvk.Kind, gvk.GroupVersion().String(), nil
}

// common run options that is used as a base before invoking various operations
//
// Note: This can be overridden if specific options are provided
// during the function invocation
var _baseRunOptions *RunOptions = &RunOptions{}
var _baseRunOptionsOnce sync.Once
var _isBaseRunOptionsRegistered bool

// RegisterBaseRunOptions is used to set default or common
// run options once instead of specifying them repeatedly
// across each function invocations
func RegisterBaseRunOptions(options *RunOptions) error {
	if options == nil {
		return errors.New("nil base run options")
	}
	if _isBaseRunOptionsRegistered {
		return errors.New("base run options already registered")
	}
	_baseRunOptionsOnce.Do(func() {
		_baseRunOptions = options
		_isBaseRunOptionsRegistered = true
	})
	return nil
}

func makeRunOptionsWithBase(options ...RunOption) (*RunOptions, error) {
	var opts = []RunOption{_baseRunOptions}
	return FromRunOptions(append(opts, options...)...)
}

func maybeSetRunOptionsWithDefaults(options *RunOptions) error {
	// ensure Kubernetes client is set
	if options.Client == nil {
		config := config.GetConfigOrDie()
		c, err := client.New(config, client.Options{})
		if err != nil {
			return errors.Wrap(err, "failed to initialise client")
		}
		options.Client = c
	}

	// ensure Kubernetes scheme is set
	if options.Scheme == nil {
		// default to the scheme that understands all native Kubernetes API schemas
		options.Scheme = scheme.Scheme
	}
	return nil
}

func makeRunOptions(options ...RunOption) (*RunOptions, error) {
	opts, err := makeRunOptionsWithBase(options...)
	if err != nil {
		return nil, err
	}
	err = maybeSetRunOptionsWithDefaults(opts)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

type InvokeFn func(ctx context.Context, object client.Object, options ...RunOption) (client.Object, error)

func InvokeOperationForAllObjects(ctx context.Context, operation InvokeFn, objects []client.Object, options ...RunOption) ([]client.Object, error) {
	var kObjs []client.Object
	var finalError error
	for _, obj := range objects {
		got, err := operation(ctx, obj, options...)
		if err != nil {
			finalError = multierror.Append(finalError, err)
			continue
		}
		kObjs = append(kObjs, got)
	}
	return kObjs, finalError
}

// InvokeOperationForAllYAMLs executes the passed function against
// the provided file paths
func InvokeOperationForAllYAMLs(ctx context.Context, operation InvokeFn, filePaths []string, options ...RunOption) ([]client.Object, error) {
	objs, err := k8sutil.BuildObjectsFromYMLs(filePaths)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, errors.Errorf("no unstructured objects found: %q", filePaths)
	}

	var cObjs = make([]client.Object, 0, len(objs))
	for _, obj := range objs {
		if !k8sutil.IsNilUnstructured(obj) {
			cObjs = append(cObjs, obj)
		}
	}
	if len(cObjs) == 0 {
		return nil, errors.Errorf("no kubernetes objects found: %q", filePaths)
	}
	return InvokeOperationForAllObjects(ctx, operation, cObjs, options...)
}

// InvokeOperationForYAML executes the passed function against
// the provided file path
func InvokeOperationForYAML(ctx context.Context, operation InvokeFn, filePath string, options ...RunOption) (kObj client.Object, err error) {
	kObjs, err := InvokeOperationForAllYAMLs(ctx, operation, []string{filePath}, options...)
	if err != nil {
		return nil, err
	}
	if len(kObjs) > 0 {
		kObj = kObjs[0]
	}
	return kObj, nil
}

func Get(ctx context.Context, given client.Object, options ...RunOption) (client.Object, error) {
	opts, err := makeRunOptions(options...)
	if err != nil {
		return nil, err
	}
	if given == nil {
		return nil, errors.New("nil object")
	}
	actual, _ := given.DeepCopyObject().(client.Object)
	err = opts.Client.Get(ctx, client.ObjectKeyFromObject(given), actual)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get")
	}
	return actual, nil
}

func GetAll(ctx context.Context, given []client.Object, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllObjects(ctx, Get, given, options...)
}

func GetForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllYAMLs(ctx, Get, filePaths, options...)
}

func GetForYAML(ctx context.Context, filePath string, options ...RunOption) (client.Object, error) {
	return InvokeOperationForYAML(ctx, Get, filePath, options...)
}

func Create(ctx context.Context, given client.Object, options ...RunOption) (client.Object, error) {
	opts, err := makeRunOptions(options...)
	if err != nil {
		return nil, err
	}
	if given == nil {
		return nil, errors.New("nil object")
	}
	actual, _ := given.DeepCopyObject().(client.Object)
	err = opts.Client.Create(ctx, actual)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create")
	}
	return actual, nil
}

func CreateAll(ctx context.Context, given []client.Object, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllObjects(ctx, Create, given, options...)
}

func CreateForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllYAMLs(ctx, Create, filePaths, options...)
}

func CreateForYAML(ctx context.Context, filePath string, options ...RunOption) (kObj client.Object, err error) {
	return InvokeOperationForYAML(ctx, Create, filePath, options...)
}

func Update(ctx context.Context, given client.Object, options ...RunOption) (client.Object, error) {
	opts, err := makeRunOptions(options...)
	if err != nil {
		return nil, err
	}
	if given == nil {
		return nil, errors.New("nil object")
	}
	actual, _ := given.DeepCopyObject().(client.Object)
	err = opts.Client.Update(ctx, actual)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update")
	}
	return actual, nil
}

func UpdateAll(ctx context.Context, given []client.Object, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllObjects(ctx, Update, given, options...)
}

func UpdateForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllYAMLs(ctx, Update, filePaths, options...)
}

func UpdateForYAML(ctx context.Context, filePath string, options ...RunOption) (kObj client.Object, err error) {
	return InvokeOperationForYAML(ctx, Update, filePath, options...)
}

func Delete(ctx context.Context, given client.Object, options ...RunOption) error {
	opts, err := makeRunOptions(options...)
	if err != nil {
		return err
	}
	if given == nil {
		return errors.New("nil object")
	}
	return opts.Client.Delete(ctx, given)
}

// DeleteWrapper invokes delete operation & ensures its signature
// matches with other invocations like Get, Create, Update, Apply & DryRun.
func DeleteWrapper(ctx context.Context, given client.Object, options ...RunOption) (_ client.Object, err error) {
	return nil, Delete(ctx, given, options...)
}

func DeleteAll(ctx context.Context, given []client.Object, options ...RunOption) error {
	_, err := InvokeOperationForAllObjects(ctx, DeleteWrapper, given, options...)
	return err
}

func DeleteForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) error {
	_, err := InvokeOperationForAllYAMLs(ctx, DeleteWrapper, filePaths, options...)
	return err
}

func DeleteForYAML(ctx context.Context, filePath string, options ...RunOption) error {
	_, err := InvokeOperationForYAML(ctx, DeleteWrapper, filePath, options...)
	return err
}

func Apply(ctx context.Context, given client.Object, options ...RunOption) (client.Object, error) {
	opts, err := makeRunOptions(options...)
	if err != nil {
		return nil, err
	}
	if given == nil {
		return nil, errors.New("nil object")
	}
	patchOpts := []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("k8s-toolkit-operation"),
	}
	actual, _ := given.DeepCopyObject().(client.Object)
	err = opts.Client.Patch(ctx, actual, client.Apply, patchOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply")
	}
	return actual, nil
}

func ApplyAll(ctx context.Context, given []client.Object, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllObjects(ctx, Apply, given, options...)
}

func ApplyAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllYAMLs(ctx, Apply, filePaths, options...)
}

func ApplyYAML(ctx context.Context, filePath string, options ...RunOption) (kObj client.Object, err error) {
	return InvokeOperationForYAML(ctx, Apply, filePath, options...)
}

// DryRun executes a ServerSideApply DryRun invocation
//
// Note: Given object should have its metadata.managedFields set to nil
func DryRun(ctx context.Context, given client.Object, options ...RunOption) (client.Object, error) {
	opts, err := makeRunOptions(options...)
	if err != nil {
		return nil, err
	}
	if given == nil {
		return nil, errors.New("nil object")
	}
	kind, version, err := GetKindVersionForObject(given, opts.Scheme)
	if err != nil {
		return nil, err
	}

	// Build an unstructured instance from the given instance
	// This is needed to execute DryRun API that expects APIVersion
	// & Kind to be set explicitly
	un, err := runtime.DefaultUnstructuredConverter.ToUnstructured(given)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert client.Object to unstructured")
	}
	unObj := &unstructured.Unstructured{Object: un}
	dryRunObj := unObj.DeepCopy()
	dryRunObj.SetKind(kind)
	dryRunObj.SetAPIVersion(version)

	patchOpts := []client.PatchOption{
		client.DryRunAll,
		client.ForceOwnership,
		client.FieldOwner("k8s-toolkit-ops"),
	}
	err = opts.Client.Patch(ctx, dryRunObj, client.Apply, patchOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dry run")
	}

	// Convert the updated unstructured instance to client.Object type
	actual, _ := given.DeepCopyObject().(client.Object)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(dryRunObj.Object, actual)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert unstructured to client.Object")
	}
	return actual, nil
}

func DryRunAll(ctx context.Context, given []client.Object, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllObjects(ctx, DryRun, given, options...)
}

func DryRunAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) ([]client.Object, error) {
	return InvokeOperationForAllYAMLs(ctx, DryRun, filePaths, options...)
}

func DryRunYAML(ctx context.Context, filePath string, options ...RunOption) (kObj client.Object, err error) {
	return InvokeOperationForYAML(ctx, DryRun, filePath, options...)
}

// HasDrifted returns true if given object differs from the object observed
// in the cluster
//
// Note:
// - Object states comparison is a server side implementation i.e. Kubernetes
// APIs are invoked to determine the comparison result
func HasDrifted(ctx context.Context, given client.Object, options ...RunOption) (isDrift bool, drift string, err error) {
	observedObj, err := Get(ctx, given)
	if err != nil {
		return false, "", err
	}

	driftedObj, err := DryRun(ctx, given)
	if err != nil {
		return false, "", err
	}

	isEqual, diff, err := IsEqualWithDiffOutput(observedObj, driftedObj)
	return !isEqual, diff, err
}

type AssertOptions struct {
	AssertType     AssertType
	CustomAssertFn func(actual, expected client.Object) (result bool, diff string, err error)
}

// Assert returns true if assertion matches the expectation
//
// Note: The return value `diff` provides the difference between the actual vs
// desired states
func Assert(ctx context.Context, expected client.Object, assertOptions AssertOptions, options ...RunOption) (result bool, diff string, err error) {
	actual, err := Get(ctx, expected, options...)
	if err != nil {
		return
	}

	switch assertOptions.AssertType {
	case AssertTypeIsEquals:
		result, diff, err = IsEqualWithDiffOutput(actual, expected)
	case AssertTypeIsNotEquals:
		result, diff, err = IsEqualWithDiffOutput(actual, expected)
		result = !result // invert assert result
	case AssertTypeIsNotFound:
		if actual == nil {
			result = true // assert succeeded
		} else {
			diff = "found a resource while expecting none"
		}
	case AssertTypeIsFound:
		if actual != nil {
			result = true // assert succeeded
		} else {
			diff = "resource was not found while expecting one"
		}
	case AssertTypeIsCustom:
		if assertOptions.CustomAssertFn != nil {
			return assertOptions.CustomAssertFn(actual, expected)
		}
		err = errors.New("missing custom assert function")
	default:
		err = errors.Errorf("un-supported assert type %q", assertOptions.AssertType)
	}

	return result, diff, err
}

func AssertEquals(ctx context.Context, expected client.Object, options ...RunOption) (result bool, diff string, err error) {
	return Assert(ctx, expected, AssertOptions{AssertType: AssertTypeIsEquals}, options...)
}

func AssertNotEquals(ctx context.Context, given client.Object, options ...RunOption) (result bool, diff string, err error) {
	return Assert(ctx, given, AssertOptions{AssertType: AssertTypeIsNotEquals}, options...)
}

func AssertIsFound(ctx context.Context, given client.Object, options ...RunOption) (result bool, diff string, err error) {
	return Assert(ctx, given, AssertOptions{AssertType: AssertTypeIsFound}, options...)
}

func AssertIsNotFound(ctx context.Context, given client.Object, options ...RunOption) (result bool, diff string, err error) {
	return Assert(ctx, given, AssertOptions{AssertType: AssertTypeIsNotFound}, options...)
}

func AssertAllYAMLs(ctx context.Context, filePaths []string, assertOptions AssertOptions, options ...RunOption) (result bool, diffs []string, err error) {
	objs, err := k8sutil.BuildObjectsFromYMLs(filePaths)
	if err != nil {
		return false, nil, err
	}

	var finalError *multierror.Error
	result = true
	for _, obj := range objs {
		assertResult, diff, err := Assert(ctx, obj, assertOptions, options...)
		if err != nil {
			finalError = multierror.Append(finalError.ErrorOrNil(), err)
			result = false
			continue
		}
		result = result && assertResult
		diffs = append(diffs, diff)
	}
	return result, diffs, finalError.ErrorOrNil()
}

func AssertYAML(ctx context.Context, filePath string, assertOptions AssertOptions, options ...RunOption) (result bool, diff string, err error) {
	result, diffs, err := AssertAllYAMLs(ctx, []string{filePath}, assertOptions, options...)
	if len(diffs) > 0 {
		diff = diffs[0]
	}
	return result, diff, err
}

func AssertEqualsForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) (result bool, diffs []string, err error) {
	return AssertAllYAMLs(ctx, filePaths, AssertOptions{AssertType: AssertTypeIsEquals}, options...)
}

func AssertEqualsForYAML(ctx context.Context, filePath string, options ...RunOption) (result bool, diff string, err error) {
	return AssertYAML(ctx, filePath, AssertOptions{AssertType: AssertTypeIsEquals}, options...)
}

func AssertIsFoundForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) (result bool, diffs []string, err error) {
	return AssertAllYAMLs(ctx, filePaths, AssertOptions{AssertType: AssertTypeIsFound}, options...)
}

func AssertIsFoundForYAML(ctx context.Context, filePath string, options ...RunOption) (result bool, diff string, err error) {
	return AssertYAML(ctx, filePath, AssertOptions{AssertType: AssertTypeIsFound}, options...)
}

func AssertIsNotFoundForAllYAMLs(ctx context.Context, filePaths []string, options ...RunOption) (result bool, diffs []string, err error) {
	return AssertAllYAMLs(ctx, filePaths, AssertOptions{AssertType: AssertTypeIsNotFound}, options...)
}

func AssertIsNotFoundForYAML(ctx context.Context, filePath string, options ...RunOption) (result bool, diff string, err error) {
	return AssertYAML(ctx, filePath, AssertOptions{AssertType: AssertTypeIsNotFound}, options...)
}

// OperationResult is the action result of a CreateOrUpdate call.
//
// credit: https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/controller/controllerutil
type OperationResult string

const (
	// OperationResultNone implies that the resource was not changed
	OperationResultNone OperationResult = "unchanged"

	// OperationResultCreated implies that a new resource got created
	OperationResultCreated OperationResult = "created"

	// OperationResultUpdatedResourceOnly implies that an existing resource got updated
	OperationResultUpdatedResourceOnly OperationResult = "updated-resource-only"

	// OperationResultUpdatedResourceAndStatus implies an existing resource as well as its status got updated
	OperationResultUpdatedResourceAndStatus OperationResult = "updated-resource-and-status"

	// OperationResultUpdatedStatusOnly implies that only an existing status got updated
	OperationResultUpdatedStatusOnly OperationResult = "updated-status-only"
)

type EventuallyOptions struct {
	RetryInterval    time.Duration
	RetryTimeout     time.Duration
	RetryOnErrorOnly bool
	RetryOnErrorType error
}

// CreateOrMerge creates or merges the desired object in the Kubernetes
// cluster. The desired state is merged into the observed state found
// in the cluster.
func CreateOrMerge(ctx context.Context, cli client.Client, scheme *runtime.Scheme, desired client.Object) (OperationResult, error) {
	result, err := createOrMerge(ctx, cli, scheme, desired)
	if err == nil {
		// this will get latest observed instance found in cluster
		// & update against the provided desired instance
		//
		// Note: error if any is ignored
		_ = cli.Get(ctx, client.ObjectKeyFromObject(desired), desired)
	}
	return result, err
}

func createOrMerge(ctx context.Context, cli client.Client, scheme *runtime.Scheme, desired client.Object) (OperationResult, error) {
	if cli == nil {
		return OperationResultNone, errors.New("nil client")
	}
	if desired == nil {
		return OperationResultNone, errors.New("nil desired object")
	}
	gvk, err := apiutil.GVKForObject(desired, scheme)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to extract gvk")
	}

	// build the observed instance
	// Note: This instance will be filled with the one found
	// in the cluster if available
	observed := &unstructured.Unstructured{}
	observed.SetKind(gvk.Kind)
	observed.SetAPIVersion(gvk.GroupVersion().String())
	observed.SetNamespace(desired.GetNamespace())
	observed.SetName(desired.GetName())

	if err := cli.Get(ctx, client.ObjectKeyFromObject(desired), observed); err != nil {
		if !apierrors.IsNotFound(err) {
			return OperationResultNone, errors.Wrap(err, "failed to get resource")
		}
		// Note: Create will update the server content into the desired object
		if err := cli.Create(ctx, desired); err != nil {
			return OperationResultNone, errors.Wrap(err, "failed to create resource")
		}
		return OperationResultCreated, nil
	}

	observedUnstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(observed)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to convert observed state to unstructured")
	}
	desiredUnstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired.DeepCopyObject())
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to convert desired state to unstructured")
	}

	// remove null type entries from the desired instance
	//
	// Note: Not doing so creates unneeded diffs between
	// merged & observed instances
	desiredUnstruct, err = DeleteNullInUnstructuredMap(desiredUnstruct)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to remove null from desired state")
	}

	// three-way client side merge of desired into observed
	mergedUnstruct, err := ThreeWayLocalMergeWithTwoObjects(observedUnstruct, desiredUnstruct)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to merge locally desired state to observed state")
	}

	// start dealing with Kubernetes defined unstructured instances
	// This is required to make K8s API calls later
	var mergedObj, observedObj unstructured.Unstructured
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(mergedUnstruct, &mergedObj)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to create merged object from unstructured")
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(observedUnstruct, &observedObj)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to create observed object from unstructured")
	}

	// Handle metadata system fields i.e. read-only fields by setting
	// them in the merged object from the observed object. Eliminating
	// read only fields enables comparing observed & merged object without
	// bothering about the system generated values in observed object.
	//
	// Note: This override is needed even with three-way merge strategy.
	// This is done since serialization can result in automatic setting
	// of default values e.g. nil, etc. against the desired state which
	// results in a difference when desired is compared against observed.
	//
	// Note: This also handles setting the resourceVersion field in the merged
	// object which in turn is mandatory for subsequent update call
	overrideObjectMetaSystemFields(&mergedObj, &observedObj)
	// fmt.Printf("==> diff: -observed +merged\n%s\n", cmp.Diff(observedObj, mergedObj))
	if equality.Semantic.DeepEqual(&observedObj, &mergedObj) {
		// return if there is no change
		// fmt.Printf("==> no diff\n")
		return OperationResultNone, nil
	}
	// fmt.Printf("==> has diff\n")

	// copy the merged object for status update call
	var mergedStatusObj = *mergedObj.DeepCopy()

	// update resource
	err = cli.Update(ctx, &mergedObj)
	if err != nil {
		return OperationResultNone, errors.Wrap(err, "failed to update to desired state")
	}

	hasStatus, err := IsStatusSubResourceSet(desiredUnstruct)
	if err != nil || !hasStatus {
		return OperationResultUpdatedResourceOnly, errors.Wrap(err, "failed to verify presence of resource status")
	}

	// update resource version before proceeding with status update
	mergedStatusObj.SetResourceVersion(mergedObj.GetResourceVersion())
	// update resource status
	err = cli.Status().Update(ctx, &mergedStatusObj)
	if err != nil {
		return OperationResultUpdatedResourceOnly, errors.Wrap(err, "failed to update to desired status")
	}

	return OperationResultUpdatedResourceAndStatus, nil
}

func IsStatusSubResourceSet(obj map[string]interface{}) (bool, error) {
	status, found, err := unstructured.NestedFieldCopy(obj, "status")
	if !found || status == nil || err != nil {
		return false, err
	}

	value := reflect.ValueOf(status)
	if value.IsNil() || value.Len() == 0 {
		return false, nil
	}

	return true, nil
}

// ThreeWayLocalMerge represents a three-way client side merge
func ThreeWayLocalMerge(observed, lastApplied, desired map[string]interface{}) (map[string]interface{}, error) {
	return apply.Merge(observed, lastApplied, desired)
}

// ThreeWayLocalMergeWithTwoObjects represents a three-way client side merge
func ThreeWayLocalMergeWithTwoObjects(observed, desired map[string]interface{}) (map[string]interface{}, error) {
	return ThreeWayLocalMerge(observed, runtime.DeepCopyJSON(desired), desired)
}

// ToComparableObjects merges the provided desired state with the
// provided observed state to form a merged state. As the function name
// suggests, this is useful before running DeepEqual check.
//
// Note:
// - Merge is done on the basis of fields present in the desired object
// - Merge is purely a client side implementation i.e. Kubernetes APIs
// are not involved in the process
// - Merged state differs from the observed state if the desired state is not
// a subset of the observed state.
// - Merged state takes care of Kubernetes read only system fields by copying
// them from the observed state into the merged state
func ToComparableObjects(observed, desired client.Object) (observedObj, mergedObj *unstructured.Unstructured, err error) {
	if observed == nil {
		return nil, nil, errors.New("nil observed")
	}
	if desired == nil {
		return nil, nil, errors.New("nil desired")
	}
	observedUnstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(observed.DeepCopyObject())
	if err != nil {
		return nil, nil, errors.Wrap(err, "convert observed to unstructured")
	}
	desiredUnstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired.DeepCopyObject())
	if err != nil {
		return nil, nil, errors.Wrap(err, "convert desired to unstructured")
	}

	// Remove null entries from the desired instance
	//
	// Note: Not doing so creates false diffs between
	// merged & observed instances
	desiredUnstruct, err = DeleteNullInUnstructuredMap(desiredUnstruct)
	if err != nil {
		return nil, nil, errors.Wrap(err, "remove null from desired")
	}

	// 3-way client side merge of desired & observed to derive the merged state
	mergedUnstruct, err := ThreeWayLocalMergeWithTwoObjects(observedUnstruct, desiredUnstruct)
	if err != nil {
		return nil, nil, err
	}

	// var mergedObj, observedObj unstructured.Unstructured
	observedObj = &unstructured.Unstructured{}
	mergedObj = &unstructured.Unstructured{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(mergedUnstruct, mergedObj)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create merged from unstructured")
	}

	// Set TypeMeta info against observed instance since they are missing in
	// observed instance
	observedUnstruct["kind"] = mergedObj.GetKind()
	observedUnstruct["apiVersion"] = mergedObj.GetAPIVersion()
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(observedUnstruct, observedObj)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create observed from unstructured")
	}

	// Ensure read-only system fields are copied into merged instance
	// from observed instance. This removes false diffs while comparing
	// these two instance
	//
	// Note: Observed instance i.e. the state found in Kubernetes cluster,
	// is assumed to have these system fields
	overrideObjectMetaSystemFields(mergedObj, observedObj)
	return observedObj, mergedObj, nil
}

// IsEqualWithMergeOutput matches any Kubernetes resource for equality. A
// match is found if desired object's fields matches the corresponding fields
// of observed object. Desired object's field values may be an exact match or
// may be a subset of corresponding values found in observed object.
//
// Note:
// - Comparison is done on the basis of fields present in the desired object
// - Comparison is purely a client side implementation i.e. Kubernetes APIs
// are not involved in the process
func IsEqualWithMergeOutput(observed, desired client.Object) (bool, *unstructured.Unstructured, error) {
	observedObj, mergedObj, err := ToComparableObjects(observed, desired)
	if err != nil {
		return false, nil, err
	}

	return equality.Semantic.DeepEqual(observedObj, mergedObj), mergedObj, nil
}

// IsEqualWithDiffOutput matches any Kubernetes resource for equality. A
// match is found if desired object's fields matches the corresponding fields
// of observed object. Desired object's field values may be an exact match or
// may be a subset of corresponding values found in observed object.
//
// Note:
// - Comparison is done on the basis of fields present in the desired object
// - Comparison is purely a client side implementation i.e. Kubernetes APIs
// are not involved in the process
// - Diff response is formatted as -observed +merged
func IsEqualWithDiffOutput(observed, desired client.Object) (bool, string, error) {
	observedObj, mergedObj, err := ToComparableObjects(observed, desired)
	if err != nil {
		return false, "", err
	}

	return equality.Semantic.DeepEqual(observedObj, mergedObj), cmp.Diff(observedObj, mergedObj), nil
}

// IsEqual matches any Kubernetes resource for equality. A match is found
// if desired object's fields matches the corresponding fields of observed
// object. Desired object's field values may be an exact match or may be
// a subset of corresponding values found in observed object.
//
// Note:
// - Comparison is done on the basis of fields present in the desired object
// - Comparison is purely a client side implementation i.e. Kubernetes APIs
// are not involved in the process
func IsEqual(observed, desired client.Object) (bool, error) {
	isEqual, _, err := IsEqualWithMergeOutput(observed, desired)
	if err != nil {
		return false, err
	}

	return isEqual, nil
}

// IsEqualOrDie executes IsEqual with an additional task of suspending
// the observed thread in case of any runtime error
func IsEqualOrDie(observed, desired client.Object) bool {
	b, err := IsEqual(observed, desired)
	if err != nil {
		panic(err)
	}
	return b
}
