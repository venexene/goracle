package runtimeinfo

import (
	"runtime"
	"runtime/metrics"
	"unsafe"
)

type Value struct {
	Enabled bool
	Count   int64
	Code    uint16
}

// CompactValue хранит те же данные, но группирует поля по выравниванию.
type CompactValue struct {
	Count   int64
	Code    uint16
	Enabled bool
}

type SchedulerSnapshot struct {
	LogicalCPUs int
	GOMAXPROCS  int
	Goroutines  int
}

func ReadScheduler() SchedulerSnapshot {
	return SchedulerSnapshot{
		LogicalCPUs: runtime.NumCPU(),
		GOMAXPROCS:  runtime.GOMAXPROCS(0),
		Goroutines:  runtime.NumGoroutine(),
	}
}

func HeapGoal() uint64 {
	samples := []metrics.Sample{{Name: "/gc/heap/goal:bytes"}}
	metrics.Read(samples)
	return samples[0].Value.Uint64()
}

func Sum(value Value) int64 { return int64(value.Code) + value.Count }

type Summer interface{ Sum() int64 }

func (value Value) Sum() int64 { return Sum(value) }

type Layout struct {
	Size, Alignment uintptr
	EnabledOffset   uintptr
	CountOffset     uintptr
	CodeOffset      uintptr
}

func ValueLayout() Layout {
	var value Value
	return Layout{
		Size:          unsafe.Sizeof(value),
		Alignment:     unsafe.Alignof(value),
		EnabledOffset: unsafe.Offsetof(value.Enabled),
		CountOffset:   unsafe.Offsetof(value.Count),
		CodeOffset:    unsafe.Offsetof(value.Code),
	}
}

func CompactValueLayout() Layout {
	var value CompactValue
	return Layout{
		Size:          unsafe.Sizeof(value),
		Alignment:     unsafe.Alignof(value),
		EnabledOffset: unsafe.Offsetof(value.Enabled),
		CountOffset:   unsafe.Offsetof(value.Count),
		CodeOffset:    unsafe.Offsetof(value.Code),
	}
}
