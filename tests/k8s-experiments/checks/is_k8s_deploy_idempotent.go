package checks

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/simplekube/kit/pkg/envutil"
	"github.com/simplekube/kit/pkg/k8s"
	"github.com/simplekube/kit/pkg/pointer"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsK8sDeploymentIdempotent(ctx context.Context, opts ...k8s.RunOption) error {
	if !envutil.IsEnabled(EnvKeyEnableIsK8sDeployIdempotent, true) {
		// check is ignore if its disabled
		return nil
	}

	var (
		name      = "is-deploy-idempotent"
		namespace = envutil.GetOrDefault(EnvKeyE2eSuiteNamespace, "k8s-deploy-testing")
		finalizer = "protect/is-deploy-idempotent"
	)

	var (
		lblKey = "e2e-testing/unique"
		lblVal = fmt.Sprintf("deploy-%d", rand.Int31()) // unique for every run
	)

	// target namespace under test
	var nsObj = &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	// target deployment under test
	var deployObj = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			RevisionHistoryLimit: pointer.Int32(0), // no old replica sets
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo":  "bar",
					lblKey: lblVal,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo":  "bar",
						lblKey: lblVal, // is set against the ReplicaSet & Pod
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

	var (
		deployResourceVersion     string
		replicasetResourceVersion string
	)

	var lblSelector = labels.SelectorFromValidatedSet(
		map[string]string{
			lblKey: lblVal,
		})

	listOpts := []client.ListOption{
		&client.ListOptions{
			LabelSelector: lblSelector,
			Namespace:     namespace,
		},
	}

	job := Job{
		&Task{
			It:       "should upsert & assert the namespace specifications match the observed state",
			Action:   CreateOrMerge,
			Resource: nsObj,
			Assert:   Equals,
		},
		&Task{
			It:       "should create, capture resource version & assert the deployment specifications match the observed state",
			Action:   Create,
			Resource: deployObj,
			PostAction: func(obj client.Object) error {
				deploy, _ := obj.(*appsv1.Deployment)
				deployResourceVersion = deploy.GetResourceVersion()
				return nil
			},
			Assert: Equals,
		},
		&ListingTask{
			It:          "should capture resource version of corresponding replicaset",
			Resource:    &appsv1.ReplicaSetList{},
			ListOptions: listOpts,
			PostAction: func(obj client.ObjectList) error {
				rsList, _ := obj.(*appsv1.ReplicaSetList)
				if len(rsList.Items) != 1 {
					return errors.Errorf("expected 1 replicaset got %d", len(rsList.Items))
				}
				replicasetResourceVersion = rsList.Items[0].GetResourceVersion()
				return nil
			},
		},
		&Task{
			It:       "should add finalizers to deployment & assert change in its resource version",
			Action:   CreateOrMerge,
			Resource: deployObj,
			PreAction: func(obj client.Object) error {
				// mutate the provided object
				d, _ := obj.(*appsv1.Deployment)
				d.SetFinalizers([]string{finalizer})
				return nil
			},
			PostAction: func(obj client.Object) error {
				d, _ := obj.(*appsv1.Deployment)
				if len(d.GetFinalizers()) != 1 {
					return errors.Errorf("expected 1 finalizers got %d", len(d.GetFinalizers()))
				}
				if d.GetResourceVersion() == deployResourceVersion {
					return errors.Errorf("expected different resource versions got same: %s", deployResourceVersion)
				}
				// capture latest resource version
				deployResourceVersion = d.GetResourceVersion()
				return nil
			},
		},
		&ListingTask{
			It:          "should assert no change in resource version of corresponding replicaset",
			Resource:    &appsv1.ReplicaSetList{},
			ListOptions: listOpts,
			PostAction: func(obj client.ObjectList) error {
				rsList, _ := obj.(*appsv1.ReplicaSetList)
				if len(rsList.Items) != 1 {
					return errors.Errorf("expected 1 replicaset got %d", len(rsList.Items))
				}
				if replicasetResourceVersion != rsList.Items[0].GetResourceVersion() {
					return errors.Errorf(
						"expected no change in resource version got different: prev %s: new %s",
						replicasetResourceVersion,
						rsList.Items[0].GetResourceVersion(),
					)
				}
				// capture latest resource version
				replicasetResourceVersion = rsList.Items[0].GetResourceVersion()
				return nil
			},
		},
		&Task{
			It:       "should update the deployment's spec.replicas to 2 & assert given specifications match the observed state",
			Resource: deployObj,
			Action:   CreateOrMerge,
			PreAction: func(obj client.Object) error {
				// mutate the provided object
				d, _ := obj.(*appsv1.Deployment)
				d.Spec.Replicas = pointer.Int32(2)
				return nil
			},
			Assert: Equals,
		},
		&Task{
			It:       "should assert change in deployment's resource version",
			Resource: deployObj,
			Action:   Get,
			PostAction: func(obj client.Object) error {
				d, _ := obj.(*appsv1.Deployment)
				if deployResourceVersion == d.GetResourceVersion() {
					return errors.New("expected different resource versions got same")
				}
				// capture latest resource version observed in cluster
				deployResourceVersion = d.GetResourceVersion()
				return nil
			},
		},
		&ListingTask{
			It:          "should assert change in resource version of corresponding replicaset",
			Resource:    &appsv1.ReplicaSetList{},
			ListOptions: listOpts,
			PostAction: func(obj client.ObjectList) error {
				rsList, _ := obj.(*appsv1.ReplicaSetList)
				if len(rsList.Items) != 1 {
					return errors.Errorf("expected 1 replica set got %d", len(rsList.Items))
				}
				if replicasetResourceVersion == rsList.Items[0].GetResourceVersion() {
					return errors.New("expected different resource versions got same")
				}
				// capture latest resource version
				replicasetResourceVersion = rsList.Items[0].GetResourceVersion()
				return nil
			},
		},
		&FinalizersRemovalTask{
			Resource: deployObj,
		},
		&Task{
			It:       "should assert deployment has change in its resource version",
			Action:   Get,
			Resource: deployObj,
			PostAction: func(obj client.Object) error {
				d, _ := obj.(*appsv1.Deployment)
				if d.GetResourceVersion() == deployResourceVersion {
					return errors.Errorf("expected different resource versions got same: %s", deployResourceVersion)
				}
				return nil
			},
		},
		&ListingTask{
			It:          "should assert no change in resource version of corresponding replicaset",
			Resource:    &appsv1.ReplicaSetList{},
			ListOptions: listOpts,
			PostAction: func(obj client.ObjectList) error {
				rsList, _ := obj.(*appsv1.ReplicaSetList)
				if len(rsList.Items) != 1 {
					return errors.Errorf("expected 1 replicaset got %d", len(rsList.Items))
				}
				if replicasetResourceVersion != rsList.Items[0].GetResourceVersion() {
					return errors.Errorf(
						"expected same resource version got different: prev %q: new %q",
						replicasetResourceVersion,
						rsList.Items[0].GetResourceVersion(),
					)
				}
				return nil
			},
		},
	}

	return errors.WithMessage(job.Run(ctx, opts...), "failed to verify if k8s deployment is idempotent")
}
