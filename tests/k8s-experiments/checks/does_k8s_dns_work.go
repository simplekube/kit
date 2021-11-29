package checks

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/simplekube/kit/pkg/envutil"
	"github.com/simplekube/kit/pkg/k8s"
	"github.com/simplekube/kit/pkg/pointer"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// refer: https://github.com/kubernetes/examples/tree/master/staging/cluster-dns
// issue: https://github.com/docker/for-win/issues/1534

// TODO (@amit.das) Make this work!!!
func DoesK8sDNSWork(ctx context.Context, opts ...k8s.RunOption) error {
	// TODO (@amit.das)
	//  enable by default when this check is fixed to work
	if !envutil.IsEnabled(EnvKeyEnableDoesK8sDNSWork, false) {
		// check is ignore if its disabled
		return nil
	}

	var (
		name      = "does-k8s-dns-work"
		namespace = envutil.GetOrDefault(EnvKeyE2eSuiteNamespace, "k8s-dns-testing")
	)

	var (
		lblKey = "e2e-testing/unique"
		lblVal = fmt.Sprintf("deploy-%d", rand.Int31()) // unique for every run
	)

	// labels to be set against the resource(s) targeted for testing
	var lbls = map[string]string{
		"name": name,
		lblKey: lblVal,
	}

	var frontendLbls = lbls
	frontendLbls["frontend"] = "true"

	var backendLbls = lbls
	backendLbls["backend"] = "true"

	// port that is used to expose the service endpoint
	var containerPort int32 = 8000

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

	// container specifications that remain same across the
	// deployment, replicaset & pod instances
	var containers = []corev1.Container{
		{
			Name:  "dns-backend",
			Image: "k8s.gcr.io/example-dns-backend:v1",
			Ports: []corev1.ContainerPort{
				{
					Name:          "backend-port",
					ContainerPort: containerPort,
				},
			},
		},
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

	// number of pods to be spawned in the cluster
	var replicas = pointer.Int32(1)

	// selector specs to map resource with its child resource(s)
	var selector = &metav1.LabelSelector{
		MatchLabels: lbls,
	}

	// target deployment under test
	var dnsBackendDeployObj = &appsv1.Deployment{
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

	// job is a set of Kubernetes tasks that represents the scenario
	// we want to verify
	job := Job{
		&UpsertThenAssertEquals{
			It:       "should upsert & then assert the backend namespace specifications",
			Resource: nsObj,
		},
		&CreateThenAssertEquals{
			It:       "should create & then assert the backend deployment specifications",
			Resource: dnsBackendDeployObj,
		},
		&CreateThenAssertEquals{
			It:       "should create & then assert the backend service specifications",
			Resource: svcObj,
		},
		&Custom{
			It: "should pretty print the backend dns specifications",
		},
		&Task{
			It:     "should pretty print the coredns configuration",
			Action: Get,
			Resource: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "coredns",
					Namespace: "kube-system",
				},
			},
			PostAction: func(obj client.Object) error {
				return errors.New("not implemented")
			},
		},
		&Custom{
			// this custom logic running within this pod acts
			// as the frontend that consumes the backend i.e. service
			It: "should assert dns resolves across namespace",
			Action: func(_ context.Context, _ ...RunOption) error {
				c := &http.Client{
					Timeout: 15 * time.Second,
				}
				resp, err := c.Get(fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", name, namespace, containerPort))
				if err != nil {
					return errors.Wrap(err, "failed to communicate with service")
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return errors.Wrap(err, "failed to read service response")
				}
				if !strings.Contains(string(body), "Hello World!") {
					return errors.Errorf("expected service response to contain 'Hello World' got %q", string(body))
				}
				return nil
			},
		},
	}

	return errors.WithMessage(job.Run(ctx, opts...), "failed to verify if k8s dns works")
}
