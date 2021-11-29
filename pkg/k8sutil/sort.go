package k8sutil

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/object"
)

type SortableUnstructureds []*unstructured.Unstructured

var _ sort.Interface = SortableUnstructureds{}

func (a SortableUnstructureds) Len() int      { return len(a) }
func (a SortableUnstructureds) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortableUnstructureds) Less(i, j int) bool {
	first, _ := object.UnstructuredToObjMeta(a[i])
	second, _ := object.UnstructuredToObjMeta(a[j])
	return less(first, second)
}

type SortableMetas []object.ObjMetadata

var _ sort.Interface = SortableMetas{}

func (a SortableMetas) Len() int      { return len(a) }
func (a SortableMetas) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortableMetas) Less(i, j int) bool {
	return less(a[i], a[j])
}

func less(i, j object.ObjMetadata) bool {
	if !Equals(i.GroupKind, j.GroupKind) {
		return IsLessThan(i.GroupKind, j.GroupKind)
	}
	// In case of tie, compare the namespace and name combination so that the output
	// order is consistent irrespective of input order
	if i.Namespace != j.Namespace {
		return i.Namespace < j.Namespace
	}
	return i.Name < j.Name
}

var kind2index = computeKind2index()

func computeKind2index() map[string]int {
	// An attempt to order things to help k8s, e.g.
	// a Service should come before things that refer to it.
	// Namespace should be first.
	// In some cases order just specified to provide determinism.
	orderFirst := []string{
		"CustomResourceDefinition",
		"Namespace",
		"ResourceQuota",
		"StorageClass",
		"ServiceAccount",
		"PodSecurityPolicy",
		"Role",
		"ClusterRole",
		"RoleBinding",
		"ClusterRoleBinding",
		"ConfigMap",
		"Secret",
		"Service",
		"LimitRange",
		"PriorityClass",
		"Deployment",
		"StatefulSet",
		"CronJob",
		"PodDisruptionBudget",
	}
	orderLast := []string{
		"MutatingWebhookConfiguration",
		"ValidatingWebhookConfiguration",
	}
	kind2indexResult := make(map[string]int, len(orderFirst)+len(orderLast))
	for i, n := range orderFirst {
		kind2indexResult[n] = -len(orderFirst) + i
	}
	for i, n := range orderLast {
		kind2indexResult[n] = 1 + i
	}
	return kind2indexResult
}

// getIndexByKind returns the index of the kind respecting the order
func getIndexByKind(kind string) int {
	return kind2index[kind]
}

func Equals(i, j schema.GroupKind) bool {
	return i.Group == j.Group && i.Kind == j.Kind
}

func IsLessThan(i, j schema.GroupKind) bool {
	indexI := getIndexByKind(i.Kind)
	indexJ := getIndexByKind(j.Kind)
	if indexI != indexJ {
		return indexI < indexJ
	}
	if i.Group != j.Group {
		return i.Group < j.Group
	}
	return i.Kind < j.Kind
}
