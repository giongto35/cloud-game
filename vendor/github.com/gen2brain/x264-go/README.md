## x264-go
[![TravisCI Build Status](https://travis-ci.org/gen2brain/x264-go.svg?branch=master)](https://travis-ci.org/gen2brain/x264-go) 
[![AppVeyor Build Status](https://ci.appveyor.com/api/projects/status/wfkqlac5ffwk5xgb?svg=true)](https://ci.appveyor.com/project/gen2brain/x264-go)
[![GoDoc](https://godoc.org/github.com/gen2brain/x264-go?status.svg)](https://godoc.org/github.com/gen2brain/x264-go) 
[![Go Report Card](https://goreportcard.com/badge/github.com/gen2brain/x264-go?branch=master)](https://goreportcard.com/report/github.com/gen2brain/x264-go) 

`x264-go` provides H.264/MPEG-4 AVC codec encoder based on [x264](https://www.videolan.org/developers/x264.html) library.

C source code is included in package. If you want to use external shared/static library (i.e. built with asm and/or OpenCL) use `-tags extlib`.

### Installation

    go get -u github.com/gen2brain/x264-go

### Examples

See [screengrab](https://github.com/gen2brain/x264-go/blob/master/examples/screengrab/screengrab.go) example.

### Usage

```go
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"

	"github.com/gen2brain/x264-go"
)

func main() {
	buf := bytes.NewBuffer(make([]byte, 0))

	opts := &x264.Options{
		Width:     640,
		Height:    480,
		FrameRate: 25,
		Tune:      "zerolatency",
		Preset:    "veryfast",
		Profile:   "baseline",
		LogLevel:  x264.LogDebug,
	}

	enc, err := x264.NewEncoder(buf, opts)
	if err != nil {
		panic(err)
	}

	img := x264.NewYCbCr(image.Rect(0, 0, opts.Width, opts.Height))
	draw.Draw(img, img.Bounds(), image.Black, image.ZP, draw.Src)

	for i := 0; i < opts.Width/2; i++ {
		img.Set(i, opts.Height/2, color.RGBA{255, 0, 0, 255})

		err = enc.Encode(img)
		if err != nil {
			panic(err)
		}
	}

	err = enc.Flush()
	if err != nil {
		panic(err)
	}

	err = enc.Close()
	if err != nil {
		panic(err)
	}
}
```

## More

For AAC encoder see [aac-go](https://github.com/gen2brain/aac-go).
