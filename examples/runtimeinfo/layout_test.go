package runtimeinfo

import "testing"

func TestOffsetsFitInsideValue(t *testing.T) {
	layout := ValueLayout()
	if layout.CountOffset%layout.Alignment != 0 {
		t.Fatalf("count offset %d is not aligned to %d", layout.CountOffset, layout.Alignment)
	}
	if layout.CodeOffset >= layout.Size || layout.EnabledOffset >= layout.Size {
		t.Fatalf("offsets do not fit: %+v", layout)
	}
}

func TestSchedulerSnapshot(t *testing.T) {
	snapshot := ReadScheduler()
	if snapshot.LogicalCPUs < 1 || snapshot.GOMAXPROCS < 1 || snapshot.Goroutines < 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestHeapGoalMetric(t *testing.T) {
	if goal := HeapGoal(); goal == 0 {
		t.Fatal("heap goal must be positive")
	}
}

func BenchmarkValueCall(b *testing.B) {
	value := Value{Count: 40, Code: 2}
	for b.Loop() {
		_ = Sum(value)
	}
}

func BenchmarkInterfaceCall(b *testing.B) {
	var value Summer = Value{Count: 40, Code: 2}
	for b.Loop() {
		_ = value.Sum()
	}
}
