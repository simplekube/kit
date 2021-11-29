package checks

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/simplekube/kit/pkg/envutil"
	"github.com/simplekube/kit/pkg/k8s"
	"github.com/simplekube/kit/pkg/pointer"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// refer:
// https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale-walkthrough/
// https://github.com/kubernetes-sigs/prometheus-adapter/blob/master/docs/walkthrough.md
// https://github.com/kubernetes-sigs/prometheus-adapter/blob/master/docs/config-walkthrough.md
// https://github.com/kubernetes-sigs/prometheus-adapter/blob/master/docs/config.md

func DoesHPAWork(ctx context.Context, opts ...k8s.RunOption) error {
	if !envutil.IsEnabled(EnvKeyEnableDoesK8sHPAWork, true) {
		// check is ignore if its disabled
		return nil
	}

	var (
		name      = "does-k8s-hpa-work"
		namespace = envutil.GetOrDefault(EnvKeyE2eSuiteNamespace, "k8s-hpa-testing")
	)

	var (
		lblKey = "e2e-testing/run-id"
		lblVal = fmt.Sprintf("test-%d", rand.Int31()) // unique for every run
	)

	// labels to be set against the resource(s) targeted for testing
	var lbls = map[string]string{
		"e2e-testing/group": "hpa",
		"e2e-testing/name":  "does-hpa-work",
		lblKey:              lblVal,
	}

	// container specifications that remain same across the
	// deployment, replicaset & pod instances
	var containers = []corev1.Container{
		{
			Name:  "php-apache",
			Image: "k8s.gcr.io/hpa-example",
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 80,
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse("500m"),
				},
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
		// {
		// 	// does hpa work for containers minus resources setting? No!
		// 	Name:  "busybox",
		// 	Image: "busybox",
		// 	Command: []string{ // forever running binary
		// 		"/bin/sh",
		// 		"-c",                            // next argument will be read from string & executed
		// 		"while true; do sleep 30; done", // run forever
		// 	},
		// },
	}

	// pod specifications that remain same across the
	// deployment, replicaset & pod instances
	var podSpec = corev1.PodSpec{
		Containers: containers,
	}

	// pod template specifications that remain same across the
	// deployment, replicaset & pod instances
	var podTemplateSpec = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: lbls,
		},
		Spec: podSpec,
	}

	// minimum number of pods to be spawned for the target deployment
	var replicas = pointer.Int32(1)

	// lblSelector specs to map resource with its child resource(s)
	var lblSelector = &metav1.LabelSelector{
		MatchLabels: lbls,
	}

	// selector useful to filter resources with matching labels
	var validatedLblSelector = labels.SelectorFromValidatedSet(
		map[string]string{
			lblKey: lblVal,
		})

	// options to list resources based on matching labels & namespace
	listOpts := []client.ListOption{
		&client.ListOptions{
			LabelSelector: validatedLblSelector,
			Namespace:     namespace,
		},
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
			Selector:             lblSelector,
			Template:             podTemplateSpec,
		},
	}

	var containerPort int32 = 80

	// target service under test
	var svcObj = &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: containerPort,
					TargetPort: intstr.IntOrString{
						IntVal: containerPort,
					},
				},
			},
			Selector: lbls,
		},
	}

	// horizontal pod auto scaler (hpa) settings
	var minHPAReplicas int32 = 1
	var maxHPAReplicas int32 = 10

	// hpa that scales up or down the deployment pods
	var hpaObj = &autoscalingv2beta2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2beta2", // this version provides hpa over custom metrics
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
				Name:       name,
			},
			MinReplicas: pointer.Int32(minHPAReplicas), // scale down to min
			MaxReplicas: maxHPAReplicas,                // scale up to max
			Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
				ScaleDown: &autoscalingv2beta2.HPAScalingRules{ // this is done for quicker testing of scale down
					StabilizationWindowSeconds: pointer.Int32(60), // scale down after 60 seconds of stabilization
				},
			},
			Metrics: []autoscalingv2beta2.MetricSpec{
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceCPU, // hpa based on cpu utilization
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: pointer.Int32(20), // utilization is percentage based
						},
					},
				},
			},
		},
	}

	// load generator that increases CPU utilization of target Pods
	var loadGenPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "load-gen",
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "busybox",
					Image: "busybox",
					Command: []string{ // forever running binary
						"/bin/sh",
						"-c", // next argument will be read from string & executed
						fmt.Sprintf("while sleep 0.01; do wget -q -O- http://%s; done", name), // forever invocation of service
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
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
		&Task{
			It:       "should create & assert the service specifications match the observed state",
			Action:   Create, // creates the resource in K8s cluster
			Resource: svcObj,
			Assert:   Equals, // asserts if observed specs matches the desired specs
		},
		&AssertPodListCount{
			It:            "should assert presence of one pod i.e. replica 1",
			ListOptions:   listOpts,
			ExpectedCount: 1,
		},
		&Task{
			It:       "should create & assert the hpa specifications match the observed state",
			Action:   Create, // create the resource in K8s cluster
			Resource: hpaObj,
			Assert:   Equals, // asserts if observed specs matches the desired specs
		},
		&Task{
			It:       "should create & assert the load gen pod specifications match the observed state",
			Action:   Create, // create the resource in K8s cluster
			Resource: loadGenPod,
			Assert:   Equals, // asserts if observed specs matches the desired specs
		},
		&EventualTask{
			Task: &AssertPodListCount{
				It:            fmt.Sprintf("should assert hpa scale up to %d pods", maxHPAReplicas),
				ListOptions:   listOpts,
				ExpectedCount: int(maxHPAReplicas), // scale up to max replicas
			},
			Timeout: pointer.Duration(360 * time.Second),
		},
		&DeletingTask{
			Resource: loadGenPod,
		},
		&EventualTask{
			Task: &AssertPodListCount{
				It:            fmt.Sprintf("should assert hpa scale down to %d pods", minHPAReplicas),
				ListOptions:   listOpts,
				ExpectedCount: int(minHPAReplicas), // scale down to min replicas
			},
			Timeout: pointer.Duration(360 * time.Second),
		},
	}

	return errors.WithMessage(job.Run(ctx, opts...), "failed to verify if k8s hpa works")
}
