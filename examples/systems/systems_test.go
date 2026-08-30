package systems

import (
	"encoding/binary"
	"sync"
	"testing"
)

func TestByteOrder(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	if got := DecodeUint32(data, binary.BigEndian); got != 0x01020304 {
		t.Fatalf("big endian = %#x", got)
	}
	if got := DecodeUint32(data, binary.LittleEndian); got != 0x04030201 {
		t.Fatalf("little endian = %#x", got)
	}
}

func TestAtomicCounter(t *testing.T) {
	var counter Counter
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				counter.Add(1)
			}
		})
	}
	workers.Wait()
	if got := counter.Load(); got != 800 {
		t.Fatalf("counter = %d", got)
	}
}
