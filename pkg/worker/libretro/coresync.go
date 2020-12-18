package libretro

import (
	"log"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
)

func SyncCores(conf worker.Config) {
	list := conf.Emulator.GetCores()

	log.Printf("[worker] start cores sync: %v", strings.Join(list, ", "))

	// get each core
	// convert to the current arch of the worker
	// check prefix of the filename
	// download nonexistent core names
	// profit
}
