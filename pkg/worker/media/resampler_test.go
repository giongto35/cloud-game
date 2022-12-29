package media

import (
	"math/rand"
	"testing"
	"time"
)

func TestResampleStretch(t *testing.T) {
	type args struct {
		pcm  []int16
		size int
	}
	tests := []struct {
		name string
		args args
		want []int16
	}{
		//1764:1920
		{
			name: "",
			args: args{
				pcm:  gen(1764),
				size: 1920,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rez := ResampleStretch(tt.args.pcm, tt.args.size)
			rez2 := ResampleStretch(tt.args.pcm, tt.args.size)

			for i := range rez {
				if rez[i] != rez2[i] {
					t.Errorf("no %v", i)
				}
			}
		})
	}
}

func Benchmark(b *testing.B) {
	pcm := gen(1764)
	size := 1920
	count := 32
	for i := 0; i < b.N; i++ {
		for j := 0; j < count; j++ {
			_ = ResampleStretch(pcm, size)
		}
	}
}

func gen(l int) []int16 {
	rand.Seed(time.Now().Unix())

	nums := make([]int16, l)
	for i := range nums {
		nums[i] = int16(rand.Intn(10))
	}
	for i := len(nums) / 2; i < len(nums)/2+42; i++ {
		nums[i] = 0
	}

	return nums
}
