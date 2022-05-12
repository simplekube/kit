package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetKindVersionForObject(t *testing.T) {
	t.Parallel()

	var scenarios = []struct {
		name            string
		object          client.Object
		expectedKind    string
		expectedVersion string
		isError         bool
	}{
		{
			name: "should fetch the kind & version of kubernetes configmap",
			object: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "howdy",
				},
			},
			expectedKind:    "ConfigMap",
			expectedVersion: "v1",
		},
		{
			name: "should fail since resource is unknown",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			isError: true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			k, v, err := GetKindVersionForObject(scenario.object, rscheme)
			if scenario.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, scenario.expectedKind, k)
				assert.Equal(t, scenario.expectedVersion, v)
			}
		})
	}
}

func TestDryRun(t *testing.T) {
	t.Parallel()

	var scenarios = []struct {
		name     string
		resource client.Object
	}{
		{
			name: "should verify successful dry run of the configmap",
			resource: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dryrun-cm",
					Namespace: "default",
				},
			},
		},
		{
			name: "should verify successful dry run of the deployment",
			resource: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-dryrun-%d", rand.Int31()),
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "busybox",
									Image: "busybox",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			got, err := DryRun(ctx, scenario.resource)
			assert.NoError(t, err)
			isEqual, err := IsEqual(got, scenario.resource)
			assert.NoError(t, err)
			assert.True(t, isEqual)
		})
	}
}

func TestHasDrifted(t *testing.T) {
	t.Parallel()

	var nsName = fmt.Sprintf("test-has-drifted-%d", rand.Int31())
	var ns = &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	_, err := Create(context.Background(), ns)
	assert.NoError(t, err)

	// Note: These scenarios must run serially i.e. one after the other
	var scenarios = []struct {
		name       string
		resource   client.Object
		preDriftFn func(obj client.Object) error // is run before invoking drift
		isDrift    bool
	}{
		{
			name: "should verify absence of drift when local state matches the cluster state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			},
		},
		{
			name: "should add label to the local state & verify presence of drift",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"dummy": "testing",
					},
				},
			},
			isDrift: true,
		},
		{
			name: "should update label against the cluster state & then verify absence of drift",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"dummy": "testing",
					},
				},
			},
			preDriftFn: func(obj client.Object) error { // object set in the resource field is sent as the argument
				// update this object at the cluster
				_, err := Update(context.Background(), obj)
				return err
			},
		},
		{
			name: "should verify absence of drift since local state matches the cluster state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"dummy": "testing",
					},
				},
			},
		},
		{
			name: "should verify presence of drift since label of local state does not match to that of cluster state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"dummy": "testing-2",
					},
				},
			},
			isDrift: true,
		},
		{
			name: "should verify presence of drift since local state has finalizers while cluster state does not",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Finalizers: []string{
						"test.protect/drift-1",
						"test.protect/drift-2",
					},
				},
			},
			isDrift: true,
		},
		{
			name: "should verify absence of drift since cluster state is updated with finalizers & hence matches local state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Finalizers: []string{
						"test.protect/drift-1",
						"test.protect/drift-2",
					},
				},
			},
			preDriftFn: func(obj client.Object) error { // object set in the resource field is sent as the argument
				// update this object at the cluster
				_, err := Update(context.Background(), obj)
				return err
			},
		},
		{
			name: "should verify absence of drift since the finalizer used in local state matches that of cluster state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Finalizers: []string{
						"test.protect/drift-2",
					},
				},
			},
		},
	}

	for _, test := range scenarios {
		test := test
		t.Run(test.name, func(t *testing.T) { // tests should be executed in serial order
			if test.preDriftFn != nil {
				err := test.preDriftFn(test.resource)
				assert.NoError(t, err)
			}
			isDrift, diff, err := HasDrifted(context.Background(), test.resource)
			assert.NoError(t, err)
			assert.Equal(t, test.isDrift, isDrift, "-want +got\n%s", diff)
		})
	}
}

func TestObjectEqual(t *testing.T) {
	t.Parallel()

	deployObj := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-abcd",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo": "bar",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "busybox",
						},
					},
				},
			},
		},
	}

	type testable struct {
		name     string
		observed client.Object
		desired  client.Object
		isEqual  bool
	}
	scenarios := []testable{
		{
			name:     "is equal if observed deployment state equals desired state",
			observed: deployObj.DeepCopy(),
			desired:  deployObj.DeepCopy(),
			isEqual:  true,
		},
		{
			name: "is equal if observed deployment state is superset of desired state",
			observed: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "deploy-abcd",
					Namespace:       "default",
					UID:             "1234-1234-1234", // extra
					ResourceVersion: "1232122",        // extra
					Finalizers: []string{
						"protect.io/storage", // extra
						"protect.io/network", // extra
					},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo":    "bar",
							"foobar": "true", // extra
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo":    "bar",
								"foobar": "true", // extra
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "busybox",
									Image:   "busybox",
									Command: []string{"busy", "there"}, // extra
								},
								{
									Name:  "sleep", // extra
									Image: "sleep", // extra
								},
							},
						},
					},
				},
			},
			desired: deployObj.DeepCopy(),
			isEqual: true,
		},
		{
			name:     "is not equal when desired deployment state is a superset of observed state",
			observed: deployObj.DeepCopy(),
			desired: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deploy-abcd",
					Namespace: "default",
					Finalizers: []string{
						"protect.io/storage", // extra
						"protect.io/network", // extra
					},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo":    "bar",
							"foobar": "true", // extra
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo":    "bar",
								"foobar": "true", // extra
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "busybox",
									Image:   "busybox",
									Command: []string{"busy", "there"}, // extra
								},
								{
									Name:  "sleep", // extra
									Image: "sleep", // extra
								},
							},
						},
					},
				},
			},
			isEqual: false,
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario // pin it
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			isEqual, err := IsEqual(scenario.observed, scenario.desired)
			assert.NoError(t, err)
			assert.Equal(t, scenario.isEqual, isEqual)
		})
	}
}
