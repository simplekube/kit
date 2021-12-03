package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetKindVersionForObject(t *testing.T) {
	t.Parallel()

	var testcases = []struct {
		name            string
		object          client.Object
		expectedKind    string
		expectedVersion string
		isError         bool
	}{
		{
			name: "should get the kind & version of kubernetes configmap",
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

	for _, test := range testcases {
		test := test // pin it
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			k, v, err := GetKindVersionForObject(test.object, rscheme)
			if err != nil && !test.isError {
				t.Fatalf("expected no error: got %+v", err)
			}
			if err == nil && test.isError {
				t.Fatal("expected error: got none")
			}
			if test.isError {
				return
			}
			if k != test.expectedKind {
				t.Fatalf("expected kind %q got %q", test.expectedKind, k)
			}
			if v != test.expectedVersion {
				t.Fatalf("expected version %q got %q", test.expectedVersion, v)
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	var testcases = []struct {
		name               string
		object             client.Object
		expectedObjectName string
		isError            bool
	}{
		{
			name: "default namespace exists",
			object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			expectedObjectName: "default",
		},
		{
			name: "none namespace does not exist",
			object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "none",
				},
			},
			isError: true,
		},
	}

	for _, test := range testcases {
		test := test // pin it
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Get(context.Background(), test.object)
			if err != nil && !test.isError {
				t.Fatalf("expected no error got %+v", err)
			}
			if err == nil && test.isError {
				t.Fatalf("expected error got none")
			}
			if test.isError {
				return
			}
			if test.expectedObjectName != got.GetName() {
				t.Fatalf("expected object name %q got %q", test.expectedObjectName, got.GetName())
			}
		})
	}
}

func TestDryRun(t *testing.T) {
	t.Parallel()

	var testcases = []struct {
		name     string
		resource client.Object
	}{
		{
			name: "should dry run a configmap",
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
			name: "should dry run a deployment",
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

	for _, test := range testcases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			got, err := DryRun(ctx, test.resource)
			if err != nil {
				t.Errorf("expected no error got: %+v", err)
			}
			isEqual, err := IsEqual(got, test.resource)
			if err != nil {
				t.Errorf("expected no error got: %+v", err)
			}
			if !isEqual {
				diff := cmp.Diff(got, test.resource)
				t.Errorf("expected no diff got:  -got +want\n%s\n", diff)
			}
		})
	}
}

func TestHasDrifted(t *testing.T) {
	t.Parallel()

	var (
		nsName = fmt.Sprintf("test-has-drifted-%d", rand.Int31())
	)
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
	if err != nil {
		t.Errorf("expected no error got: %+v", err)
	}

	var testcases = []struct {
		name       string
		resource   client.Object
		preDriftFn func(obj client.Object) error // is run before invoking drift
		isDrift    bool
	}{
		{
			name: "verify absence of drift when local namespace instance matches the cluster instance",
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
			name: "add label to the local namespace instance & verify presence of drift",
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
			name: "update label against the cluster namespace instance & then verify absence of drift",
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
			name: "provide a local namespace instance same as cluster instance instance & then verify absence of drift",
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
			name: "update existing label to the local namespace instance & then verify presence of drift",
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
			name: "add finalizers to the local namespace instance & verify presence of drift",
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
			name: "update finalizers to the cluster namespace instance & then verify absence of drift",
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
			name: "use a finalizer in the local namespace instance that is also present in cluster instance & then verify absence of drift",
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

	for _, test := range testcases {
		test := test
		t.Run(test.name, func(t *testing.T) { // tests should be executed in serial order
			if test.preDriftFn != nil {
				err := test.preDriftFn(test.resource)
				if err != nil {
					t.Errorf("expected no error during update, got: %+v", err)
				}
			}
			isDrift, diff, err := HasDrifted(context.Background(), test.resource)
			if err != nil {
				t.Errorf("expected no error while checking for drift, got: %+v", err)
			}
			if test.isDrift != isDrift {
				t.Errorf(
					"expected drift '%t', got '%t': diff '%t': -actual +expected\n%s",
					test.isDrift,
					isDrift,
					diff != "",
					diff,
				)
			}
		})
	}
}

func TestApply(t *testing.T) {
	t.Parallel()

	var nsName = fmt.Sprintf("test-apply-%d", rand.Int31())
	var testcases = []struct {
		name     string
		resource client.Object
	}{
		{
			name: "should create a namespace",
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
			name: "should update the namespace with labels",
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
			name: "should update the namespace with annotations",
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
			name: "should update the namespace with finalizers",
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
			name: "no issues if state to be applied matches the cluster state",
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

	for _, test := range testcases {
		test := test                          // pin it for parallel tests to be executed without issues
		t.Run(test.name, func(t *testing.T) { // tests should be run in serial order
			got, err := Apply(context.Background(), test.resource)
			if err != nil {
				t.Errorf("expected no error, got: %+v", err)
			}
			isEqual, diff, err := IsEqualWithDiffOutput(got, test.resource)
			if err != nil {
				t.Errorf("expected no error while checking for equality, got: %+v", err)
			}
			if !isEqual {
				t.Errorf("expected local state equal to cluster state, got diff: -cluster +local\n%s", diff)
			}
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
	tests := []*testable{
		{
			name:      "should create the deployment & verify absence of drift",
			toApply:   deploy.DeepCopy(),
			toCompare: deploy.DeepCopy(),
		},
		{
			name:      "should not have issues by re-applying existing deployment & verify absence of drift",
			toApply:   deploy.DeepCopy(),
			toCompare: deploy.DeepCopy(),
		},
		{
			name: "should update existing deployment labels & verify presence of drift",
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
			name: "should update existing deployment with annotations & verify presence of drift",
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
			name: "should not result in any drift with local state exactly same as cluster state",
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
					Annotations: map[string]string{
						"foo-0": "bar-1", // value is changed
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
			name: "should update existing deployment with finalizers & verify absence of drift",
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

	for _, testcase := range tests {
		testcase := testcase                      // pin name to avoid issues due to parallel run of testcases
		t.Run(testcase.name, func(t *testing.T) { //
			ctx := context.Background()
			got, err := Apply(ctx, testcase.toApply)
			if err != nil {
				t.Errorf("expected no error while apply, got: %+v", err)
			}

			// required before invoking drift against old state
			testcase.toCompare.SetResourceVersion(got.GetResourceVersion())
			// verify for difference w.r.t cluster state
			isDrift, diff, err := HasDrifted(ctx, testcase.toCompare)
			if err != nil {
				t.Errorf("expected no error while checking for drift, got: %+v", err)
			}
			if testcase.isDrift != isDrift {
				t.Errorf(
					"expected drift '%t', got '%t': diff '%t': -actual + expected \n%s",
					testcase.isDrift,
					isDrift,
					diff != "",
					diff,
				)
			}
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
		it          string
		deployObj   *appsv1.Deployment
		annotations map[string]string
		labels      map[string]string
		finalizers  []string
		expected    OperationResult
	}
	tests := []*testable{
		{
			it:        "should create the deployment",
			deployObj: deploy.DeepCopy(),
			expected:  OperationResultCreated,
		},
		{
			it:        "should not result in any change",
			deployObj: deploy.DeepCopy(),
			expected:  OperationResultNone,
		},
		{
			it:        "should update the deployment with labels",
			deployObj: deploy.DeepCopy(),
			labels: map[string]string{
				"foo-1": "bar-1",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			it:        "should update the deployment with annotations",
			deployObj: deploy.DeepCopy(),
			annotations: map[string]string{
				"foo-1": "bar-1",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			it:        "should not result in any change since labels & annotations remain same",
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
			it:        "should update the deployment with finalizers",
			deployObj: deploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
				"protect.io/compute",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			it:        "should update the deployment by updating the finalizers",
			deployObj: deploy.DeepCopy(),
			finalizers: []string{
				"protect.io/storage",
			},
			expected: OperationResultUpdatedResourceOnly,
		},
		{
			it:        "should not result in any change since labels, annotations & finalizers remain same",
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
	for _, testcase := range tests {
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
			if len(testcase.finalizers) == 0 {
				testcase.deployObj.SetFinalizers(nil)
			} else {
				testcase.deployObj.SetFinalizers(testcase.finalizers)
			}
		}
		result, err := CreateOrMerge(ctx, klient, scheme.Scheme, testcase.deployObj)
		if err != nil {
			t.Errorf("%s: expected no error got %+v", testcase.it, err)
		}
		if err == nil && result != testcase.expected {
			t.Errorf("%s: expected %q got %q", testcase.it, testcase.expected, result)
		}
	}
}

func TestObjectEqual(t *testing.T) {
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
		observed client.Object
		desired  client.Object
		isEqual  bool
	}
	tests := map[string]testable{
		"observed equals desired deployment": {
			observed: deployObj.DeepCopy(),
			desired:  deployObj.DeepCopy(),
			isEqual:  true,
		},
		"observed is superset of desired deployment": {
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
		"desired is superset of observed deployment": {
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
	for name, testcase := range tests {
		isEqual, err := IsEqual(testcase.observed, testcase.desired)
		if err != nil {
			t.Errorf("%s: expected no error got %+v", name, err)
		}
		if err == nil && isEqual != testcase.isEqual {
			t.Errorf("%s: expected %t got %t", name, testcase.isEqual, isEqual)
		}
	}
}
