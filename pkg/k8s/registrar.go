package k8s

import (
	"sync"

	"github.com/pkg/errors"
)

type BaseRegistrar struct {
	EntityType EntityType
	Store      map[Key]Runner

	// Keys that follow the insertion order
	orderedEntries []Key

	// all operations should be thread safe & hence should use
	// this mutex
	mu sync.Mutex
}

// compile time check to assert if BaseRegistrar
// implements the interface Registrar
var _ Registrar = (*BaseRegistrar)(nil)

// Type defines the kind of entries that this
// registry can store
func (r *BaseRegistrar) Type() EntityType {
	return r.EntityType
}

// Get the Runner corresponding to the given Key
func (r *BaseRegistrar) Get(key Key) Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Store[key]
}

// GetKeys returns the entry keys based on their
// inserted order
func (r *BaseRegistrar) GetKeys() []Key {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.orderedEntries
}

// GetRunners returns the registered runner(s)
// based on their insertion order
func (r *BaseRegistrar) GetRunners() []Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	var runners []Runner
	for _, key := range r.orderedEntries {
		runners = append(runners, r.Store[key])
	}
	return runners
}

// Register the provided Runner instance to be retrieved
// later
func (r *BaseRegistrar) Register(runner Runner) error {
	regEntry, ok := runner.(RegistrarEntry)
	if !ok {
		return errors.Errorf(
			"failed to register: unsupported runner: want %q",
			r.EntityType,
		)
	}

	if r.EntityType != regEntry.Type() {
		return errors.Errorf(
			"failed to register: type mismatch: want %q: got %q",
			r.EntityType,
			regEntry.Type(),
		)
	}

	if regVal, ok := runner.(Validator); ok {
		if err := regVal.Validate(); err != nil {
			return err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, duplicate := r.Store[regEntry.Key()]; duplicate {
		return errors.Errorf(
			"duplicate runner: registry type %q: runner key %q", regEntry.Type(), regEntry.Key(),
		)
	}
	r.orderedEntries = append(r.orderedEntries, regEntry.Key())
	r.Store[regEntry.Key()] = runner
	return nil
}

// IsRegistered returns true if the provided key has an entry
// in the registry
func (r *BaseRegistrar) IsRegistered(key Key) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, found := r.Store[key]
	return found
}

// gcRegistrar registers garbage collection Job
type gcRegistrar struct {
	*BaseRegistrar
}

// compile time check to assert if gcRegistrar
// implements the interface Registrar
var _ Registrar = (*gcRegistrar)(nil)

// TODO (@amit.das) check if this is the right way to use sync.Once
// default garbage collection registry
var _gcRegistry *gcRegistrar
var _gcRegistryOnce sync.Once

// getDefaultGCRegistry returns the default registry
// for garbage collection Job
func getDefaultGCRegistry() *gcRegistrar {
	if _gcRegistry != nil {
		return _gcRegistry
	}

	// When `.Do` is invoked, if there is an ongoing simultaneous operation,
	// name will block until name has completed. Alternatively if the operation
	// has already completed once before, this call is a no-op and doesn't
	// block.
	_gcRegistryOnce.Do(func() {
		_gcRegistry = &gcRegistrar{
			&BaseRegistrar{
				EntityType: EntityTypeGarbageCollector,
				Store:      map[Key]Runner{},
			}}
	})
	return _gcRegistry
}
