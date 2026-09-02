package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/venexene/goracle/examples/runtimeinfo"
)

func main() {
	rounds := flag.Int("rounds", 200, "число раундов")
	blocks := flag.Int("blocks", 256, "число блоков в раунде")
	size := flag.Int("size", 4096, "размер блока в байтах")
	reuse := flag.Bool("reuse", false, "повторно использовать блоки между раундами")
	flag.Parse()

	runtime.GC()
	before := runtimeinfo.ReadGC()
	started := time.Now()
	var checksum uint64
	if *reuse {
		checksum = runtimeinfo.RunGCWorkloadReusing(*rounds, *blocks, *size)
	} else {
		checksum = runtimeinfo.RunGCWorkload(*rounds, *blocks, *size)
	}
	elapsed := time.Since(started)
	runtime.GC()
	after := runtimeinfo.ReadGC()
	delta := runtimeinfo.DifferenceGC(before, after)

	fmt.Printf("go=%s goos=%s goarch=%s gomaxprocs=%d\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
	fmt.Printf("GOGC=%q GOMEMLIMIT=%q rounds=%d blocks=%d size=%d reuse=%t\n",
		os.Getenv("GOGC"), os.Getenv("GOMEMLIMIT"), *rounds, *blocks, *size, *reuse)
	fmt.Printf("elapsed=%s checksum=%d\n", elapsed, checksum)
	fmt.Printf("alloc_bytes=%d automatic_cycles=%d forced_cycles=%d gc_cpu=%.6fs assist_cpu=%.6fs pauses=%d\n",
		delta.HeapAllocs, delta.AutomaticCycles, delta.ForcedCycles,
		delta.GCCPU, delta.AssistCPU, delta.Pauses)
	fmt.Printf("live_heap=%d heap_goal=%d accounted_memory=%d limiter_cycle=%d pause_p50=%gs pause_p95=%gs pause_p99=%gs\n",
		after.LiveHeap, after.HeapGoal, after.AccountedMemory(), after.LastLimiterCycle,
		after.Pauses.P50, after.Pauses.P95, after.Pauses.P99)
}
