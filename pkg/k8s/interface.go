package k8s

import (
	"context"
)

const ManagerName = "wonderland-governance-manager"

// Key is used to look up any entity. It is advisable
// to make name human-readable.
type Key string

// EntityType defines the type of Entity
type EntityType string

const (
	EntityTypeGarbageCollector EntityType = "gc"
)

// Registrar exposes the contract(s) to store & retrieve
// Job
type Registrar interface {
	// Get fetches the Runner instance corresponding to
	// the provided key
	Get(key Key) Runner

	// GetKeys fetches all Keys found in the registry
	GetKeys() []Key

	// GetRunners fetches all Runner instances found in
	// the registry
	GetRunners() []Runner

	// Type of entities that this registrar is supposed
	// to store
	Type() EntityType

	// Register the provided Runner
	Register(s Runner) error

	// IsRegistered returns true if the provided key
	// was Store earlier
	IsRegistered(key Key) bool
}

// RegistrarEntry defines those entries that can be stored
// in a registrar
type RegistrarEntry interface {
	// Key of the entry that uniquely identifies name
	// amongst the entries present in the registrar
	Key() Key

	// Type of the entry. Entry's type should match
	// registrar's type.
	Type() EntityType
}

// Validator exposes the contract to validate a Runner
type Validator interface {
	Validate() error
}

// Runner exposes contract(s) to run an entity
type Runner interface {
	Run(ctx context.Context, opts ...RunOption) error
}

type noop struct{}

func Noop() *noop {
	return &noop{}
}

// compile time check to AssertType if the structure
// noop implements the interface Runner
var _ Runner = (*noop)(nil)

// compile time check to AssertType if the structure
// noop implements the interface Validator
var _ Validator = (*noop)(nil)

// compile time check to AssertType if the structure
// noop implements the interface RegistrarEntry
var _ RegistrarEntry = (*noop)(nil)

// compile time check to AssertType if the structure
// noop implements the interface Runner
var _ Runner = (*noop)(nil)

func (n *noop) Key() Key {
	return "noop"
}

func (n *noop) Type() EntityType {
	return "noop"
}

func (n *noop) Validate() error {
	return nil
}

func (n *noop) Run(ctx context.Context, opts ...RunOption) error {
	return nil
}

// ActionType defines the ActionType performed in the step
type ActionType string

const (
	// ActionTypeCreate defines a Kubernetes resource create operation
	ActionTypeCreate ActionType = "Create"

	// ActionTypeCreateOrMerge defines a Kubernetes resource create or
	// merge operation
	ActionTypeCreateOrMerge ActionType = "CreateOrMerge"

	// ActionTypeGet defines a Kubernetes resource get operation
	ActionTypeGet ActionType = "Get"

	// ActionTypeDelete defines a Kubernetes resource delete operation
	ActionTypeDelete ActionType = "Delete"

	// ActionTypeUpdate defines a Kubernetes resource update operation
	ActionTypeUpdate ActionType = "Update"
)

// AssertType defines the assertion performed in the step
type AssertType string

const (
	// AssertTypeIsEquals defines Equals assertion
	AssertTypeIsEquals AssertType = "Equals"

	// AssertTypeIsNotEquals defines NotEquals assertion
	AssertTypeIsNotEquals AssertType = "NotEquals"

	// AssertTypeIsFound defines IsFound assertion
	AssertTypeIsFound AssertType = "IsFound"

	// AssertTypeIsNotFound defines IsNotFound assertion
	AssertTypeIsNotFound AssertType = "IsNotFound"

	// AssertTypeIsNoop defines a no operation assertion
	//
	// This is the default assertion type when assert
	// is not set
	AssertTypeIsNoop AssertType = "Noop"

	// AssertTypeIsCustom defines a custom assertion
	AssertTypeIsCustom AssertType = "Custom"
)
