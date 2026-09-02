package runtimeinfo

import (
	"math"
	"runtime"
	"runtime/metrics"
)

// GCSnapshot объединяет величины с разной семантикой. Мгновенные значения
// сравнивают напрямую, а монотонные счётчики — по разности двух снимков.
type GCSnapshot struct {
	LiveHeap         uint64
	HeapGoal         uint64
	HeapAllocs       uint64
	AutomaticCycles  uint64
	ForcedCycles     uint64
	ScanHeap         uint64
	ScanStack        uint64
	ScanGlobals      uint64
	RuntimeMemory    uint64
	HeapReleased     uint64
	LastLimiterCycle uint64
	GCCPU            float64
	AssistCPU        float64
	Pauses           PauseSummary
}

// PauseSummary содержит приближённые границы корзин распределения пауз.
type PauseSummary struct {
	Count uint64
	P50   float64
	P95   float64
	P99   float64
}

var gcMetricNames = []string{
	"/gc/heap/live:bytes",
	"/gc/heap/goal:bytes",
	"/gc/heap/allocs:bytes",
	"/gc/cycles/automatic:gc-cycles",
	"/gc/cycles/forced:gc-cycles",
	"/gc/scan/heap:bytes",
	"/gc/scan/stack:bytes",
	"/gc/scan/globals:bytes",
	"/memory/classes/total:bytes",
	"/memory/classes/heap/released:bytes",
	"/gc/limiter/last-enabled:gc-cycle",
	"/cpu/classes/gc/total:cpu-seconds",
	"/cpu/classes/gc/mark/assist:cpu-seconds",
	"/sched/pauses/total/gc:seconds",
}

func ReadGC() GCSnapshot {
	samples := make([]metrics.Sample, len(gcMetricNames))
	for index, name := range gcMetricNames {
		samples[index].Name = name
	}
	metrics.Read(samples)

	return GCSnapshot{
		LiveHeap:         samples[0].Value.Uint64(),
		HeapGoal:         samples[1].Value.Uint64(),
		HeapAllocs:       samples[2].Value.Uint64(),
		AutomaticCycles:  samples[3].Value.Uint64(),
		ForcedCycles:     samples[4].Value.Uint64(),
		ScanHeap:         samples[5].Value.Uint64(),
		ScanStack:        samples[6].Value.Uint64(),
		ScanGlobals:      samples[7].Value.Uint64(),
		RuntimeMemory:    samples[8].Value.Uint64(),
		HeapReleased:     samples[9].Value.Uint64(),
		LastLimiterCycle: samples[10].Value.Uint64(),
		GCCPU:            samples[11].Value.Float64(),
		AssistCPU:        samples[12].Value.Float64(),
		Pauses:           SummarizeHistogram(samples[13].Value.Float64Histogram()),
	}
}

// AccountedMemory возвращает величину, которую среда выполнения сопоставляет с
// мягким пределом памяти.
func (snapshot GCSnapshot) AccountedMemory() uint64 {
	if snapshot.HeapReleased >= snapshot.RuntimeMemory {
		return 0
	}
	return snapshot.RuntimeMemory - snapshot.HeapReleased
}

// GCDelta возвращает изменения монотонных показателей между снимками.
type GCDelta struct {
	HeapAllocs      uint64
	AutomaticCycles uint64
	ForcedCycles    uint64
	GCCPU           float64
	AssistCPU       float64
	Pauses          uint64
}

func DifferenceGC(before, after GCSnapshot) GCDelta {
	return GCDelta{
		HeapAllocs:      subtractUint64(after.HeapAllocs, before.HeapAllocs),
		AutomaticCycles: subtractUint64(after.AutomaticCycles, before.AutomaticCycles),
		ForcedCycles:    subtractUint64(after.ForcedCycles, before.ForcedCycles),
		GCCPU:           math.Max(0, after.GCCPU-before.GCCPU),
		AssistCPU:       math.Max(0, after.AssistCPU-before.AssistCPU),
		Pauses:          subtractUint64(after.Pauses.Count, before.Pauses.Count),
	}
}

func subtractUint64(after, before uint64) uint64 {
	if after < before {
		return 0
	}
	return after - before
}

func SummarizeHistogram(histogram *metrics.Float64Histogram) PauseSummary {
	var count uint64
	for _, bucketCount := range histogram.Counts {
		count += bucketCount
	}
	return PauseSummary{
		Count: count,
		P50:   HistogramQuantile(histogram, 0.50),
		P95:   HistogramQuantile(histogram, 0.95),
		P99:   HistogramQuantile(histogram, 0.99),
	}
}

// HistogramQuantile возвращает верхнюю границу корзины, содержащей квантиль.
// Это приближение, а не точное значение наблюдения.
func HistogramQuantile(histogram *metrics.Float64Histogram, quantile float64) float64 {
	if quantile < 0 {
		quantile = 0
	}
	if quantile > 1 {
		quantile = 1
	}

	var total uint64
	for _, count := range histogram.Counts {
		total += count
	}
	if total == 0 {
		return 0
	}

	target := uint64(math.Ceil(quantile * float64(total)))
	if target == 0 {
		target = 1
	}
	var seen uint64
	for index, count := range histogram.Counts {
		seen += count
		if seen < target {
			continue
		}
		upper := histogram.Buckets[index+1]
		if math.IsInf(upper, 1) {
			return histogram.Buckets[index]
		}
		return upper
	}
	return histogram.Buckets[len(histogram.Buckets)-1]
}

var gcWorkloadSink uint64

// RunGCWorkload создаёт одинаковый объём временной работы для сравнительных
// запусков. Возвращаемое значение мешает компилятору удалить вычисления.
func RunGCWorkload(rounds, blocks, size int) uint64 {
	mult := uint64(0)
	for round := 0; round < rounds; round++ {
		values := make([][]byte, blocks)
		for index := range values {
			values[index] = make([]byte, size)
			if size > 0 {
				values[index][0] = byte(round + index)
				mult += uint64(values[index][0])
			}
		}
		runtime.KeepAlive(values)
	}
	gcWorkloadSink = mult
	return mult
}

// RunGCWorkloadReusing выполняет те же записи, но повторно использует набор
// блоков между раундами. Функция нужна как проверяемое изменение для сравнения
// числа выделений и работы сборщика.
func RunGCWorkloadReusing(rounds, blocks, size int) uint64 {
	values := make([][]byte, blocks)
	for index := range values {
		values[index] = make([]byte, size)
	}

	mult := uint64(0)
	for round := 0; round < rounds; round++ {
		for index := range values {
			if size > 0 {
				values[index][0] = byte(round + index)
				mult += uint64(values[index][0])
			}
		}
	}
	runtime.KeepAlive(values)
	gcWorkloadSink = mult
	return mult
}
