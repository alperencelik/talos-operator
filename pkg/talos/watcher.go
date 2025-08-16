package talos

import (
	"context"
	"fmt"
	"sync"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Watcher is an interface for external watchers.
type Watcher interface {
	// AddWatcher adds a new watcher to the map.
	AddWatcher(name string, ch chan<- reconcile.Result)
	// HasWatcher checks if a watcher for a given name already exists in the watchers map.
	HasWatcher(name string) bool
	// RemoveWatcher removes a watcher from the map.
	RemoveWatcher(name string)
	// StartWatchers starts the watchers for the given object.
	StartWatcher(name string, startWatcherfunc func(ctx context.Context, stopChan <-chan struct{}) (ctrl.Result, error))
	// TriggerReconciliation triggers a reconciliation for a given object.
	TriggerReconciliation(name string, result reconcile.Result)
}

// ExternalWatchers is a struct that holds a map of external watchers.
type ExternalWatchers struct {
	watchers map[string]chan<- reconcile.Result
	mu       sync.Mutex
}

// NewExternalWatchers creates a new ExternalWatchers object.
func NewExternalWatchers() *ExternalWatchers {
	return &ExternalWatchers{
		watchers: make(map[string]chan<- reconcile.Result),
	}
}

// AddWatcher adds a new watcher to the map.
func (ew *ExternalWatchers) AddWatcher(name string, ch chan<- reconcile.Result) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	ew.watchers[name] = ch
}

// HasWatcher checks if a watcher for a given name already exists in the watchers map.
func (ew *ExternalWatchers) HasWatcher(name string) bool {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	_, ok := ew.watchers[name]
	return ok
}

// RemoveWatcher removes a watcher from the map.
func (ew *ExternalWatchers) RemoveWatcher(name string) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	delete(ew.watchers, name)
}

// StartWatcher starts the watchers for the given object.
func (ew *ExternalWatchers) StartWatcher(name string,
	startWatcherfunc func(ctx context.Context, stopChan <-chan struct{}) (ctrl.Result, error)) {

	go func() {
		stopChan := make(chan struct{})
		result, err := startWatcherfunc(context.Background(), stopChan)
		if err != nil {
			fmt.Printf("watcher for %s returned an error: %v\n", name, err)
		}
		if ch, ok := ew.watchers[name]; ok {
			ch <- result
		}
	}()
}

// TriggerReconciliation triggers a reconciliation for a given object.
func (ew *ExternalWatchers) TriggerReconciliation(name string, result reconcile.Result) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	if ch, ok := ew.watchers[name]; ok {
		ch <- result
	}
}
