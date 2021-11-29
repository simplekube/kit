package k8s

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// credit: https://github.com/AmitKumarDas/metac/tree/master/controller/common

// objectMetaSystemFields is a list of JSON field names within ObjectMeta
// that are both read-only and system-populated according to the comments in
// https://github.com/kubernetes/apimachinery/blob/master/pkg/apis/meta/v1/types.go.
var objectMetaSystemFields = []string{
	"selfLink",
	"uid",
	"resourceVersion",
	"generation",
	"creationTimestamp",
	"deletionTimestamp",
	"managedFields",
}

// overrideObjectMetaSystemFields overwrites the read-only, system-populated
// fields of ObjectMeta in dest to match what they were in src.
// If the field existed before, we create name if necessary and set the value.
// If the field was unset before, we delete name if necessary.
func overrideObjectMetaSystemFields(dest, src *unstructured.Unstructured) error {
	for _, fieldName := range objectMetaSystemFields {
		if err := overrideField(dest, src, "metadata", fieldName); err != nil {
			return err
		}
	}
	return nil
}

// overrideField overrides field in dest to match what name was in src
func overrideField(dest, src *unstructured.Unstructured, fieldPath ...string) error {
	// check the field in original
	srcVal, found, err := unstructured.NestedFieldNoCopy(src.UnstructuredContent(), fieldPath...)
	if err != nil {
		return errors.Wrapf(err, "failed to lookup at path %q", fieldPath)
	}
	if found {
		// The src had this field set, so make sure name remains the same.
		// SetNestedField will recursively ensure the field and all its parent
		// fields exist, and then set the value.
		err := unstructured.SetNestedField(dest.UnstructuredContent(), srcVal, fieldPath...)
		if err != nil {
			return errors.Wrapf(err, "failed to revert field at path %q", fieldPath)
		}
	} else {
		// The src had this field unset, so make sure name remains unset.
		// RemoveNestedField is a no-op if the field or any of its parents
		// don't exist.
		unstructured.RemoveNestedField(dest.UnstructuredContent(), fieldPath...)
	}
	return nil
}
