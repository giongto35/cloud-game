package opus

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestConverters(t *testing.T) {

	var data []int16
	for i := -32768; i < 32768; i ++ {
		data = append(data, int16(i))
	}

	a := toBytes(data)
	b := toInt16(a)

	if !reflect.DeepEqual(data, b) {
		t.Fatalf("convertion has failed, %v -> %v != %v", data, a, b)
	}

}
