package setup

import (
	"context"
	"fmt"

	"github.com/simplekube/kit/pkg/k8s"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Tasks = k8s.Tasks
type Task = k8s.Task

var (
	Get           = k8s.ActionTypeGet
	Create        = k8s.ActionTypeCreate
	Delete        = k8s.ActionTypeDelete
	CreateOrMerge = k8s.ActionTypeCreateOrMerge
)

var (
	Equals     = k8s.AssertTypeIsEquals
	IsNotFound = k8s.AssertIsNotFound
)

type TestEnv struct {
	name      string
	namespace string
}

func New(name string) *TestEnv {
	return &TestEnv{
		name: name,
	}
}

func (t *TestEnv) GetNamespace() string {
	return t.namespace
}

func (t *TestEnv) Setup(ctx context.Context) (err error) {
	tasks := Tasks{
		{
			It:     "should create suite namespace",
			Action: Create,
			Resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-", t.name),
					Labels: map[string]string{
						"e2e-testing/env-name": t.name, // This is not same as namespace name
						"e2e-testing":          "yes",  // This is used in Makefile
					},
				},
			},
			PostAction: func(obj client.Object) error {
				ns, _ := obj.(*corev1.Namespace)
				t.namespace = ns.Name // fetch namespace value at runtime
				return nil
			},
		},
		{
			It:     "should assert presence of suite namespace",
			Action: Get,
			Resource: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			PreAction: func(obj client.Object) error {
				ns, _ := obj.(*corev1.Namespace)
				ns.Name = t.namespace // lazy setting since value is fetched at runtime
				return nil
			},
			Assert: Equals,
		},
	}

	return errors.WithMessage(tasks.Run(ctx), "failed to setup testing env")
}

func (t *TestEnv) Teardown(ctx context.Context) error {
	return errors.WithMessage(k8s.Teardown(ctx), "failed to teardown testing env")
}
