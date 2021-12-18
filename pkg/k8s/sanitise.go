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

// func DeleteNullInUnstruct(given []byte) ([]byte, map[string]interface{}, error) {
// 	var patchMap map[string]interface{}
//
// 	err := json.Unmarshal(given, &patchMap)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "could not unmarshal json patch")
// 	}
//
// 	filteredMap, err := DeleteNullInUnstruct(patchMap)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "could not delete null values from patch map")
// 	}
//
// 	o, err := json.ConfigCompatibleWithStandardLibrary.Marshal(filteredMap)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "could not marshal filtered patch map")
// 	}
//
// 	return o, filteredMap, err
// }

func DeleteNullInUnstruct(m map[string]interface{}) (map[string]interface{}, error) {
	var err error
	filteredMap := make(map[string]interface{})

	for key, val := range m {
		if val == nil || IsZero(reflect.ValueOf(val)) {
			continue
		}
		switch typedVal := val.(type) {
		default:
			return nil, errors.Errorf("unknown type: %T", val)
		case []interface{}:
			slice, err := DeleteNullInSlice(typedVal)
			if err != nil {
				return nil, errors.Errorf("failed to delete null value(s): key %q", key)
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
			filteredSubMap, err = DeleteNullInUnstruct(typedVal)
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

func DeleteNullInSlice(m []interface{}) ([]interface{}, error) {
	filteredSlice := make([]interface{}, len(m))
	for key, val := range m {
		if val == nil {
			continue
		}
		switch typedVal := val.(type) {
		default:
			return nil, errors.Errorf("unknown type: %T", val)
		case []interface{}:
			filteredSubSlice, err := DeleteNullInSlice(typedVal)
			if err != nil {
				return nil, err
			}
			filteredSlice[key] = filteredSubSlice
		case string, float64, bool, int64, nil:
			filteredSlice[key] = val
		case map[string]interface{}:
			filteredMap, err := DeleteNullInUnstruct(typedVal)
			if err != nil {
				return nil, err
			}
			filteredSlice[key] = filteredMap
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
