package workflow

import "strings"

// Engine provides workflow transition logic derived from a WorkflowDefinition.
// It pre-computes lookup tables for O(1) transition and gate queries.
type Engine struct {
	def              WorkflowDefinition
	validTransitions map[StateID][]StateID // from -> ordered list of valid to states
	gateForEdge      map[string]*GateDef   // "from|to" -> *GateDef (nil if free)
	states           map[StateID]StateDef
}

// NewEngine builds an Engine from a WorkflowDefinition.
// It panics if the definition references an unknown state in a transition.
func NewEngine(def WorkflowDefinition) *Engine {
	e := &Engine{
		def:              def,
		validTransitions: make(map[StateID][]StateID),
		gateForEdge:      make(map[string]*GateDef),
		states:           make(map[StateID]StateDef),
	}

	// Copy states map.
	for id, s := range def.States {
		e.states[id] = s
	}

	// Build transition and gate lookup tables from the ordered transition list.
	for _, t := range def.Transitions {
		from := StateID(t.From)
		to := StateID(t.To)
		e.validTransitions[from] = append(e.validTransitions[from], to)

		edgeKey := edgeKey(from, to)
		if t.Gate != "" {
			if gateDef, ok := def.Gates[t.Gate]; ok {
				g := gateDef // copy
				e.gateForEdge[edgeKey] = &g
			}
			// If the gate name is not found in Gates map, treat as free (no gate).
		} else {
			e.gateForEdge[edgeKey] = nil
		}
	}

	return e
}

// edgeKey returns a stable map key for a from→to pair.
func edgeKey(from, to StateID) string {
	return string(from) + "|" + string(to)
}

// CanTransition returns true if a transition from→to is defined in the workflow.
func (e *Engine) CanTransition(from, to StateID) bool {
	targets, ok := e.validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// NextStates returns the ordered list of valid next states from the given state.
// Returns nil if the state is terminal or has no outgoing transitions.
func (e *Engine) NextStates(from StateID) []StateID {
	return e.validTransitions[from]
}

// Gate returns the GateDef for the from→to transition, or nil if the transition
// is free (no evidence required) or the transition does not exist.
func (e *Engine) Gate(from, to StateID) *GateDef {
	gate, ok := e.gateForEdge[edgeKey(from, to)]
	if !ok {
		return nil
	}
	return gate
}

// IsTerminal returns true if the state is marked as terminal in the definition.
func (e *Engine) IsTerminal(s StateID) bool {
	if def, ok := e.states[s]; ok {
		return def.Terminal
	}
	return false
}

// IsActiveWork returns true if the state is marked as active work.
func (e *Engine) IsActiveWork(s StateID) bool {
	if def, ok := e.states[s]; ok {
		return def.ActiveWork
	}
	return false
}

// InitialState returns the workflow's configured initial state.
func (e *Engine) InitialState() StateID {
	return e.def.InitialState
}

// States returns all state IDs defined in the workflow.
func (e *Engine) States() []StateID {
	ids := make([]StateID, 0, len(e.states))
	for id := range e.states {
		ids = append(ids, id)
	}
	return ids
}

// StateLabel returns the display label for a state ID.
// Returns the raw state ID string if the state is not found.
func (e *Engine) StateLabel(s StateID) string {
	if def, ok := e.states[s]; ok && def.Label != "" {
		return def.Label
	}
	return string(s)
}

// Definition returns the underlying WorkflowDefinition.
func (e *Engine) Definition() WorkflowDefinition {
	return e.def
}

// IsSkippableFor reports whether the gate at from→to is auto-passable for the
// given feature kind. Returns false if the transition has no gate.
func (e *Engine) IsSkippableFor(from, to StateID, kind string) bool {
	gate := e.Gate(from, to)
	if gate == nil {
		return false
	}
	kindLower := strings.ToLower(kind)
	for _, k := range gate.SkippableFor {
		if strings.ToLower(k) == kindLower {
			return true
		}
	}
	return false
}
