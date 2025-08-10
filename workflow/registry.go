package workflow

import (
	"errors"
	"sort"
	"sync"
)

// Global registry for named workflows to support optional debugging/diagram export.
var (
	regMu       sync.RWMutex
	workflowReg = make(map[string]*Workflow)
)

// Register adds a workflow under a name. Returns error if the name already exists or workflow is nil.
func Register(name string, wf *Workflow) error {
	if wf == nil {
		return errors.New("nil workflow")
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := workflowReg[name]; exists {
		return errors.New("workflow already registered")
	}
	workflowReg[name] = wf
	return nil
}

// Get returns a registered workflow by name.
func Get(name string) (*Workflow, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	wf, ok := workflowReg[name]
	return wf, ok
}

// List returns sorted workflow names.
func List() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	names := make([]string, 0, len(workflowReg))
	for k := range workflowReg {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
