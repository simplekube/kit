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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
