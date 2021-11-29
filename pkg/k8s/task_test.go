package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTaskOperations(t *testing.T) {
	var tests = []struct {
		name       string
		resource   client.Object
		action     ActionType
		preAction  func(object client.Object) error
		postAction func(object client.Object) error
		assert     AssertType
	}{
		{
			name: "should assert presence of default namespace",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			action: ActionTypeGet,
			assert: AssertTypeIsFound,
		},
		{
			name: "should assert absence of the configmap",
			resource: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-%d", rand.Int31()),
					Namespace: "default",
				},
			},
			action: ActionTypeGet,
			assert: AssertTypeIsNotFound,
		},
		{
			name: "should create a configmap via create action",
			resource: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-%d", rand.Int31()),
					Namespace: "default",
				},
			},
			action: ActionTypeCreate,
		},
		{
			name: "should create a configmap via upsert action",
			resource: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-%d", rand.Int31()),
					Namespace: "default",
				},
			},
			action: ActionTypeCreateOrMerge,
		},
		{
			name: "should upsert default namespace with labels & assert the entire state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
					Labels: map[string]string{
						"testing": "ok",
					},
				},
			},
			action: ActionTypeCreateOrMerge,
			assert: AssertTypeIsEquals,
		},
		{
			name: "should assert presence of configmap labels via post action",
			resource: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-%d", rand.Int31()),
					Namespace: "default",
					Labels: map[string]string{
						"test": "ok",
					},
				},
			},
			action: ActionTypeCreate,
			postAction: func(obj client.Object) error {
				cm, _ := obj.(*corev1.ConfigMap)
				expectedCount := 1
				if expectedCount != len(cm.Labels) {
					return errors.Errorf("expected labels count %d got %d", expectedCount, len(cm.Labels))
				}
				return nil
			},
		},
		{
			name: "should add configmap annotations via pre action & assert its presence via post action",
			resource: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-%d", rand.Int31()),
					Namespace: "default",
				},
			},
			action: ActionTypeCreate,
			preAction: func(obj client.Object) error {
				cm, _ := obj.(*corev1.ConfigMap)
				cm.SetAnnotations(map[string]string{
					"test": "ok",
				})
				return nil
			},
			postAction: func(obj client.Object) error {
				cm, _ := obj.(*corev1.ConfigMap)
				expectedCount := 1
				if expectedCount != len(cm.Annotations) {
					return errors.Errorf("expected annotations count %d got %d", expectedCount, len(cm.Annotations))
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		test := test // pin name for parallel execution of these tests
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tsk := &Task{
				It:         test.name,
				Action:     test.action,
				Resource:   test.resource,
				Assert:     test.assert,
				PreAction:  test.preAction,
				PostAction: test.postAction,
			}
			err := tsk.Run(context.Background(), &RunOptions{
				Client: klient,
			})
			assert.NoError(t, err, "failed to verify %s", test.name)
		})
	}
}
