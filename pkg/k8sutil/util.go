package k8sutil

import (
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func IsNilUnstructured(given *unstructured.Unstructured) bool {
	return given == nil || given.Object == nil
}

func MaybeAppendUnstructured(list []*unstructured.Unstructured, add *unstructured.Unstructured) []*unstructured.Unstructured {
	if add == nil || add.Object == nil {
		return list
	}
	return append(list, add)
}

func MaybeAppendUnstructuredList(list []*unstructured.Unstructured, add []*unstructured.Unstructured) []*unstructured.Unstructured {
	for _, a := range add {
		list = MaybeAppendUnstructured(list, a)
	}
	return list
}

func MaybeAppend(list []client.Object, add client.Object) []client.Object {
	if add == nil {
		return list
	}
	return append(list, add)
}

func MaybeAppendList(list []client.Object, add []client.Object) []client.Object {
	for _, a := range add {
		list = MaybeAppend(list, a)
	}
	return list
}

// IsKubernetesObject returns true if the provided unstructured instance
// resembles a Kubernetes schema
func IsKubernetesObject(object *unstructured.Unstructured) bool {
	if object.GetName() == "" || object.GetKind() == "" || object.GetAPIVersion() == "" {
		return false
	}
	return true
}

// EnsureKubernetesObject returns error if the provided unstructured instance
// is not a Kubernetes schema
func EnsureKubernetesObject(object *unstructured.Unstructured) error {
	if !IsKubernetesObject(object) {
		return errors.Errorf("is not a kubernetes object: %s", DescribeObj(object))
	}
	return nil
}

// IsKustomizeObject returns true if the provided unstructured instance
// resembles a Kustomize schema
func IsKustomizeObject(object *unstructured.Unstructured) bool {
	if object.GetKind() == "Kustomization" && object.GroupVersionKind().GroupKind().Group == "kustomize.config.k8s.io" {
		return true
	}
	return false
}

// EnsureKustomizeObject returns error if the provided unstructured instance
// is not a Kustomize schema
func EnsureKustomizeObject(object *unstructured.Unstructured) error {
	if !IsKustomizeObject(object) {
		return errors.Errorf("is not a kustomize object: %s", DescribeObj(object))
	}
	return nil
}

// DescribeObj returns a string format of the provided
// object that may be used for logging purposes
func DescribeObj(obj client.Object) string {
	gvk, _ := apiutil.GVKForObject(obj, scheme.Scheme)
	return fmt.Sprintf("ns=%s: name=%s: %s", obj.GetNamespace(), obj.GetName(), gvk)
}

// ObjKey returns a string that can be used as a key
// to store objects of type client.Object
func ObjKey(obj client.Object) string {
	gvk, _ := apiutil.GVKForObject(obj, scheme.Scheme)
	return fmt.Sprintf("%s:%s:%s", obj.GetNamespace(), obj.GetName(), gvk)
}

// ToTyped transforms the provided unstructured instance
// to dest instance
func ToTyped(src *unstructured.Unstructured, dest interface{}) error {
	if src == nil || src.Object == nil {
		return errors.Errorf(
			"Can't transform to typed: Nil src",
		)
	}
	if dest == nil {
		return errors.Errorf(
			"Can't transform to typed: Nil dest",
		)
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(
		src.UnstructuredContent(),
		dest,
	)
}

// ToUnstructured transforms the provided object instance
// to unstructured
func ToUnstructured(src metav1.Object, dest *unstructured.Unstructured) error {
	if src == nil {
		return errors.Errorf(
			"Can't transform to unstructured: Nil src",
		)
	}
	if dest == nil {
		return errors.Errorf(
			"Can't transform to unstructured: Nil dest",
		)
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(src)
	if err != nil {
		return err
	}
	dest.Object = obj
	return nil
}
