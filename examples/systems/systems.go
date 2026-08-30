package systems

import (
	"encoding/binary"
	"sync/atomic"
)

func DecodeUint32(data []byte, order binary.ByteOrder) uint32 {
	return order.Uint32(data)
}

type Counter struct {
	value atomic.Int64
}

func (c *Counter) Add(delta int64) { c.value.Add(delta) }
func (c *Counter) Load() int64     { return c.value.Load() }
