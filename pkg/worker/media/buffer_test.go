package media

import "testing"

func Test_buffer2_write2(t *testing.T) {

	buf := newOpusBuffer(1000)

	t.Logf("%+v", buf)
}
