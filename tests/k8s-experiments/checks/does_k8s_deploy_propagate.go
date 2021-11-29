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

func DoesK8sDeploymentPropagate(ctx context.Context, opts ...k8s.RunOption) error {
	if !envutil.IsEnabled(EnvKeyEnableDoesK8sDeployPropagate, true) {
		// check is ignore if its disabled
		return nil
	}

	var (
		name      = "does-k8s-deploy-propagate"
		namespace = envutil.GetOrDefault(EnvKeyE2eSuiteNamespace, "k8s-deploy-testing")
	)

	var (
		lblKey = "e2e-testing/unique"
		lblVal = fmt.Sprintf("deploy-%d", rand.Int31()) // unique for every run
	)

	// labels to be set against the resource(s) targeted for testing
	var lbls = map[string]string{
		"foo":  "bar",
		lblKey: lblVal,
	}

	// container specifications that remain same across the
	// deployment, replicaset & pod instances
	var containers = []corev1.Container{
		{
			Name:  "busybox",
			Image: "busybox",
			Args: []string{
				"dont", "do", "anything",
			},
			Env: []corev1.EnvVar{
				{
					Name:  "MY_ENV_ONE",
					Value: "my-val-1",
				},
				{
					Name: "MY_NODE_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
				{
					Name: "MY_NODE_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
			},
			Ports: []corev1.ContainerPort{
				{
					Name:          "eighty-port",
					ContainerPort: 80,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
	}

	// pod specifications that remain same across the
	// deployment, replicaset & pod instances
	var podSpec = corev1.PodSpec{
		Containers:    containers,
		RestartPolicy: corev1.RestartPolicyAlways,
	}

	// pod template specifications that remain same across the
	// deployment, replicaset & pod instances
	var podTemplateSpec = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: lbls,
		},
		Spec: podSpec,
	}

	// number of pods to be spawned in the cluster
	var replicas = pointer.Int32(1)

	// selector specs to map resource with its child resource(s)
	var selector = &metav1.LabelSelector{
		MatchLabels: lbls,
	}

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
			Replicas:             replicas,
			RevisionHistoryLimit: pointer.Int32(0), // no old replica sets
			Selector:             selector,
			Template:             podTemplateSpec,
		},
	}

	var observedReplicaSetName string
	var observedPodName string

	// selector useful to filter resources with matching labels
	var lblSelector = labels.SelectorFromValidatedSet(
		map[string]string{
			lblKey: lblVal,
		})

	// options to list resources based on matching labels & namespace
	listOpts := []client.ListOption{
		&client.ListOptions{
			LabelSelector: lblSelector,
			Namespace:     namespace,
		},
	}

	// job is a set of Kubernetes tasks that represents the scenario
	// we want to verify
	job := Job{
		&Task{
			It:       "should upsert & assert the namespace specifications match the observed state",
			Action:   CreateOrMerge, // create if not available in cluster or merge to observed state
			Resource: nsObj,
			Assert:   Equals, // asserts if observed specs matches the desired specs
		},
		&Task{
			It:       "should create & assert the deployment specifications match the observed state",
			Action:   Create, // creates the resource in K8s cluster
			Resource: deployObj,
			Assert:   Equals, // asserts if observed specs matches the desired specs
		},
		&ListingTask{
			It:          "should assert presence of corresponding replicaset",
			Resource:    &appsv1.ReplicaSetList{},
			ListOptions: listOpts,
			PostAction: func(obj client.ObjectList) error {
				rsList, _ := obj.(*appsv1.ReplicaSetList)
				if len(rsList.Items) != 1 {
					return errors.Errorf("expected 1 replicaset got %d", len(rsList.Items))
				}
				observedReplicaSetName = rsList.Items[0].Name // fetch name of the ReplicaSet at runtime
				return nil
			},
		},
		&Task{
			It:     "should assert the replicaset specifications match the observed state",
			Action: Get, // fetches the resource from the K8s cluster
			Resource: &appsv1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels:    lbls,
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: replicas,
					Selector: selector,
					Template: podTemplateSpec,
				},
				Status: appsv1.ReplicaSetStatus{
					Replicas: 1,
				},
			},
			PreAction: func(obj client.Object) error {
				rs, _ := obj.(*appsv1.ReplicaSet)
				rs.SetName(observedReplicaSetName) // lazy setting since value is fetched at runtime
				return nil
			},
			Assert: Equals, // asserts if observed specs matches the desired specs
		},
		&ListingTask{
			It:          "should assert presence of corresponding pod",
			Resource:    &corev1.PodList{},
			ListOptions: listOpts,
			PostAction: func(obj client.ObjectList) error {
				podList, _ := obj.(*corev1.PodList)
				if len(podList.Items) != 1 {
					return errors.Errorf("expected 1 pod got %d", len(podList.Items))
				}
				observedPodName = podList.Items[0].Name // fetch name of the Pod at runtime
				return nil
			},
		},
		&Task{
			It:     "should assert the pod specifications match the observed state",
			Action: Get, // fetches the resource from the K8s cluster
			Resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "core/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels:    lbls,
				},
				Spec: podSpec,
			},
			PreAction: func(obj client.Object) error {
				p, _ := obj.(*corev1.Pod)
				p.SetName(observedPodName) // lazy setting since value is fetched at runtime
				return nil
			},
			Assert: Equals, // asserts if observed specs matches the desired specs
		},
	}

	return errors.WithMessage(job.Run(ctx, opts...),
		"failed to verify if k8s deployment specs propagates across replicaset & pod")
}
