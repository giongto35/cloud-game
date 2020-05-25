// This package contains functions for
// Pi/2 step rotation of points in a 2-dimensional space.
package image

type Angle uint

const (
	Angle0 Angle = iota
	Angle90
	Angle180
	Angle270
)

// A helper to choose appropriate rotation by its angle
var Angles = [4]Rotate{
	Angle0:   {Call: Rotate0, IsEven: false},
	Angle90:  {Call: Rotate90, IsEven: true},
	Angle180: {Call: Rotate180, IsEven: false},
	Angle270: {Call: Rotate270, IsEven: true},
}

func GetRotation(angle Angle) Rotate {
	return Angles[angle]
}

// An interface for rotation of a given point
// with the coordinates x, y in the matrix of w x h.
// Returns a pair of new coordinates x, y in the resulting
// matrix.
// Be aware that w / h values are 0 index-based and
// it's meant to be used with h corresponded
// to matrix height and y coordinate, and with w to x coordinate.
type Rotate struct {
	Call   func(x, y, w, h int) (int, int)
	IsEven bool
}

// 0° or the original orientation
/* Example: */
/* 1 2 3    1 2 3 */
/* 4 5 6 -> 4 5 6 */
/* 7 8 9    7 8 9 */
func Rotate0(x, y, _, _ int) (int, int) {
	return x, y
}

// 90° CCW or 270° CW
/* Example: */
/* 1 2 3    3 6 9 */
/* 4 5 6 -> 2 5 8 */
/* 7 8 9    1 4 7 */
func Rotate90(x, y, w, _ int) (int, int) {
	return y, (w - 1) - x
}

// 180° CCW
/* Example: */
/* 1 2 3    9 8 7 */
/* 4 5 6 -> 6 5 4 */
/* 7 8 9    3 2 1 */
func Rotate180(x, y, w, h int) (int, int) {
	return (w - 1) - x, (h - 1) - y
}

// 270° CCW or 90° CW
/* Example: */
/* 1 2 3    7 4 1 */
/* 4 5 6 -> 8 5 2 */
/* 7 8 9    9 6 3 */
func Rotate270(x, y, _, h int) (int, int) {
	return (h - 1) - y, x
}

/*
[1 2 3 4 5 6 7 8 9]
[7 4 1 8 5 2 9 6 3]
*/
func ExampleRotate(data []uint8, w int, h int, angle Angle) []uint8 {
	dest := make([]uint8, len(data))
	rotationFn := Angles[angle]

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx, ny := rotationFn.Call(x, y, w, h)
			stride := w
			if rotationFn.IsEven {
				stride = h
			}
			//fmt.Printf("%v:%v (%v) -> %v:%v (%v)\n", x, y, n1, nx, ny, n2)

			dest[nx+ny*stride] = data[x+y*w]
		}
	}

	return dest
}
