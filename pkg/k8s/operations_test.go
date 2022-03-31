package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/simplekube/kit/pkg/pointer"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
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

// TestUpsertVerbose verifies Upsert logic
//
// Note: All the scenarios should be run in a serial order
// Note: Each scenario is dependent on its previous scenario
func TestUpsertVerbose(t *testing.T) {
	desiredDeploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("test-deploy-%d", rand.Int31()),
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
		it          string
		deployObj   *appsv1.Deployment
		annotations map[string]string
		labels      map[string]string
		finalizers  []string
		result      OperationResult
		isUpsert    bool
	}

	// these test scenarios must run serially
	scenarios := []*testable{
		{
			it:        "should verify successful creation of deployment",
			deployObj: desiredDeploy.DeepCopy(),
			result:    OperationResultCreated,
			isUpsert:  true,
		},
		{
			it:        "should verify no change to cluster state since it matches the local state",
			deployObj: desiredDeploy.DeepCopy(),
			result:    OperationResultNone,
		},
		{
			it:        "should verify no change to cluster state since label is set to empty value",
			deployObj: desiredDeploy.DeepCopy(),
			labels: map[string]string{
				"foo-1": "",
			},
			result: OperationResultNone,
		},
		{
			it:        "should verify no change to cluster state since annotation set to empty value",
			deployObj: desiredDeploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "",
			},
			result: OperationResultNone,
		},
		{
			it:        "should verify successful update to cluster state due to addition of new label pairs",
			deployObj: desiredDeploy.DeepCopy(),
			labels: map[string]string{
				"foo-1": "bar-1",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			it:        "should verify successful update to cluster state due to addition of new annotation pairs",
			deployObj: desiredDeploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "bar-1",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			it:        "should verify no change to cluster state since labels & annotations remain same as previous",
			deployObj: desiredDeploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "bar-1", // no change
			},
			labels: map[string]string{
				"foo-1": "bar-1", // no change
			},
			result: OperationResultNone,
		},
		{
			it:        "should update the deployment with new finalizers",
			deployObj: desiredDeploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
				"protect.io/compute",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			it:        "should update the deployment by updating the finalizers",
			deployObj: desiredDeploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			it:        "should not result in any change since labels, annotations & finalizers remain same",
			deployObj: desiredDeploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage", // no new change
			},
			annotations: map[string]string{
				"foo-1": "bar-1", // no new change
			},
			labels: map[string]string{
				"foo-1": "bar-1", // no new change
			},
			result: OperationResultNone,
		},
	}
	ctx := context.Background()
	// teardown in defer statement
	defer func() {
		desiredDeploy.SetFinalizers(nil) // remove finalizers before delete
		updateErr := klient.Update(ctx, desiredDeploy)
		if updateErr != nil {
			t.Logf("teardown deployment - remove finalizers: %s %s: %v", desiredDeploy.Namespace, desiredDeploy.Name, updateErr)
		}
		deleteErr := klient.Delete(ctx, desiredDeploy, &client.DeleteOptions{
			GracePeriodSeconds: new(int64), // immediate delete
		})
		if deleteErr != nil {
			t.Logf("teardown deployment - delete: %s %s: %v", desiredDeploy.Namespace, desiredDeploy.Name, deleteErr)
		}
	}()
	// Note: These scenarios should not be run in parallel since
	// execution of each scenario depends on the execution of
	// previous scenario
	for _, testcase := range scenarios {
		testcase := testcase // pin it
		t.Run(testcase.it, func(t *testing.T) {
			if len(testcase.labels) != 0 {
				lbls := testcase.deployObj.GetLabels()
				if lbls == nil {
					lbls = make(map[string]string)
				}
				for k, v := range testcase.labels {
					lbls[k] = v
				}
				testcase.deployObj.SetLabels(lbls)
			}
			if len(testcase.annotations) != 0 {
				anns := testcase.deployObj.GetAnnotations()
				if anns == nil {
					anns = make(map[string]string)
				}
				for k, v := range testcase.annotations {
					anns[k] = v
				}
				testcase.deployObj.SetAnnotations(anns)
			}
			if testcase.finalizers != nil {
				testcase.deployObj.SetFinalizers(testcase.finalizers)
			}
			upsertedObj, result, err := UpsertVerbose(ctx, testcase.deployObj)
			assert.NoError(t, err)
			assert.Equal(t, testcase.result, result)
			if testcase.isUpsert {
				assert.NotNil(t, upsertedObj)
			} else {
				assert.Nil(t, upsertedObj)
			}
		})
	}
}

// TestUpsertVerboseWithOptionAcceptNullFieldValues verifies Upsert logic
// with AcceptNullFieldValuesDuringUpsert option set to true
//
// Note: All the scenarios should be run in a serial order
// Note: Execution of each scenario is **dependent** on execution of
// previous scenario
func TestUpsertVerboseWithOptionAcceptNullFieldValues(t *testing.T) {
	desiredDeploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("test-deploy-%d", rand.Int31()),
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

	// these test scenarios must run one after the other
	scenarios := []struct {
		name        string
		deployObj   *appsv1.Deployment
		annotations map[string]string
		labels      map[string]string
		finalizers  []string
		result      OperationResult
		isUpsert    bool
	}{
		{
			name:      "should verify successful creation of deployment",
			deployObj: desiredDeploy.DeepCopy(),
			result:    OperationResultCreated,
			isUpsert:  true,
		},
		{
			name:      "should verify no change to cluster state since it matches the desired state",
			deployObj: desiredDeploy.DeepCopy(),
			result:    OperationResultNone,
		},
		{
			name:      "should verify change to cluster state since a label with empty value is added",
			deployObj: desiredDeploy.DeepCopy(),
			labels: map[string]string{
				"foo-1": "",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			name:      "should verify change to cluster state since an annotation with empty value is added",
			deployObj: desiredDeploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			name:      "should verify no change to cluster state since it matches the desired state",
			deployObj: desiredDeploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "", // no change
			},
			labels: map[string]string{
				"foo-1": "", // no change
			},
			result: OperationResultNone,
		},
		{
			name:      "should verify change to cluster state due to addition of finalizers",
			deployObj: desiredDeploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
				"protect.io/compute",
			},
			result:   OperationResultUpdatedResourceOnly,
			isUpsert: true,
		},
		{
			name:       "should verify no change to cluster state when finalizers is set to empty value",
			deployObj:  desiredDeploy.DeepCopy(),
			finalizers: []string{},
			result:     OperationResultNone, // BUG?: []string{} is not set
		},
		{
			name:       "should verify no change to cluster state when finalizers is set to nil",
			deployObj:  desiredDeploy.DeepCopy(),
			finalizers: []string(nil),
			result:     OperationResultNone, // BUG?: []string(nil) is not set
		},
	}
	ctx := context.Background()
	// teardown in defer statement
	defer func() {
		desiredDeploy.SetFinalizers(nil)
		updateErr := klient.Update(ctx, desiredDeploy)
		if updateErr != nil {
			t.Logf("teardown deployment: remove finalizers: %s %s: %v", desiredDeploy.Namespace, desiredDeploy.Name, updateErr)
		}
		deleteErr := klient.Delete(ctx, desiredDeploy, &client.DeleteOptions{
			GracePeriodSeconds: new(int64), // immediate delete
		})
		if deleteErr != nil {
			t.Logf("teardown deployment: delete operation: %s %s: %v", desiredDeploy.Namespace, desiredDeploy.Name, deleteErr)
		}
	}()
	for _, scenario := range scenarios {
		testcase := scenario // pin it
		t.Run(testcase.name, func(t *testing.T) {
			if testcase.labels != nil {
				lbls := testcase.deployObj.GetLabels()
				if lbls == nil {
					lbls = make(map[string]string)
				}
				for k, v := range testcase.labels {
					lbls[k] = v
				}
				testcase.deployObj.SetLabels(lbls)
			}
			if testcase.annotations != nil {
				anns := testcase.deployObj.GetAnnotations()
				if anns == nil {
					anns = make(map[string]string)
				}
				for k, v := range testcase.annotations {
					anns[k] = v
				}
				testcase.deployObj.SetAnnotations(anns)
			}
			if testcase.finalizers != nil {
				testcase.deployObj.SetFinalizers(testcase.finalizers)
			}
			// target run options under test
			opts := &RunOptions{AcceptNullFieldValuesDuringUpsert: pointer.Bool(true)}
			// target function under test
			upsertedObj, result, err := UpsertVerbose(ctx, testcase.deployObj, opts)
			assert.NoError(t, err)
			assert.Equal(t, testcase.result, result)
			if testcase.isUpsert {
				assert.NotNil(t, upsertedObj)
			} else {
				assert.Nil(t, upsertedObj)
			}
		})
	}
}

// TestUpsertVerboseWithOptionSetFinalizersToNull verifies Upsert logic
// with SetFinalizersToNullDuringUpsert option set
//
// Note: All the scenarios should be run in a serial order
// Note: Execution of each scenario is **dependent** on execution of
// previous scenario
func TestUpsertVerboseWithOptionSetFinalizersToNull(t *testing.T) {
	desiredDeploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       fmt.Sprintf("test-deploy-%d", rand.Int31()),
			Namespace:  "default",
			Finalizers: []string{"protect/testing"},
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

	// these test scenarios must run one after the other
	scenarios := []struct {
		name               string
		deployObj          *appsv1.Deployment
		setFinalizersToNil bool
		result             OperationResult
		isUpsert           bool
	}{
		{
			name:               "should verify successful creation of deployment",
			deployObj:          desiredDeploy.DeepCopy(),
			setFinalizersToNil: false,
			result:             OperationResultCreated,
			isUpsert:           true,
		},
		{
			name:               "should verify no change to cluster state when run options is not configured to set finalizers to nil",
			deployObj:          desiredDeploy.DeepCopy(),
			setFinalizersToNil: false,
			result:             OperationResultNone,
		},
		{
			name:               "should verify change to cluster state when run options is configured to set finalizers to nil",
			deployObj:          desiredDeploy.DeepCopy(),
			setFinalizersToNil: true,
			result:             OperationResultUpdatedResourceOnly,
			isUpsert:           true,
		},
	}
	ctx := context.Background()
	// teardown in defer statement
	defer func() {
		desiredDeploy.SetFinalizers(nil)
		updateErr := klient.Update(ctx, desiredDeploy)
		if updateErr != nil {
			t.Logf("teardown deployment: remove finalizers: %s %s: %v", desiredDeploy.Namespace, desiredDeploy.Name, updateErr)
		}
		deleteErr := klient.Delete(ctx, desiredDeploy, &client.DeleteOptions{
			GracePeriodSeconds: new(int64), // immediate delete
		})
		if deleteErr != nil {
			t.Logf("teardown deployment: delete operation: %s %s: %v", desiredDeploy.Namespace, desiredDeploy.Name, deleteErr)
		}
	}()
	for _, scenario := range scenarios {
		testcase := scenario // pin it
		t.Run(testcase.name, func(t *testing.T) {
			// target run options under test
			opts := &RunOptions{}
			if testcase.setFinalizersToNil {
				opts.SetFinalizersToNullDuringUpsert = pointer.Bool(true)
			}
			// target function under test
			upsertedObj, result, err := UpsertVerbose(ctx, testcase.deployObj, opts)
			assert.NoError(t, err)
			assert.Equal(t, testcase.result, result)
			if testcase.isUpsert {
				assert.NotNil(t, upsertedObj)
			} else {
				assert.Nil(t, upsertedObj)
			}
		})
	}
}
