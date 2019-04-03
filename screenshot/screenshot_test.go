package screenshot

import (
	"fmt"
	"testing"
)

func TestScreenshotToYuv(t *testing.T) {
	w, h := GetScreenSize()
	rgbaImg := GetScreenshot(0, 0, w, h, w, h)
	yuv := RgbaToYuv(rgbaImg)
	fmt.Println("yuv len", len(yuv))
}
