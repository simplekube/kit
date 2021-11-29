// Package apply is a client-side substitute for `Kubernetes server side apply`.
// It tries to guess the right thing to do without any type-specific knowledge.
// Instead of generating a PATCH request, it does the patching locally and
// returns a full object with the ResourceVersion intact.
//
// credit: https://github.com/AmitKumarDas/metac/tree/master/dynamic/apply
package apply

import (
	"fmt"

	"github.com/simplekube/kit/pkg/k8sutil"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	lastAppliedAnnotation = "kit.simplekube.github.com/last-applied-state"
)

// SetLastApplied sets the last applied state against a
// predefined annotation key
func SetLastApplied(obj *unstructured.Unstructured, lastApplied map[string]interface{}) error {
	return SetLastAppliedByAnnKey(obj, lastApplied, lastAppliedAnnotation)
}

// SetLastAppliedByAnnKey sets the last applied state against the
// provided annotation key
func SetLastAppliedByAnnKey(
	obj *unstructured.Unstructured,
	lastApplied map[string]interface{},
	annKey string,
) error {
	if len(lastApplied) == 0 {
		return nil
	}

	lastAppliedJSON, err := json.Marshal(lastApplied)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to marshal last applied state: %s: annotation %s",
			k8sutil.DescribeObj(obj),
			annKey,
		)
	}

	ann := obj.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string, 1)
	}
	ann[annKey] = string(lastAppliedJSON)
	obj.SetAnnotations(ann)

	return nil
}

// SanitizeLastAppliedByAnnKey sanitizes the last applied state
// by removing last applied state related info (i.e. its own info)
// to avoid building up of nested last applied states.
//
// In other words, last applied state might have an annotation that stores
// the previous last applied state which in turn might have an annotation
// that stores the previous to previous last applied state. This nested
// annotation gets added everytime a reconcile event happens for the
// resource that needs to be applied against Kubernetes cluster.
func SanitizeLastAppliedByAnnKey(last map[string]interface{}, annKey string) {
	if len(last) == 0 {
		return
	}
	unstructured.RemoveNestedField(last, "metadata", "annotations", annKey)
}

// GetLastApplied returns the last applied state of the given
// object. Last applied state is derived based on a predefined
// annotation key to store last applied state.
func GetLastApplied(obj *unstructured.Unstructured) (map[string]interface{}, error) {
	return GetLastAppliedByAnnKey(obj, lastAppliedAnnotation)
}

// GetLastAppliedByAnnKey returns the last applied state of the given
// object based on the provided annotation
func GetLastAppliedByAnnKey(
	obj *unstructured.Unstructured, annKey string,
) (map[string]interface{}, error) {

	lastAppliedJSON := obj.GetAnnotations()[annKey]
	if lastAppliedJSON == "" {
		return nil, nil
	}

	lastApplied := make(map[string]interface{})
	err := json.Unmarshal([]byte(lastAppliedJSON), &lastApplied)
	return lastApplied, errors.Wrapf(
		err,
		"failed to unmarshal last applied state: %s : annotation %s",
		k8sutil.DescribeObj(obj),
		annKey,
	)
}

// Merge updates the observed object with the desired changes.
// Merge is based on a 3-way apply that takes in observed state,
// last applied state & desired state into consideration.
func Merge(observed, lastApplied, desired map[string]interface{}) (map[string]interface{}, error) {
	// Make a copy of observed & use it as the destination for final merged state
	observedAsDest := runtime.DeepCopyJSON(observed)

	if _, err := mergeToObserved("", observedAsDest, lastApplied, desired); err != nil {
		return nil, errors.Wrapf(err, "failed to merge desired state")
	}
	return observedAsDest, nil
}

func mergeToObserved(fieldPath string, observed, lastApplied, desired interface{}) (interface{}, error) {
	switch observedVal := observed.(type) {
	case map[string]interface{}:
		// In this case, observed is a **map**.
		// Make sure the others are maps too.
		// Nil desired &/ nil last applied are OK.
		lastAppliedVal, ok := lastApplied.(map[string]interface{})
		if !ok && lastAppliedVal != nil {
			return nil,
				errors.Errorf(
					"type mismatch: observed state %T: last applied state %T: field %q",
					observed, lastApplied, fieldPath,
				)
		}
		desiredVal, ok := desired.(map[string]interface{})
		if !ok && desiredVal != nil {
			return nil,
				errors.Errorf(
					"type mismatch: observed state %T: desired state %T: field %q",
					observed, desired, fieldPath,
				)
		}
		return mergeMapToObserved(fieldPath, observedVal, lastAppliedVal, desiredVal)
	case []interface{}:
		// In this case observed is an **array**.
		// Make sure desired & last applied are arrays too.
		// Nil desired &/ last applied are OK.
		lastAppliedVal, ok := lastApplied.([]interface{})
		if !ok && lastAppliedVal != nil {
			return nil,
				errors.Errorf(
					"type mismatch: observed state %T: last applied state %T: field %q",
					observed, lastApplied, fieldPath,
				)
		}
		desiredVal, ok := desired.([]interface{})
		if !ok && desiredVal != nil {
			return nil,
				fmt.Errorf(
					"type mismatch: observed state %T: desired state %T: field %q",
					observed, desired, fieldPath,
				)
		}
		return mergeArrayToObserved(fieldPath, observedVal, lastAppliedVal, desiredVal)
	default:
		// Observed is either a **scalar** or **null**.
		//
		// NOTE:
		// 	We have traversed to the leaf of the object. There is no further
		// traversal that needs to be done. At this point desired value is the
		// final merge value.
		//
		// NOTE:
		//	Since merge method is being called recursively, this point signals
		// end of last recursion
		return desired, nil
	}
}

func mergeMapToObserved(fieldPath string, observed, lastApplied, desired map[string]interface{}) (interface{}, error) {
	// Remove fields that were present in lastApplied, but no longer
	// in desired. In other words, this decision to delete a field
	// is based on last applied state.
	//
	// NOTE:
	//	If there is no last applied then there will be **no** removals
	for key := range lastApplied {
		if _, present := desired[key]; !present {
			delete(observed, key)
		}
	}

	// Once deletion is done try adding or updating fields
	//
	// NOTE:
	//	If there is no desired state i.e. nil, then there will be
	// no add or update
	var err error
	for key, desiredVal := range desired {
		// destination is mutated here either as an add or update map operation
		nestedPath := fmt.Sprintf("%s[%s]", fieldPath, key)
		observed[key], err = mergeToObserved(nestedPath, observed[key], lastApplied[key], desiredVal)
		if err != nil {
			return nil, err
		}
	}

	// NOTE:
	//	If there is nil last applied state & nil desired state then
	// observed map will be returned
	return observed, nil
}

func mergeArrayToObserved(fieldPath string, observed, lastApplied, desired []interface{}) (interface{}, error) {
	// If it looks like a list of map, use the special mergeListMapToObserved
	// by determining the best possible **merge key**
	if mergeKey := detectListMapKey(observed, lastApplied, desired); mergeKey != "" {
		return mergeListMapToObserved(fieldPath, mergeKey, observed, lastApplied, desired)
	}

	// It's a normal array of scalars.
	// Hence, consider the desired array.
	//
	// For Example: metadata.finalizers is considered as a normal array
	// of scalars since it is of type '[]string'
	//
	// TODO(enisoc / amit.das): Check if there are any common cases where we
	// want to merge. E.g. should finalizers receive a special treatment?
	return desired, nil
}

func mergeListMapToObserved(fieldPath, mergeKey string, observed, lastApplied, desired []interface{}) (interface{}, error) {
	// transform the lists to corresponding maps, keyed by the mergeKey field
	observedMap := makeMapFromList(mergeKey, observed)
	lastAppliedMap := makeMapFromList(mergeKey, lastApplied)
	desiredMap := makeMapFromList(mergeKey, desired)

	// once in map, try map based merge
	_, err := mergeMapToObserved(fieldPath, observedMap, lastAppliedMap, desiredMap)
	if err != nil {
		return nil, err
	}

	// Turn merged map back into a list, trying to preserve order
	//
	// NOTE:
	//	In most of the cases, this ordering is more than sufficient.
	// This ordering helps in negating the diff found between two
	// lists each with same items but with different order.
	observedList := make([]interface{}, 0, len(observedMap))
	added := make(map[string]bool, len(observedMap))

	// First take items that were already in destination.
	// This helps in maintaining the order that was found before
	// the merge operation.
	for _, item := range observed {
		valueAsKey := stringMergeKey(item.(map[string]interface{})[mergeKey])
		if mergedMap, ok := observedMap[valueAsKey]; ok {
			observedList = append(observedList, mergedMap)
			// Remember which items we've already added to the final list.
			added[valueAsKey] = true
		}
	}
	// Then take items in desired that haven't been added yet.
	//
	// NOTE:
	//	This handles the case of newly added items in the desried
	// state. These items won't be present in observed or last applied
	// states.
	for _, item := range desired {
		valueAsKey := stringMergeKey(item.(map[string]interface{})[mergeKey])
		if !added[valueAsKey] {
			// append it since it is not available in the final list
			observedList = append(observedList, observedMap[valueAsKey])
			added[valueAsKey] = true
		}
	}

	return observedList, nil
}

func makeMapFromList(mergeKey string, list []interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(list))
	for _, item := range list {
		// We only end up here if detectListMapKey() already verified that
		// all items are of type map
		itemMap := item.(map[string]interface{})
		result[stringMergeKey(itemMap[mergeKey])] = item
	}
	return result
}

// stringMergeKey converts the provided value that is not of type
// string to string
func stringMergeKey(val interface{}) string {
	switch tval := val.(type) {
	case string:
		return tval
	default:
		return fmt.Sprintf("%v", val)
	}
}

// knownMergeKeys lists the key names we will guess as merge keys
//
// The order determines precedence if multiple entries are found.
// The first occurring item has the highest precedence & is used
// as the merge key.
//
// NOTE:
// 	As of now we don't do merges on status because the controller is
// solely responsible for providing the entire contents of status.
// As a result, we don't try to handle things like status.conditions
var knownMergeKeys = []string{
	"uid",
	"id",
	"alias",
	"name",
	"key",
	"component",
	"containerPort",
	"container-port",
	"port",
	"ip",
}

// detectListMapKey tries to guess whether a field is a
// k8s-style "list of maps".
//
// For example in the given sample 'names' is a list of maps:
// ```yaml
// names:
// - name: abc
//   desc: blah blah
// - name: def
//   desc: blah blah blah
// - name: xyz
//   desc: blabber
// ```
//
// You pass in all known examples of values for the field.
// If a likely merge key can be found, we return it.
// Otherwise, we return an empty string.
//
// NOTE:
//	Above sample yaml will return 'name' if this yaml is run
// against this method. In other words, 'name' is decided to be
// the key that is fit to be considered as merge key for above
// list of maps.
//
// NOTE:
//	For this to work all items in **observed**, **lastApplied** &
// **desired** lists should have at-least one key in common. In
// addition, this common key should be part of knownMergeKeys.
//
// NOTE:
//	If any particular list is empty then common keys will be formed
// out of non-empty lists.
func detectListMapKey(lists ...[]interface{}) string {
	// Remember the set of keys that every object has in common
	var commonKeys map[string]bool

	// loop over observed, last applied & desired lists
	for _, list := range lists {
		for _, item := range list {
			// All the items must be objects.
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				// no need to proceed since this is not a
				// list of maps
				return ""
			}
			if commonKeys == nil {
				// one time initialization
				// initialize commonKeys to consider all the fields
				// found in this map.
				commonKeys = make(map[string]bool, len(itemMap))
				for key := range itemMap {
					commonKeys[key] = true
				}
				continue
			}

			// For all other objects, prune the set.
			for key := range commonKeys {
				if _, ok := itemMap[key]; !ok {
					// remove the earlier added key, since its not
					// common across all the items of this list
					delete(commonKeys, key)
				}
			}
		}
	}
	// If all objects have **one** of the known conventional
	// merge keys in common, we'll guess that this is a list map.
	for _, key := range knownMergeKeys {
		if commonKeys[key] {
			// first possible match is the merge key
			//
			// NOTE:
			//	If an obj has more than one keys as known merge key,
			// preference is given to the first key found in
			// knownMergeKeys
			return key
		}
	}
	// If there were no matches for the common keys, then
	// this list will **not** be considered a list of maps even
	// though technically it will be at this point.
	//
	// Returning empty string implies this is not a list of maps
	return ""
}
