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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
