package thread

import "testing"

var f = func() {}

func BenchmarkDirectCall(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f()
	}
}

func BenchmarkMainThreadCall(b *testing.B) {
	Run(func() {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Call(f)
		}
	})
}
