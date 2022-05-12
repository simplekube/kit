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
	"k8s.io/client-go/kubernetes/scheme"
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

func TestApply(t *testing.T) {
	t.Parallel()

	var nsName = fmt.Sprintf("test-apply-%d", rand.Int31())
	var scenarios = []struct {
		name     string
		resource client.Object
	}{
		{
			name: "should verify creation of namespace",
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
			name: "should verify namespace was updated with labels",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"hi": "there",
					},
				},
			},
		},
		{
			name: "should verify namespace was updated with annotations",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Annotations: map[string]string{
						"hi": "there",
					},
				},
			},
		},
		{
			name: "should verify namespace was updated with finalizers",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Finalizers: []string{
						"test/protect-1",
					},
				},
			},
		},
		{
			name: "should verify local state matches the cluster state",
			resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"hi": "there",
					},
					Annotations: map[string]string{
						"hi": "there",
					},
					Finalizers: []string{
						"test/protect-1",
					},
				},
			},
		},
	}

	for _, test := range scenarios {
		test := test
		t.Run(test.name, func(t *testing.T) { // tests should be run in serial order
			got, err := Apply(context.Background(), test.resource)
			assert.NoError(t, err)
			isEqual, diff, err := IsEqualWithDiffOutput(got, test.resource)
			assert.NoError(t, err)
			assert.Equal(t, true, isEqual, "-cluster +local\n%s", diff)
		})
	}
}

func TestApplyWithDriftChecks(t *testing.T) {
	t.Parallel()

	var deployName = fmt.Sprintf("test-apply-%d", rand.Int31())
	var deploySpec = appsv1.DeploymentSpec{
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
	}
	var deploy = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: "default",
			Labels: map[string]string{
				"foo-0": "bar-0",
			},
			Annotations: map[string]string{
				"foo-0": "bar-0",
			},
		},
		Spec: deploySpec,
	}

	type testable struct {
		name      string        // description of this test
		toApply   client.Object // resource to be applied
		toCompare client.Object // resource to be compared for drift check
		isDrift   bool          // update verification
	}

	// tests should be run serially i.e. one after the other
	// in the given order
	scenarios := []*testable{
		{
			name:      "should verify creation of the deployment & then verify absence of drift",
			toApply:   deploy.DeepCopy(),
			toCompare: deploy.DeepCopy(),
		},
		{
			name:      "should verify successful re-apply of the deployment & then verify absence of drift",
			toApply:   deploy.DeepCopy(),
			toCompare: deploy.DeepCopy(),
		},
		{
			name: "should verify successful update of the deployment labels & then verify presence of drift",
			toApply: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployName,
					Namespace: "default",
					Labels: map[string]string{
						"foo-0": "bar-1", // value is changed
					},
				},
				Spec: deploySpec,
			},
			toCompare: deploy.DeepCopy(),
			isDrift:   true,
		},
		{
			name: "should verify successful update of the deployment annotations & then verify presence of drift",
			toApply: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployName,
					Namespace: "default",
					Annotations: map[string]string{
						"foo-0": "bar-1", // value is changed
					},
				},
				Spec: deploySpec,
			},
			toCompare: deploy.DeepCopy(),
			isDrift:   true,
		},
		{
			name: "should verify absence of drift since local state matches cluster state",
			toApply: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployName,
					Namespace: "default",
					Labels: map[string]string{
						"foo-0": "bar-1", // value is un-changed from previous apply
					},
					Annotations: map[string]string{
						"foo-0": "bar-1", // value is un-changed from previous apply
					},
				},
				Spec: deploySpec,
			},
			toCompare: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployName,
					Namespace: "default",
					Labels: map[string]string{
						"foo-0": "bar-1",
					},
					Annotations: map[string]string{
						"foo-0": "bar-1",
					},
				},
				Spec: deploySpec,
			},
		},
		{
			name: "should verify successful update of deployment with finalizers & then verify absence of drift",
			toApply: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployName,
					Namespace: "default",
					Finalizers: []string{
						"protect.io/f-1",
					},
				},
				Spec: deploySpec,
			},
			toCompare: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployName,
					Namespace: "default",
					Finalizers: []string{
						"protect.io/f-1",
					},
				},
				Spec: deploySpec,
			},
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := Apply(ctx, scenario.toApply)
			assert.NoError(t, err)

			// required before invoking drift against old state
			scenario.toCompare.SetResourceVersion(got.GetResourceVersion())

			// verify for difference from cluster state
			isDrift, diff, err := HasDrifted(ctx, scenario.toCompare)
			assert.NoError(t, err)
			assert.Equal(t, scenario.isDrift, isDrift, "-actual + result \n%s", diff)
		})
	}
}

func TestCreateOrMerge(t *testing.T) {
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-1234",
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
	}

	type testable struct {
		name        string
		deployObj   *appsv1.Deployment
		annotations map[string]string
		labels      map[string]string
		finalizers  []string
		expected    OperationResult
	}
	// These scenarios should run serially i.e. one after the other
	// in the given order
	scenarios := []*testable{
		{
			name:      "should verify successful creation of the deployment",
			deployObj: deploy.DeepCopy(),
			expected:  OperationResultCreated,
		},
		{
			name:      "should verify no change to cluster state since it matches the local state",
			deployObj: deploy.DeepCopy(),
			expected:  OperationResultNone,
		},
		{
			name:      "should verify successful update of the deployment with labels",
			deployObj: deploy.DeepCopy(),
			labels: map[string]string{
				"foo-1": "bar-1",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			name:      "should verify successful update of the deployment with annotations",
			deployObj: deploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "bar-1",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			name:      "should verify no change in cluster state since its labels & annotations matches the local state",
			deployObj: deploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "bar-1", // no change
			},
			labels: map[string]string{
				"foo-1": "bar-1", // no change
			},
			expected: OperationResultNone,
		},
		{
			name:      "should verify successful update of the deployment with finalizers",
			deployObj: deploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
				"protect.io/compute",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			name:      "should verify successful update of the deployment by updating the finalizers",
			deployObj: deploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			name:      "should verify no change to cluster state since its labels, annotations & finalizers matches the local state",
			deployObj: deploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage", // no new change
			},
			annotations: map[string]string{
				"foo-1": "bar-1", // no new change
			},
			labels: map[string]string{
				"foo-1": "bar-1", // no new change
			},
			expected: OperationResultNone,
		},
	}
	ctx := context.Background()
	// teardown in defer statement
	defer func() {
		err := klient.Delete(ctx, deploy, &client.DeleteOptions{
			GracePeriodSeconds: new(int64), // immediate delete
		})
		if err != nil {
			t.Logf(
				"failed to teardown deployment: %s %s: %v",
				deploy.Namespace, deploy.Name, err,
			)
		}
	}()
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			if len(scenario.labels) != 0 {
				lbls := scenario.deployObj.GetLabels()
				if lbls == nil {
					lbls = make(map[string]string)
				}
				for k, v := range scenario.labels {
					lbls[k] = v
				}
				scenario.deployObj.SetLabels(lbls)
			}
			if len(scenario.annotations) != 0 {
				anns := scenario.deployObj.GetAnnotations()
				if anns == nil {
					anns = make(map[string]string)
				}
				for k, v := range scenario.annotations {
					anns[k] = v
				}
				scenario.deployObj.SetAnnotations(anns)
			}
			if scenario.finalizers != nil {
				if len(scenario.finalizers) == 0 {
					scenario.deployObj.SetFinalizers(nil)
				} else {
					scenario.deployObj.SetFinalizers(scenario.finalizers)
				}
			}
			result, err := CreateOrMerge(ctx, klient, scheme.Scheme, scenario.deployObj)
			assert.NoError(t, err)
			assert.Equal(t, scenario.expected, result)
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
