package runtimeinfo

import (
	"runtime"
	"runtime/metrics"
	"testing"
)

func TestGCMetricsAreAvailable(t *testing.T) {
	runtime.GC()
	snapshot := ReadGC()
	if snapshot.HeapGoal == 0 || snapshot.RuntimeMemory == 0 {
		t.Fatalf("GC metrics are unavailable: %+v", snapshot)
	}
	if snapshot.AccountedMemory() > snapshot.RuntimeMemory {
		t.Fatalf("accounted memory exceeds runtime memory: %+v", snapshot)
	}
}

func TestGCDifference(t *testing.T) {
	before := ReadGC()
	RunGCWorkload(4, 32, 1024)
	runtime.GC()
	after := ReadGC()
	delta := DifferenceGC(before, after)
	if delta.HeapAllocs == 0 {
		t.Fatalf("workload produced no measured allocations: before=%+v after=%+v", before, after)
	}
	if delta.ForcedCycles == 0 {
		t.Fatalf("runtime.GC did not increase forced cycles: before=%+v after=%+v", before, after)
	}
}

func TestHistogramQuantileUsesBucketBoundary(t *testing.T) {
	histogram := &metrics.Float64Histogram{
		Buckets: []float64{0, 0.001, 0.01, 0.1},
		Counts:  []uint64{1, 7, 2},
	}
	if got := HistogramQuantile(histogram, 0.50); got != 0.01 {
		t.Fatalf("p50 = %v, want upper bucket boundary 0.01", got)
	}
}

func TestGCWorkloadVariantsDoSameUsefulWrites(t *testing.T) {
	const (
		rounds = 4
		blocks = 32
		size   = 1024
	)
	want := RunGCWorkload(rounds, blocks, size)
	if got := RunGCWorkloadReusing(rounds, blocks, size); got != want {
		t.Fatalf("reusing workload checksum = %d, want %d", got, want)
	}
}

func BenchmarkGCWorkload(b *testing.B) {
	for b.Loop() {
		gcWorkloadSink = RunGCWorkload(8, 64, 1024)
	}
}

func BenchmarkGCWorkloadReusing(b *testing.B) {
	for b.Loop() {
		gcWorkloadSink = RunGCWorkloadReusing(8, 64, 1024)
	}
}
