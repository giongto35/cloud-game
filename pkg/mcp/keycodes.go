package mcp

var keys = map[string]uint32{
	"ArrowUp":    273,
	"ArrowDown":  274,
	"ArrowRight": 275,
	"ArrowLeft":  276,
	"Enter":      13,
	"Space":      32,
	"KeyA":       97,
	"KeyB":       98,
	"KeyX":       99,
	"KeyY":       100,
}

func keyCode(k string) uint32 {
	if v, ok := keys[k]; ok {
		return v
	}
	return 0
}
