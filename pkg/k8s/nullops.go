package k8s

import (
	"reflect"

	"github.com/pkg/errors"
)

// credit: https://github.com/banzaicloud/k8s-objectmatcher/blob/master/patch/deletenull.go

// func init() {
// 	// k8s.io/apimachinery/pkg/util/intstr.IntOrString behaves really badly
// 	// from JSON marshaling point of view, name can't be empty basically.
// 	// So we need to override the defined marshaling behaviour and write nil
// 	// instead of 0, because usually (in all observed cases) 0 means "not set"
// 	// for IntOrStr types.
// 	// To make this happen we need to pull in json-iterator and override the
// 	// factory marshaling overrides.
// 	json.RegisterTypeEncoderFunc("intstr.IntOrString",
// 		func(ptr unsafe.Pointer, stream *json.Stream) {
// 			i := (*intstr.IntOrString)(ptr)
// 			if i.IntValue() == 0 {
// 				if i.StrVal != "" && i.StrVal != "0" {
// 					stream.WriteString(i.StrVal)
// 				} else {
// 					stream.WriteNil()
// 				}
// 			} else {
// 				stream.WriteInt(i.IntValue())
// 			}
// 		},
// 		func(ptr unsafe.Pointer) bool {
// 			i := (*intstr.IntOrString)(ptr)
// 			return i.IntValue() == 0 && (i.StrVal == "" || i.StrVal == "0")
// 		},
// 	)
// }

// func DeleteNullInUnstructuredBytes(given []byte) ([]byte, map[string]interface{}, error) {
// 	var givenMap map[string]interface{}
//
// 	err := json.Unmarshal(given, &givenMap)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "unmarshal bytes to map")
// 	}
//
// 	filteredMap, err := DeleteNullInUnstructuredMap(givenMap)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	o, err := json.ConfigCompatibleWithStandardLibrary.Marshal(filteredMap)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "marshal map to bytes")
// 	}
//
// 	return o, filteredMap, err
// }

// DeleteNullInUnstructuredMap removes the key value pairs for those value(s)
// that represent a nil. It also removes the key: value when value of string
// type is empty i.e "".
//
// Note: This supports Kubernetes compatible unstructured types only
func DeleteNullInUnstructuredMap(m map[string]interface{}) (map[string]interface{}, error) {
	var err error
	filteredMap := make(map[string]interface{}, len(m))

	for key, val := range m {
		if val == nil || IsZero(reflect.ValueOf(val)) {
			continue
		}
		switch typedVal := val.(type) {
		default:
			// Only Kubernetes unstructured types are supported
			return nil, errors.Errorf("unsupported type %T: key %q", val, key)
		case []interface{}:
			slice, err := DeleteNullInUnstructuredSlice(typedVal)
			if err != nil {
				return nil, errors.Wrapf(err, "delete null in slice: key %q", key)
			}
			filteredMap[key] = slice
		case string, float64, bool, int64, nil:
			filteredMap[key] = val
		case map[string]interface{}:
			if len(typedVal) == 0 {
				filteredMap[key] = typedVal
				continue
			}
			var filteredSubMap map[string]interface{}
			filteredSubMap, err = DeleteNullInUnstructuredMap(typedVal)
			if err != nil {
				return nil, err
			}
			if len(filteredSubMap) != 0 {
				filteredMap[key] = filteredSubMap
			}
		}
	}
	return filteredMap, nil
}

// DeleteNullInUnstructuredSlice removes the key value pairs for those value(s)
// that represent a nil.
//
// Note: This supports Kubernetes compatible unstructured types only
func DeleteNullInUnstructuredSlice(m []interface{}) ([]interface{}, error) {
	filteredSlice := make([]interface{}, len(m))
	for idx, val := range m {
		if val == nil {
			continue
		}
		switch typedVal := val.(type) {
		default:
			// Only Kubernetes unstructured types are supported
			return nil, errors.Errorf("unsupported type %T", val)
		case []interface{}:
			filteredSubSlice, err := DeleteNullInUnstructuredSlice(typedVal)
			if err != nil {
				return nil, err
			}
			filteredSlice[idx] = filteredSubSlice
		case string, float64, bool, int64, nil:
			filteredSlice[idx] = val
		case map[string]interface{}:
			filteredMap, err := DeleteNullInUnstructuredMap(typedVal)
			if err != nil {
				return nil, err
			}
			filteredSlice[idx] = filteredMap
		}
	}
	return filteredSlice, nil
}

func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	default:
		z := reflect.Zero(v.Type())
		return v.Interface() == z.Interface()
	case reflect.Float64, reflect.Int64:
		return false
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && IsZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && IsZero(v.Field(i))
		}
		return z
	}
}
