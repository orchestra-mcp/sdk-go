package workflow

import (
	"strings"
	"sync"
	"time"

	"github.com/orchestra-mcp/sdk-go/globaldb"
	"github.com/orchestra-mcp/sdk-go/helpers"
)

// EngineResolver resolves per-project workflow engines from the database.
// It caches engines per project and invalidates them when the DB record changes.
// If no DB workflow exists for a project, the fallback (default) engine is used.
type EngineResolver struct {
	fallback *Engine

	mu    sync.RWMutex
	cache map[string]*cachedEngine
}

type cachedEngine struct {
	engine    *Engine
	updatedAt string // DB record's updated_at for cache invalidation
	fetchedAt time.Time
}

// cacheTTL defines how long we trust a cached engine before re-checking the DB.
const cacheTTL = 30 * time.Second

// NewResolver creates an EngineResolver with the given fallback engine.
func NewResolver(fallback *Engine) *EngineResolver {
	return &EngineResolver{
		fallback: fallback,
		cache:    make(map[string]*cachedEngine),
	}
}

// Resolve returns the workflow engine for the given project.
// It checks the database for a project-specific workflow, caches the result,
// and falls back to the default engine if no DB workflow exists.
func (r *EngineResolver) Resolve(projectID string) *Engine {
	if projectID == "" {
		return r.fallback
	}

	// Fast path: check cache under read lock.
	r.mu.RLock()
	if cached, ok := r.cache[projectID]; ok {
		if time.Since(cached.fetchedAt) < cacheTTL {
			r.mu.RUnlock()
			return cached.engine
		}
	}
	r.mu.RUnlock()

	// Slow path: query DB and rebuild engine under write lock.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if cached, ok := r.cache[projectID]; ok {
		if time.Since(cached.fetchedAt) < cacheTTL {
			return cached.engine
		}
	}

	rec, err := globaldb.GetProjectWorkflow(projectID)
	if err != nil || rec == nil {
		// No DB workflow — auto-seed the default for migration of existing projects.
		seeded, seedErr := SeedDefaultWorkflow(projectID)
		if seedErr != nil || seeded == nil {
			// Seeding failed or already exists — use fallback.
			r.cache[projectID] = &cachedEngine{
				engine:    r.fallback,
				fetchedAt: time.Now(),
			}
			return r.fallback
		}
		rec = seeded
	}

	// Convert DB record to WorkflowDefinition.
	def := recordToDefinition(rec)
	eng := NewEngine(def)

	r.cache[projectID] = &cachedEngine{
		engine:    eng,
		updatedAt: rec.UpdatedAt,
		fetchedAt: time.Now(),
	}
	return eng
}

// Invalidate removes the cached engine for a project, forcing a re-fetch.
func (r *EngineResolver) Invalidate(projectID string) {
	r.mu.Lock()
	delete(r.cache, projectID)
	r.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (r *EngineResolver) InvalidateAll() {
	r.mu.Lock()
	r.cache = make(map[string]*cachedEngine)
	r.mu.Unlock()
}

// Fallback returns the default engine.
func (r *EngineResolver) Fallback() *Engine {
	return r.fallback
}

// SeedDefaultWorkflow creates the default workflow record for a project if none exists.
// Returns the created record, or nil if a workflow already exists.
// Safe to call multiple times — it's a no-op when a workflow already exists.
func SeedDefaultWorkflow(projectID string) (*globaldb.WorkflowRecord, error) {
	existing, err := globaldb.GetProjectWorkflow(projectID)
	if err == nil && existing != nil {
		return nil, nil
	}

	def := DefaultDefinition()

	states := make(map[string]globaldb.WorkflowStateRec, len(def.States))
	for id, s := range def.States {
		states[string(id)] = globaldb.WorkflowStateRec{
			Label:      s.Label,
			Terminal:   s.Terminal,
			ActiveWork: s.ActiveWork,
		}
	}

	transitions := make([]globaldb.WorkflowTransitionRec, len(def.Transitions))
	for i, t := range def.Transitions {
		transitions[i] = globaldb.WorkflowTransitionRec{
			From: t.From,
			To:   t.To,
			Gate: t.Gate,
		}
	}

	gates := make(map[string]globaldb.WorkflowGateRec, len(def.Gates))
	for id, g := range def.Gates {
		gates[id] = globaldb.WorkflowGateRec{
			Label:           g.Label,
			RequiredSection: g.RequiredSection,
			FilePatterns:    g.FilePatterns,
			DocsFolder:      g.DocsFolder,
			SkippableFor:    g.SkippableFor,
		}
	}

	rec := &globaldb.WorkflowRecord{
		ID:           helpers.NewWorkflowID(),
		ProjectID:    projectID,
		Name:         def.Name,
		Description:  def.Description,
		InitialState: string(def.InitialState),
		IsDefault:    true,
		States:       states,
		Transitions:  transitions,
		Gates:        gates,
	}

	if err := globaldb.CreateWorkflowRecord(rec); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, nil
		}
		return nil, err
	}

	return rec, nil
}

// recordToDefinition converts a globaldb.WorkflowRecord to a WorkflowDefinition.
func recordToDefinition(rec *globaldb.WorkflowRecord) WorkflowDefinition {
	states := make(map[StateID]StateDef, len(rec.States))
	for id, s := range rec.States {
		states[StateID(id)] = StateDef{
			Label:      s.Label,
			Terminal:   s.Terminal,
			ActiveWork: s.ActiveWork,
		}
	}

	transitions := make([]TransitionDef, len(rec.Transitions))
	for i, t := range rec.Transitions {
		transitions[i] = TransitionDef{
			From: t.From,
			To:   t.To,
			Gate: t.Gate,
		}
	}

	gates := make(map[string]GateDef, len(rec.Gates))
	for id, g := range rec.Gates {
		gates[id] = GateDef{
			Label:           g.Label,
			RequiredSection: g.RequiredSection,
			FilePatterns:    g.FilePatterns,
			DocsFolder:      g.DocsFolder,
			SkippableFor:    g.SkippableFor,
		}
	}

	return WorkflowDefinition{
		Name:         rec.Name,
		Description:  rec.Description,
		InitialState: StateID(rec.InitialState),
		States:       states,
		Transitions:  transitions,
		Gates:        gates,
	}
}
