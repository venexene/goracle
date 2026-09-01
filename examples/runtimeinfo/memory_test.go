package runtimeinfo

import (
	"runtime"
	"testing"
)

var retainedBlocks [][]byte

func TestMemoryMetricsAreAvailable(t *testing.T) {
	snapshot := ReadMemory()
	if snapshot.RuntimeMemory == 0 {
		t.Fatalf("runtime memory must be positive: %+v", snapshot)
	}
	if snapshot.TotalAllocs < snapshot.HeapObjects {
		t.Fatalf("cumulative allocations are smaller than heap objects: %+v", snapshot)
	}
}

func TestAllocateBlocks(t *testing.T) {
	blocks := AllocateBlocks(3, 16)
	if len(blocks) != 3 || len(blocks[0]) != 16 || blocks[2][0] != 2 {
		t.Fatalf("unexpected blocks: %#v", blocks)
	}
}

func BenchmarkRetainedBlocks(b *testing.B) {
	// Достаточно большой набор остаётся достижимым до записи профиля процесса.
	retainedBlocks = AllocateBlocks(2048, 1024)
	for b.Loop() {
		runtime.KeepAlive(retainedBlocks)
	}
}

func BenchmarkTemporaryBlocks(b *testing.B) {
	for b.Loop() {
		blocks := AllocateBlocks(32, 1024)
		runtime.KeepAlive(blocks)
	}
}
