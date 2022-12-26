package zip

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestCompression(t *testing.T) {
	type args struct {
		data []byte
		name string
	}
	tests := []struct {
		name     string
		args     args
		want     []byte
		wantName string
		wantErr  bool
	}{
		{
			name: "a simple compression/decompression check",
			args: args{
				data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
				name: "test",
			},
			want:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
			wantName: "test",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compress(tt.args.data, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, name, err := Read(got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if name != tt.wantName {
				t.Errorf("Compress() got name = %v, want %v", name, tt.wantName)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Compress() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkCompressions(b *testing.B) {
	benchmarks := []struct {
		name string
		size int
	}{
		{name: "compress", size: 1024 * 1024 * 1},
		{name: "compress", size: 1024 * 1024 * 2},
	}
	for _, bm := range benchmarks {
		rand.Seed(time.Now().UnixNano())
		b.Run(fmt.Sprintf("%v %v", bm.name, bm.size), func(b *testing.B) {
			dat := make([]byte, bm.size)
			rand.Read(dat)
			for i := 0; i < b.N; i++ {
				_, _ = Compress(dat, "test")
			}
		})
	}
}
