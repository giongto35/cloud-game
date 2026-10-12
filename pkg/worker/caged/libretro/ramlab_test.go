//go:build ramlab

// RAM lab: drives a game headless with a scripted input sequence to discover and
// verify RAM addresses (match end, winner). Not part of the normal test run.
//
//	RAMLAB_SCRIPT=lab/umk3.txt RAMLAB_OUT=/tmp/out go test -tags ramlab -run TestRamLab ./pkg/worker/caged/libretro
//
// Script lines (# comments):
//
//	game <system> <rom path relative to assets/games>
//	run <frames> [p1 buttons] [p2 buttons]   buttons: b,y,select,start,up,down,left,right,a,x,l,r joined by +, or -
//	tap <p1 buttons> [p2 buttons]            press 10 frames, release 10
//	shot <name>                              save the screen as PNG
//	snap <name>                              dump the work RAM
//	save <name> / load <name>                emulator save states (to resume experiments)
//	watch <addr> [addr...]                   print these WRAM bytes on every change (hex, 7E prefix optional)
//	until <addr> <op> <value> <max frames> [p1] [p2]  run until the byte matches (==,!=,<,>)
//	log <every frames>                       periodic print of watched bytes
package libretro

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
)

var labButtons = map[string]uint16{
	"b": 1 << 0, "y": 1 << 1, "select": 1 << 2, "start": 1 << 3, "up": 1 << 4, "down": 1 << 5,
	"left": 1 << 6, "right": 1 << 7, "a": 1 << 8, "x": 1 << 9, "l": 1 << 10, "r": 1 << 11,
}

// labSpec is a button spec; a "~" prefix mashes (press 6 frames, release 6).
// "a|b" alternates between two specs every 40 frames (e.g. walk|~attack).
type labSpec struct {
	mask uint16
	mash bool
	alt  *labSpec
}

func spec(s string) labSpec {
	if i := strings.Index(s, "|"); i >= 0 {
		first, second := spec(s[:i]), spec(s[i+1:])
		first.alt = &second
		return first
	}
	return labSpec{mask: labMask(strings.TrimPrefix(s, "~")), mash: strings.HasPrefix(s, "~")}
}

func (s labSpec) at(frame int) uint16 {
	if s.alt != nil && (frame/40)%2 == 1 {
		return s.alt.at(frame)
	}
	if s.mash && (frame/6)%2 == 1 {
		return 0
	}
	return s.mask
}

func labMask(s string) uint16 {
	if s == "" || s == "-" {
		return 0
	}
	var m uint16
	for _, b := range strings.Split(s, "+") {
		m |= labButtons[strings.ToLower(b)]
	}
	return m
}

func labAddr(s string) int {
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(s) == 6 && strings.HasPrefix(s, "7e") {
		s = s[2:]
	} else if len(s) == 6 && strings.HasPrefix(s, "7f") {
		v, _ := strconv.ParseInt(s[2:], 16, 32)
		return int(v) + 0x10000
	}
	v, _ := strconv.ParseInt(s, 16, 32)
	return int(v)
}

type lab struct {
	t       *testing.T
	emu     *TestFrontend
	out     string
	frame   int
	watch   []int
	last    map[int]byte
	logEach int
	video   app.RawFrame
	trace   *os.File
	every   int
	events  *os.File
}

func (l *lab) input(p1, p2 uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], p1)
	l.emu.Input(0, byte(RetroPad), b[:])
	binary.LittleEndian.PutUint16(b[:], p2)
	l.emu.Input(1, byte(RetroPad), b[:])
}

func (l *lab) step(p1, p2 uint16) {
	l.input(p1, p2)
	l.emu.Tick()
	l.frame++
	ram := nanoarch.SystemRAM()
	for _, a := range l.watch {
		if v := ram[a]; v != l.last[a] {
			fmt.Printf("f%06d 7E%04X %3d -> %3d\n", l.frame, a, l.last[a], v)
			l.last[a] = v
		}
	}
	if l.trace != nil && l.frame%l.every == 0 {
		var hdr [4]byte
		binary.LittleEndian.PutUint32(hdr[:], uint32(l.frame))
		_, _ = l.trace.Write(hdr[:])
		_, _ = l.trace.Write(ram)
	}
	if l.logEach > 0 && l.frame%l.logEach == 0 && len(l.watch) > 0 {
		var sb strings.Builder
		for _, a := range l.watch {
			fmt.Fprintf(&sb, " %04X=%d", a, ram[a])
		}
		fmt.Printf("f%06d%s\n", l.frame, sb.String())
	}
}

func (l *lab) shot(name string) {
	f := l.video
	if f.W == 0 {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, f.W, f.H))
	pf := l.emu.PixFormat()
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			switch pf {
			case 1: // XRGB8888
				o := y*f.Stride + x*4
				img.Set(x, y, color.RGBA{f.Data[o+2], f.Data[o+1], f.Data[o], 255})
			default: // RGB565
				o := y*f.Stride + x*2
				v := binary.LittleEndian.Uint16(f.Data[o:])
				img.Set(x, y, color.RGBA{uint8(v>>11) << 3, uint8(v>>5&0x3f) << 2, uint8(v&0x1f) << 3, 255})
			}
		}
	}
	fp, err := os.Create(filepath.Join(l.out, fmt.Sprintf("%06d-%s.png", l.frame, name)))
	if err != nil {
		l.t.Fatal(err)
	}
	_ = png.Encode(fp, img)
	_ = fp.Close()
}

func cmp(v byte, op string, x int) bool {
	switch op {
	case "==":
		return int(v) == x
	case "!=":
		return int(v) != x
	case "<":
		return int(v) < x
	case ">":
		return int(v) > x
	}
	return false
}

func TestRamLab(t *testing.T) {
	script := os.Getenv("RAMLAB_SCRIPT")
	if script == "" {
		t.Skip("RAMLAB_SCRIPT is not set")
	}
	out := os.Getenv("RAMLAB_OUT")
	if out == "" {
		out = filepath.Join(os.TempDir(), "ramlab")
	}
	_ = os.MkdirAll(out, 0o755)

	f, err := os.Open(script)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	l := &lab{t: t, out: out, last: map[int]byte{}}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		a := strings.Fields(line)
		arg := func(i int) string {
			if i < len(a) {
				return a[i]
			}
			return ""
		}
		switch a[0] {
		case "game":
			if os.Getenv("RAMLAB_NOHID") != "" { // plain joypads instead of the configured devices (multitap)
				l.emu = EmulatorMock("ramlab", a[1])
				core := l.emu.conf.Libretro.Cores.List[a[1]]
				core.Hid = nil
				l.emu.conf.Libretro.Cores.List[a[1]] = core
				l.emu.loadRom(strings.Join(a[2:], " "))
				l.emu.SetAudioCb(func(app.Audio) {})
			} else {
				l.emu = DefaultFrontend("ramlab", a[1], strings.Join(a[2:], " "))
			}
			l.emu.SetVideoCb(func(v app.Video) {
				l.video.W, l.video.H, l.video.Stride = v.Frame.W, v.Frame.H, v.Frame.Stride
				l.video.Data = append(l.video.Data[:0], v.Frame.Data...)
			})
			defer l.emu.Shutdown()
			fmt.Printf("loaded %s, WRAM %d bytes\n", a[2], len(nanoarch.SystemRAM()))
		case "run":
			n, _ := strconv.Atoi(a[1])
			p1, p2 := spec(arg(2)), spec(arg(3))
			for range n {
				l.step(p1.at(l.frame), p2.at(l.frame))
			}
		case "tap":
			p1, p2 := labMask(arg(1)), labMask(arg(2))
			for range 10 { // some games only sample the pad every few frames
				l.step(p1, p2)
			}
			for range 10 {
				l.step(0, 0)
			}
		case "shot":
			l.shot(a[1])
		case "snap":
			ram := nanoarch.SystemRAM()
			if err := os.WriteFile(filepath.Join(out, a[1]+".ram"), ram, 0o644); err != nil {
				t.Fatal(err)
			}
			fmt.Printf("f%06d snap %s\n", l.frame, a[1])
		case "save":
			st, err := nanoarch.SaveState()
			if err != nil {
				t.Fatal(err)
			}
			_ = os.WriteFile(filepath.Join(out, a[1]+".state"), st, 0o644)
		case "load":
			st, err := os.ReadFile(filepath.Join(out, a[1]+".state"))
			if err != nil { // shared states made by the setup scripts
				st, err = os.ReadFile(filepath.Join("lab", "states", a[1]+".state"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := nanoarch.RestoreSaveState(st); err != nil {
				t.Fatal(err)
			}
		case "watch":
			ram := nanoarch.SystemRAM()
			for _, s := range a[1:] {
				ad := labAddr(s)
				l.watch = append(l.watch, ad)
				l.last[ad] = ram[ad]
				fmt.Printf("f%06d watch 7E%04X = %d\n", l.frame, ad, ram[ad])
			}
		case "trace": // trace <name> <every frames>: full RAM time series
			l.every, _ = strconv.Atoi(a[2])
			l.trace, err = os.Create(filepath.Join(out, a[1]+".trace"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = l.trace.Close() }()
		case "event": // event <name>: marks a frame for the analysis
			if l.events == nil {
				l.events, _ = os.Create(filepath.Join(out, "events.txt"))
				defer func() { _ = l.events.Close() }()
			}
			fmt.Fprintf(l.events, "%d %s\n", l.frame, a[1])
			fmt.Printf("f%06d event %s\n", l.frame, a[1])
		case "poke": // poke <addr> <value>: writes a WRAM byte (to force a game situation)
			v, _ := strconv.Atoi(a[2])
			nanoarch.SystemRAM()[labAddr(a[1])] = byte(v)
		case "unwatch":
			l.watch = nil
		case "log":
			l.logEach, _ = strconv.Atoi(a[1])
		case "until":
			ad := labAddr(a[1])
			x, _ := strconv.Atoi(a[3])
			max, _ := strconv.Atoi(a[4])
			p1, p2 := spec(arg(5)), spec(arg(6))
			ok := false
			for range max {
				l.step(p1.at(l.frame), p2.at(l.frame))
				if cmp(nanoarch.SystemRAM()[ad], a[2], x) {
					ok = true
					break
				}
			}
			fmt.Printf("f%06d until 7E%04X %s %d: %v\n", l.frame, ad, a[2], x, ok)
		default:
			t.Fatalf("unknown command %q", a[0])
		}
	}
}
