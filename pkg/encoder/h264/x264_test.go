package h264

import "testing"

func TestH264Encode(t *testing.T) {
	h264, err := NewEncoder(120, 120, 0, nil)
	if err != nil {
		t.Error(err)
		return
	}
	data := make([]byte, 120*120*1.5)
	h264.LoadBuf(data)
	h264.Encode()
	if err := h264.Shutdown(); err != nil {
		t.Error(err)
	}
}

func Benchmark(b *testing.B) {
	w, h := 1920, 1080
	h264, err := NewEncoder(w, h, 0, nil)
	if err != nil {
		b.Error(err)
		return
	}
	data := make([]byte, int(float64(w)*float64(h)*1.5))
	for i := 0; i < b.N; i++ {
		h264.LoadBuf(data)
		h264.Encode()
	}
}
