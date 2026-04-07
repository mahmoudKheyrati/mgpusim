package runner

import (
	"path"
	"strings"
	"sync"

	"github.com/sarchlab/akita/v4/tracing"
)

// TraceFilter holds allow/deny rules applied at two levels:
//
//  1. Component-level: ComponentAllowed decides whether to install a tracer on
//     a given component at all.  No hook overhead is paid for denied components.
//
//  2. Task-level: TaskAllowed decides whether an individual task (identified by
//     its Kind and What fields) is forwarded to the underlying tracer.
//
// Rules are evaluated in this order:
//
//   - AllowComponents (whitelist): if non-empty, a component must match at
//     least one pattern or it is excluded.
//   - DenyComponents  (blacklist): if non-empty, a matching component is
//     excluded even if it also matched AllowComponents.
//   - AllowKinds / DenyKinds: same logic applied to Task.Kind.
//   - AllowWhats / DenyWhats: same logic applied to Task.What.
//
// Patterns support the `*` and `?` glob wildcards (via path.Match).
// A pattern without wildcards is treated as a substring match.
//
// Example CLI usage (flags added in flag.go):
//
//	--trace-vis --trace-deny-components="*TLB*,*RDMA*"
//	--trace-vis --trace-allow-kinds="wavefront,pipeline"
//	--trace-vis --trace-deny-whats="fetch"
type TraceFilter struct {
	AllowComponents []string
	DenyComponents  []string
	AllowKinds      []string
	DenyKinds       []string
	AllowWhats      []string
	DenyWhats       []string
}

// HasComponentFilters returns true when at least one component-level rule is set.
func (f *TraceFilter) HasComponentFilters() bool {
	return len(f.AllowComponents) > 0 || len(f.DenyComponents) > 0
}

// HasTaskFilters returns true when at least one task-level rule is set.
func (f *TraceFilter) HasTaskFilters() bool {
	return len(f.AllowKinds) > 0 || len(f.DenyKinds) > 0 ||
		len(f.AllowWhats) > 0 || len(f.DenyWhats) > 0
}

// ComponentAllowed returns true when name passes the component-level rules.
func (f *TraceFilter) ComponentAllowed(name string) bool {
	if len(f.AllowComponents) > 0 && !matchesAny(name, f.AllowComponents) {
		return false
	}
	if len(f.DenyComponents) > 0 && matchesAny(name, f.DenyComponents) {
		return false
	}
	return true
}

// TaskAllowed returns true when the task passes the kind/what rules.
func (f *TraceFilter) TaskAllowed(task tracing.Task) bool {
	if len(f.AllowKinds) > 0 && !containsStr(f.AllowKinds, task.Kind) {
		return false
	}
	if len(f.DenyKinds) > 0 && containsStr(f.DenyKinds, task.Kind) {
		return false
	}
	if len(f.AllowWhats) > 0 && !containsStr(f.AllowWhats, task.What) {
		return false
	}
	if len(f.DenyWhats) > 0 && containsStr(f.DenyWhats, task.What) {
		return false
	}
	return true
}

// FilteringTracer wraps any tracing.Tracer and applies a TraceFilter.
//
// Tasks that fail TaskAllowed are silently dropped.  Correlated StepTask,
// AddMilestone and EndTask calls are suppressed automatically so the
// underlying tracer always sees balanced Start/End pairs.
//
// FilteringTracer is safe for concurrent use (required when --parallel is set).
type FilteringTracer struct {
	underlying    tracing.Tracer
	filter        *TraceFilter
	mu            sync.RWMutex
	acceptedTasks map[string]struct{}
}

// NewFilteringTracer creates a FilteringTracer wrapping underlying.
// If filter has no task-level rules, every task is forwarded unchanged.
func NewFilteringTracer(underlying tracing.Tracer, filter *TraceFilter) *FilteringTracer {
	return &FilteringTracer{
		underlying:    underlying,
		filter:        filter,
		acceptedTasks: make(map[string]struct{}),
	}
}

// StartTask forwards the task if it passes the filter.
func (t *FilteringTracer) StartTask(task tracing.Task) {
	if !t.filter.TaskAllowed(task) {
		return
	}
	t.mu.Lock()
	t.acceptedTasks[task.ID] = struct{}{}
	t.mu.Unlock()
	t.underlying.StartTask(task)
}

// StepTask is forwarded only if the corresponding StartTask was accepted.
func (t *FilteringTracer) StepTask(task tracing.Task) {
	t.mu.RLock()
	_, ok := t.acceptedTasks[task.ID]
	t.mu.RUnlock()
	if !ok {
		return
	}
	t.underlying.StepTask(task)
}

// AddMilestone is forwarded only if the corresponding task was accepted.
func (t *FilteringTracer) AddMilestone(milestone tracing.Milestone) {
	t.mu.RLock()
	_, ok := t.acceptedTasks[milestone.TaskID]
	t.mu.RUnlock()
	if !ok {
		return
	}
	t.underlying.AddMilestone(milestone)
}

// EndTask is forwarded only if the corresponding StartTask was accepted.
// The task ID is removed from the accepted set on completion.
func (t *FilteringTracer) EndTask(task tracing.Task) {
	t.mu.Lock()
	_, ok := t.acceptedTasks[task.ID]
	if ok {
		delete(t.acceptedTasks, task.ID)
	}
	t.mu.Unlock()
	if !ok {
		return
	}
	t.underlying.EndTask(task)
}

// parseTraceFilter constructs a TraceFilter from the CLI flag values.
func parseTraceFilter() *TraceFilter {
	f := &TraceFilter{}
	if *traceAllowComponents != "" {
		f.AllowComponents = splitTrimmed(*traceAllowComponents)
	}
	if *traceDenyComponents != "" {
		f.DenyComponents = splitTrimmed(*traceDenyComponents)
	}
	if *traceAllowKinds != "" {
		f.AllowKinds = splitTrimmed(*traceAllowKinds)
	}
	if *traceDenyKinds != "" {
		f.DenyKinds = splitTrimmed(*traceDenyKinds)
	}
	if *traceAllowWhats != "" {
		f.AllowWhats = splitTrimmed(*traceAllowWhats)
	}
	if *traceDenyWhats != "" {
		f.DenyWhats = splitTrimmed(*traceDenyWhats)
	}
	return f
}

// matchesAny returns true when name matches at least one pattern.
// Patterns containing *, ? or [ are evaluated with path.Match (glob).
// All other patterns are treated as plain substrings.
func matchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?[") {
			matched, err := path.Match(p, name)
			if err == nil && matched {
				return true
			}
		} else if strings.Contains(name, p) {
			return true
		}
	}
	return false
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
