package k8s

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestDeleteNullInUnstructuredMap(t *testing.T) {
	t.Parallel()

	errContentUnsupportedType := "unsupported type"
	var tests = []struct {
		name       string
		given      map[string]interface{}
		expect     map[string]interface{}
		errContent string
		isErr      bool
	}{
		{
			name: "field with int value is unsupported",
			given: map[string]interface{}{
				"hi":           "there",
				"i-am-invalid": 10,
			},
			errContent: errContentUnsupportedType,
			isErr:      true,
		},
		{
			name: "field with int64 value is supported & is preserved",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-valid": int64(10),
			},
			expect: map[string]interface{}{
				"hi":         "there",
				"i-am-valid": int64(10),
			},
		},
		{
			name: "field with int64(0) value is supported & is preserved",
			given: map[string]interface{}{
				"i-am-valid": int64(0),
			},
			expect: map[string]interface{}{
				"i-am-valid": int64(0),
			},
		},
		{
			name: "field with float64(0.0) is supported & is preserved",
			given: map[string]interface{}{
				"i-am-valid": float64(0),
			},
			expect: map[string]interface{}{
				"i-am-valid": float64(0),
			},
		},
		{
			name: "field with empty string value is deleted",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": "",
			},
			expect: map[string]interface{}{
				"hi": "there",
			},
		},
		{
			name: "field with nil value is deleted",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": nil,
			},
			expect: map[string]interface{}{
				"hi": "there",
			},
		},
		{
			name: "field with interface{}(nil) value is deleted",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": interface{}(nil),
			},
			expect: map[string]interface{}{
				"hi": "there",
			},
		},
		{
			name: `field with interface{}("") value is deleted`,
			given: map[string]interface{}{
				"i-am-empty": interface{}(""),
			},
			expect: map[string]interface{}{},
		},
		{
			name: "field with []int value is unsupported",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": []int{},
			},
			errContent: errContentUnsupportedType,
			isErr:      true,
		},
		{
			name: "field with []int64 value is unsupported",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": []int64{},
			},
			errContent: errContentUnsupportedType,
			isErr:      true,
		},
		{
			name: "field with []string{} is unsupported",
			given: map[string]interface{}{
				"hi":                             "there",
				"array-of-string-without-values": []string{},
			},
			errContent: errContentUnsupportedType,
			isErr:      true,
		},
		{
			name: "field with []interface{}{} is preserved",
			given: map[string]interface{}{
				"i-am-empty": []interface{}{},
			},
			expect: map[string]interface{}{
				"i-am-empty": []interface{}{},
			},
		},
		{
			name: "field with []interface{nil} is preserved",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": []interface{}{nil},
			},
			expect: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": []interface{}{nil},
			},
		},
		{
			name: "field with []interface{}(nil) value is deleted",
			given: map[string]interface{}{
				"i-am-empty": []interface{}(nil),
			},
			expect: map[string]interface{}{},
		},
		{
			name: `field with []interface{}{""} is preserved`,
			given: map[string]interface{}{
				"list-of-empty-string": []interface{}{""},
			},
			expect: map[string]interface{}{
				"list-of-empty-string": []interface{}{""},
			},
		},
		{
			name: `field with list of empty strings is preserved`,
			given: map[string]interface{}{
				"list-of-empty-strings": []interface{}{"", ""},
			},
			expect: map[string]interface{}{
				"list-of-empty-strings": []interface{}{"", ""},
			},
		},
		{
			name: "field with list of strings is preserved",
			given: map[string]interface{}{
				"list-of-string": []interface{}{"hi", "there"},
			},
			expect: map[string]interface{}{
				"list-of-string": []interface{}{"hi", "there"},
			},
		},
		{
			name: "field with list of int64 is preserved",
			given: map[string]interface{}{
				"hi":            "there",
				"list-of-int64": []interface{}{int64(1), int64(2)},
			},
			expect: map[string]interface{}{
				"hi":            "there",
				"list-of-int64": []interface{}{int64(1), int64(2)},
			},
		},
		{
			name: "field with list of string maps is preserved",
			given: map[string]interface{}{
				"list-of-string-map": []interface{}{
					map[string]interface{}{
						"k": "v",
					},
					map[string]interface{}{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
			expect: map[string]interface{}{
				"list-of-string-map": []interface{}{
					map[string]interface{}{
						"k": "v",
					},
					map[string]interface{}{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
		{
			name: "field with list of int64 maps is preserved",
			given: map[string]interface{}{
				"list-of-int64-map": []interface{}{
					map[string]interface{}{
						"k": int64(1),
					},
					map[string]interface{}{
						"k1": int64(1),
						"k2": int64(2),
					},
				},
			},
			expect: map[string]interface{}{
				"list-of-int64-map": []interface{}{
					map[string]interface{}{
						"k": int64(1),
					},
					map[string]interface{}{
						"k1": int64(1),
						"k2": int64(2),
					},
				},
			},
		},
	}
	for _, test := range tests {
		test := test // pin it
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := DeleteNullInUnstructuredMap(test.given)
			if !test.isErr {
				assert.NoError(t, err)
				if !reflect.DeepEqual(got, test.expect) {
					diff := cmp.Diff(got, test.expect)
					assert.Equal(t, "", fmt.Sprintf("-actual +want\n%s\n", diff))
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}
