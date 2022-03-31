package k8s

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This file makes use of functional options pattern
// credit: https://github.com/uber-go/guide/blob/master/style.md

type RunOption interface {
	// ApplyTo sets the provided RunOption instance
	ApplyTo(RunOption) error
}

// RunOptions defines standard runtime options for a Runner
type RunOptions struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
	Scheme    *runtime.Scheme

	// Desired state field(s) with null or empty value(s) are considered
	// as valid during Upsert operation
	AcceptNullFieldValuesDuringUpsert *bool

	// SetFinalizersToNullDuringUpsert when true will set the target's
	// finalizers to nil during Upsert operation
	SetFinalizersToNullDuringUpsert *bool
}

// compile time check to assert if the structure
// RunOptions implements the interface RunOption
var _ RunOption = (*RunOptions)(nil)

// ApplyTo applies properties from the method receiver
// to the provided target instance
func (o *RunOptions) ApplyTo(target RunOption) error {
	if o == nil {
		return errors.Errorf("nil receiver options")
	}
	if target == nil {
		return errors.Errorf("nil target options")
	}
	targetObj, ok := target.(*RunOptions)
	if !ok {
		return errors.Errorf("invalid options type: want 'RunOptions' got %T", target)
	}
	if o.Client != nil {
		targetObj.Client = o.Client
	}
	if o.Clientset != nil {
		targetObj.Clientset = o.Clientset
	}
	if o.Scheme != nil {
		targetObj.Scheme = o.Scheme
	}
	if o.AcceptNullFieldValuesDuringUpsert != nil {
		targetObj.AcceptNullFieldValuesDuringUpsert = o.AcceptNullFieldValuesDuringUpsert
	}
	if o.SetFinalizersToNullDuringUpsert != nil {
		targetObj.SetFinalizersToNullDuringUpsert = o.SetFinalizersToNullDuringUpsert
	}
	return nil
}

// ApplyRunOptionsToTarget builds the target instance from the list of
// provided options
func ApplyRunOptionsToTarget(target *RunOptions, options ...RunOption) error {
	if target == nil {
		return errors.New("nil target to build options")
	}
	for _, o := range options {
		err := o.ApplyTo(target)
		if err != nil {
			return err
		}
	}
	return nil
}

// FromRunOptions creates a new options instance assembled from the
// provided list of options
func FromRunOptions(options ...RunOption) (*RunOptions, error) {
	var target RunOptions
	err := ApplyRunOptionsToTarget(&target, options...)
	if err != nil {
		return nil, err
	}
	return &target, nil
}
