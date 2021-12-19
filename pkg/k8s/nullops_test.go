package k8s

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeleteNullInUnstructuredMap(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name   string
		given  map[string]interface{}
		expect map[string]interface{}
		isErr  bool
	}{
		{
			name: "int is an invalid data type",
			given: map[string]interface{}{
				"hi":           "there",
				"i-am-invalid": 10,
			},
			isErr: true,
		},
		{
			name: "int64 is a valid data type",
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
			name: "int64(0) is preserved",
			given: map[string]interface{}{
				"i-am-valid": int64(0),
			},
			expect: map[string]interface{}{
				"i-am-valid": int64(0),
			},
		},
		{
			name: "float64(0.0) is preserved",
			given: map[string]interface{}{
				"i-am-valid": float64(0),
			},
			expect: map[string]interface{}{
				"i-am-valid": float64(0),
			},
		},
		{
			name: "empty string is deleted",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": "",
			},
			expect: map[string]interface{}{
				"hi": "there",
			},
		},
		{
			name: "nil is deleted",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": nil,
			},
			expect: map[string]interface{}{
				"hi": "there",
			},
		},
		{
			name: "interface{}(nil) is deleted",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": interface{}(nil),
			},
			expect: map[string]interface{}{
				"hi": "there",
			},
		},
		{
			name: `interface{}("") is deleted`,
			given: map[string]interface{}{
				"i-am-empty": interface{}(""),
			},
			expect: map[string]interface{}{},
		},
		{
			name: "[]int is invalid",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": []int{},
			},
			isErr: true,
		},
		{
			name: "[]int64 is invalid",
			given: map[string]interface{}{
				"hi":         "there",
				"i-am-empty": []int64{},
			},
			isErr: true,
		},
		{
			name: "[]interface{}{} is preserved",
			given: map[string]interface{}{
				"i-am-empty": []interface{}{},
			},
			expect: map[string]interface{}{
				"i-am-empty": []interface{}{},
			},
		},
		{
			name: "[]interface{nil} is preserved",
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
			name: "[]interface{}(nil) is deleted",
			given: map[string]interface{}{
				"i-am-empty": []interface{}(nil),
			},
			expect: map[string]interface{}{},
		},
		{
			name: `list of empty string is preserved`,
			given: map[string]interface{}{
				"list-of-empty-string": []interface{}{""},
			},
			expect: map[string]interface{}{
				"list-of-empty-string": []interface{}{""},
			},
		},
		{
			name: `list of empty strings are preserved`,
			given: map[string]interface{}{
				"list-of-empty-strings": []interface{}{"", ""},
			},
			expect: map[string]interface{}{
				"list-of-empty-strings": []interface{}{"", ""},
			},
		},
		{
			name: "list of string is preserved",
			given: map[string]interface{}{
				"list-of-string": []interface{}{"hi", "there"},
			},
			expect: map[string]interface{}{
				"list-of-string": []interface{}{"hi", "there"},
			},
		},
		{
			name: "list of int64 is preserved",
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
			name: "list of string maps is preserved",
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
			name: "list of int64 maps is preserved",
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
			if !test.isErr && err != nil {
				t.Errorf("expected no error got %+v", err)
				return
			}
			if test.isErr && err == nil {
				t.Errorf("expected error got none")
				return
			}
			if !reflect.DeepEqual(got, test.expect) {
				diff := cmp.Diff(got, test.expect)
				t.Errorf("expected no diff got: -actual +want\n%s\n", diff)
			}
		})
	}
}
