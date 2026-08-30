package runtimeinfo

import "testing"

func TestOffsetsFitInsideValue(t *testing.T) {
	layout := ValueLayout()
	if layout.CountOffset%layout.Alignment != 0 {
		t.Fatalf("count offset %d is not aligned to %d", layout.CountOffset, layout.Alignment)
	}
	if layout.CodeOffset >= layout.Size || layout.EnabledOffset >= layout.Size {
		t.Fatalf("offsets do not fit: %+v", layout)
	}
}
