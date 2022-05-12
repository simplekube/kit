package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGet(t *testing.T) {
	t.Parallel()

	var scenarios = []struct {
		name               string
		object             client.Object
		expectedObjectName string
		isError            bool
	}{
		{
			name: "should verify existence of default namespace",
			object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			expectedObjectName: "default",
		},
		{
			name: "should verify non existence of 'none' namespace",
			object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "none",
				},
			},
			isError: true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			got, err := Get(context.Background(), scenario.object)
			if scenario.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, scenario.expectedObjectName, got.GetName())
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	t.Parallel()

	var scenarios = []struct {
		name                string
		objects             []client.Object
		expectedObjectNames []string
		isError             bool
	}{
		{
			name: "should verify existence of default namespace",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
			},
			expectedObjectNames: []string{"default"},
		},
		{
			name: "should verify configmap can not be fetched without a namespace",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
			},
			isError: true,
		},
		{
			// Note: Since local k8s binaries are used for testing
			name: "should verify non existence of default service account",
			objects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "default",
					},
				},
			},
			isError: true,
		},
		{
			name: "should verify non existence of 'none' namespace",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "none",
					},
				},
			},
			isError: true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetAll(context.Background(), scenario.objects)
			if scenario.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				var expectedNameSet = sets.NewString(scenario.expectedObjectNames...)
				var actualNameSet = sets.String{}
				for _, g := range got {
					actualNameSet.Insert(g.GetName())
				}
				assert.Condition(t, func() (success bool) {
					return actualNameSet.Equal(expectedNameSet)
				})
			}
		})
	}
}

func TestGetForYAML(t *testing.T) {
	t.Parallel()

	var scenarios = []struct {
		name               string
		fixture            string
		expectedObjectName string
		isError            bool
	}{
		{
			name:    "should verify yaml with empty content errors out",
			fixture: "testdata/empty.yaml",
			isError: true,
		},
		{
			name:    "should verify yaml with non kubernetes schema errors out",
			fixture: "testdata/non_kubernetes.yaml",
			isError: true,
		},
		{
			name:               "should verify yaml with default namespace exists",
			fixture:            "testdata/default_namespace.yaml",
			expectedObjectName: "default",
		},
		{
			name:    "should verify yaml with non-existing namespace errors out",
			fixture: "testdata/custom_namespace.yaml",
			isError: true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetForYAML(context.Background(), scenario.fixture)
			if scenario.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, scenario.expectedObjectName, got.GetName())
			}
		})
	}
}

func TestGetForAllYAMLs(t *testing.T) {
	t.Parallel()

	var scenarios = []struct {
		name                string
		fixtures            []string
		expectedObjectNames []string
		isError             bool
	}{
		{
			name:     "should verify yaml with empty content errors out",
			fixtures: []string{"testdata/empty.yaml"},
			isError:  true,
		},
		{
			name:     "should verify yaml with non kubernetes schema errors out",
			fixtures: []string{"testdata/non_kubernetes.yaml"},
			isError:  true,
		},
		{
			name:     "should verify yaml with invalid kubernetes schema errors out",
			fixtures: []string{"testdata/invalid_kubernetes.yaml"},
			isError:  true,
		},
		{
			name:     "should verify yaml with non kubernetes & kubernetes schema errors out",
			fixtures: []string{"testdata/non_kubernetes_and_custom_namespace.yaml"},
			isError:  true,
		},
		{
			name:                "should verify yaml with default namespace exists",
			fixtures:            []string{"testdata/default_namespace.yaml"},
			expectedObjectNames: []string{"default"},
		},
		{
			name:     "should verify yaml with non-existing namespace errors out",
			fixtures: []string{"testdata/custom_namespace.yaml"},
			isError:  true,
		},
		{
			name:     "verify yaml with a list of non-existent namespaces errors out",
			fixtures: []string{"testdata/custom_namespace_list.yaml"},
			isError:  true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetForAllYAMLs(context.Background(), scenario.fixtures)
			if scenario.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				var expectedNameSet = sets.NewString(scenario.expectedObjectNames...)
				var actualNameSet = sets.String{}
				for _, g := range got {
					actualNameSet.Insert(g.GetName())
				}
				assert.Condition(t, func() (success bool) {
					return actualNameSet.Equal(expectedNameSet)
				})
			}
		})
	}
}
