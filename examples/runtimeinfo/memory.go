package runtimeinfo

import "runtime/metrics"

// MemorySnapshot содержит несколько величин с разной семантикой. Их нельзя
// складывать: снимок нужен для сопоставления динамики показателей.
type MemorySnapshot struct {
	LiveHeap      uint64
	HeapObjects   uint64
	TotalAllocs   uint64
	HeapReleased  uint64
	RuntimeMemory uint64
}

func ReadMemory() MemorySnapshot {
	names := []string{
		"/gc/heap/live:bytes",
		"/memory/classes/heap/objects:bytes",
		"/gc/heap/allocs:bytes",
		"/memory/classes/heap/released:bytes",
		"/memory/classes/total:bytes",
	}
	samples := make([]metrics.Sample, len(names))
	for index, name := range names {
		samples[index].Name = name
	}
	metrics.Read(samples)
	return MemorySnapshot{
		LiveHeap:      samples[0].Value.Uint64(),
		HeapObjects:   samples[1].Value.Uint64(),
		TotalAllocs:   samples[2].Value.Uint64(),
		HeapReleased:  samples[3].Value.Uint64(),
		RuntimeMemory: samples[4].Value.Uint64(),
	}
}

func AllocateBlocks(count, size int) [][]byte {
	blocks := make([][]byte, count)
	for index := range blocks {
		blocks[index] = make([]byte, size)
		if size > 0 {
			blocks[index][0] = byte(index)
		}
	}
	return blocks
}
