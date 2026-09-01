package runtimeinfo

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestOffsetsFitInsideValue(t *testing.T) {
	layout := ValueLayout()
	if layout.CountOffset%layout.Alignment != 0 {
		t.Fatalf("count offset %d is not aligned to %d", layout.CountOffset, layout.Alignment)
	}
	if layout.CodeOffset >= layout.Size || layout.EnabledOffset >= layout.Size {
		t.Fatalf("offsets do not fit: %+v", layout)
	}
}

func TestCompactValueDoesNotGrow(t *testing.T) {
	before := ValueLayout()
	after := CompactValueLayout()
	if after.Size > before.Size {
		t.Fatalf("compact layout grew: before=%+v after=%+v", before, after)
	}
}

func TestAMD64ScalarAlignments(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("проверка относится к amd64")
	}
	if got := unsafe.Alignof(complex64(0)); got != 4 {
		t.Fatalf("complex64 alignment = %d, want 4", got)
	}
	if got := unsafe.Alignof(complex128(0)); got != 8 {
		t.Fatalf("complex128 alignment = %d, want 8", got)
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
