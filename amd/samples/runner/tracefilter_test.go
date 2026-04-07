package runner

import (
	"testing"

	"github.com/sarchlab/akita/v4/tracing"
)

// recordingTracer records every call for inspection in tests.
type recordingTracer struct {
	startCalls     []tracing.Task
	stepCalls      []tracing.Task
	milestoneCalls []tracing.Milestone
	endCalls       []tracing.Task
}

func (r *recordingTracer) StartTask(t tracing.Task)         { r.startCalls = append(r.startCalls, t) }
func (r *recordingTracer) StepTask(t tracing.Task)          { r.stepCalls = append(r.stepCalls, t) }
func (r *recordingTracer) AddMilestone(m tracing.Milestone) { r.milestoneCalls = append(r.milestoneCalls, m) }
func (r *recordingTracer) EndTask(t tracing.Task)           { r.endCalls = append(r.endCalls, t) }

func makeTask(id, kind, what string) tracing.Task {
	return tracing.Task{ID: id, Kind: kind, What: what}
}

func makeMilestone(taskID string) tracing.Milestone {
	return tracing.Milestone{TaskID: taskID}
}

// --- TraceFilter.ComponentAllowed ---

func TestComponentAllowed_NoRules(t *testing.T) {
	f := &TraceFilter{}
	if !f.ComponentAllowed("GPU0.CU1") {
		t.Fatal("expected allowed when no rules set")
	}
}

func TestComponentAllowed_AllowList_Match(t *testing.T) {
	f := &TraceFilter{AllowComponents: []string{"*CU*"}}
	if !f.ComponentAllowed("GPU0.CU1") {
		t.Fatal("expected allowed via glob allow-list")
	}
}

func TestComponentAllowed_AllowList_NoMatch(t *testing.T) {
	f := &TraceFilter{AllowComponents: []string{"*Cache*"}}
	if f.ComponentAllowed("GPU0.CU1") {
		t.Fatal("expected denied when not in allow-list")
	}
}

func TestComponentAllowed_DenyList_Match(t *testing.T) {
	f := &TraceFilter{DenyComponents: []string{"*TLB*"}}
	if f.ComponentAllowed("GPU0.TLB0") {
		t.Fatal("expected denied via deny-list")
	}
}

func TestComponentAllowed_DenyList_NoMatch(t *testing.T) {
	f := &TraceFilter{DenyComponents: []string{"*TLB*"}}
	if !f.ComponentAllowed("GPU0.CU1") {
		t.Fatal("expected allowed when not in deny-list")
	}
}

func TestComponentAllowed_SubstringMatch(t *testing.T) {
	// Plain string (no wildcards) uses substring matching.
	f := &TraceFilter{DenyComponents: []string{"TLB"}}
	if f.ComponentAllowed("GPU0.TLB0") {
		t.Fatal("expected denied via substring deny")
	}
	if !f.ComponentAllowed("GPU0.CU1") {
		t.Fatal("expected allowed when substring not present")
	}
}

// --- TraceFilter.TaskAllowed ---

func TestTaskAllowed_NoRules(t *testing.T) {
	f := &TraceFilter{}
	if !f.TaskAllowed(makeTask("t1", "wavefront", "")) {
		t.Fatal("expected allowed when no task rules")
	}
}

func TestTaskAllowed_AllowKinds_Match(t *testing.T) {
	f := &TraceFilter{AllowKinds: []string{"wavefront", "pipeline"}}
	if !f.TaskAllowed(makeTask("t1", "wavefront", "")) {
		t.Fatal("expected allowed when kind in allow list")
	}
}

func TestTaskAllowed_AllowKinds_NoMatch(t *testing.T) {
	f := &TraceFilter{AllowKinds: []string{"wavefront"}}
	if f.TaskAllowed(makeTask("t1", "req_in", "")) {
		t.Fatal("expected denied when kind not in allow list")
	}
}

func TestTaskAllowed_DenyKinds(t *testing.T) {
	f := &TraceFilter{DenyKinds: []string{"req_in", "req_out"}}
	if f.TaskAllowed(makeTask("t1", "req_in", "")) {
		t.Fatal("expected denied by DenyKinds")
	}
	if !f.TaskAllowed(makeTask("t2", "wavefront", "")) {
		t.Fatal("expected allowed when kind not in deny list")
	}
}

func TestTaskAllowed_AllowWhats(t *testing.T) {
	f := &TraceFilter{AllowWhats: []string{"VALU", "VMem"}}
	if !f.TaskAllowed(makeTask("t1", "inst", "VALU")) {
		t.Fatal("expected allowed when What in allow list")
	}
	if f.TaskAllowed(makeTask("t2", "inst", "Scalar")) {
		t.Fatal("expected denied when What not in allow list")
	}
}

func TestTaskAllowed_DenyWhats(t *testing.T) {
	f := &TraceFilter{DenyWhats: []string{"fetch"}}
	if f.TaskAllowed(makeTask("t1", "fetch", "fetch")) {
		t.Fatal("expected denied by DenyWhats")
	}
}

// --- FilteringTracer ---

func TestFilteringTracer_AllowedTask_Forwarded(t *testing.T) {
	rec := &recordingTracer{}
	f := &TraceFilter{AllowKinds: []string{"wavefront"}}
	ft := NewFilteringTracer(rec, f)

	task := makeTask("id1", "wavefront", "")
	ft.StartTask(task)
	ft.StepTask(task)
	ft.AddMilestone(makeMilestone("id1"))
	ft.EndTask(task)

	if len(rec.startCalls) != 1 {
		t.Fatalf("expected 1 StartTask, got %d", len(rec.startCalls))
	}
	if len(rec.stepCalls) != 1 {
		t.Fatalf("expected 1 StepTask, got %d", len(rec.stepCalls))
	}
	if len(rec.milestoneCalls) != 1 {
		t.Fatalf("expected 1 AddMilestone, got %d", len(rec.milestoneCalls))
	}
	if len(rec.endCalls) != 1 {
		t.Fatalf("expected 1 EndTask, got %d", len(rec.endCalls))
	}
}

func TestFilteringTracer_DeniedTask_Suppressed(t *testing.T) {
	rec := &recordingTracer{}
	f := &TraceFilter{DenyKinds: []string{"req_in"}}
	ft := NewFilteringTracer(rec, f)

	task := makeTask("id2", "req_in", "")
	ft.StartTask(task)
	ft.StepTask(task)
	ft.AddMilestone(makeMilestone("id2"))
	ft.EndTask(task)

	if len(rec.startCalls) != 0 {
		t.Fatalf("expected 0 StartTask calls for denied task, got %d", len(rec.startCalls))
	}
	if len(rec.stepCalls) != 0 {
		t.Fatalf("expected 0 StepTask calls for denied task, got %d", len(rec.stepCalls))
	}
	if len(rec.milestoneCalls) != 0 {
		t.Fatalf("expected 0 Milestone calls for denied task, got %d", len(rec.milestoneCalls))
	}
	if len(rec.endCalls) != 0 {
		t.Fatalf("expected 0 EndTask calls for denied task, got %d", len(rec.endCalls))
	}
}

func TestFilteringTracer_MixedTasks(t *testing.T) {
	rec := &recordingTracer{}
	f := &TraceFilter{AllowKinds: []string{"wavefront"}}
	ft := NewFilteringTracer(rec, f)

	allowed := makeTask("a1", "wavefront", "")
	denied := makeTask("d1", "req_in", "")

	ft.StartTask(allowed)
	ft.StartTask(denied)
	ft.EndTask(allowed)
	ft.EndTask(denied)

	if len(rec.startCalls) != 1 || rec.startCalls[0].ID != "a1" {
		t.Fatal("only the allowed task should have been forwarded")
	}
	if len(rec.endCalls) != 1 || rec.endCalls[0].ID != "a1" {
		t.Fatal("only the allowed task end should have been forwarded")
	}
}

func TestFilteringTracer_NoFilters_ForwardsAll(t *testing.T) {
	rec := &recordingTracer{}
	ft := NewFilteringTracer(rec, &TraceFilter{})

	for _, kind := range []string{"wavefront", "req_in", "pipeline"} {
		task := makeTask(kind, kind, "")
		ft.StartTask(task)
		ft.EndTask(task)
	}

	if len(rec.startCalls) != 3 {
		t.Fatalf("expected 3 starts with no filter, got %d", len(rec.startCalls))
	}
}

// --- matchesAny ---

func TestMatchesAny_GlobStar(t *testing.T) {
	if !matchesAny("GPU0.CU1", []string{"*CU*"}) {
		t.Fatal("*CU* should match GPU0.CU1")
	}
}

func TestMatchesAny_Substring(t *testing.T) {
	if !matchesAny("GPU0.TLB0", []string{"TLB"}) {
		t.Fatal("plain TLB should match GPU0.TLB0 as substring")
	}
}

func TestMatchesAny_NoMatch(t *testing.T) {
	if matchesAny("GPU0.CU1", []string{"*TLB*", "RDMA"}) {
		t.Fatal("should not match")
	}
}

// --- splitTrimmed ---

func TestSplitTrimmed(t *testing.T) {
	got := splitTrimmed("  wavefront , pipeline , req_in  ")
	want := []string{"wavefront", "pipeline", "req_in"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}
