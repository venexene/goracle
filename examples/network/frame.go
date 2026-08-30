package network

import (
	"encoding/binary"
	"errors"
	"io"
)

func WriteFrame(w io.Writer, payload []byte, limit uint32) error {
	if uint64(len(payload)) > uint64(limit) || uint64(len(payload)) > uint64(^uint32(0)) {
		return errors.New("сообщение слишком велико")
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, payload)
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
		data = data[n:]
	}
	return nil
}

func ReadFrame(r io.Reader, limit uint32) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > limit {
		return nil, errors.New("сообщение слишком велико")
	}
	payload := make([]byte, size)
	_, err := io.ReadFull(r, payload)
	return payload, err
}
