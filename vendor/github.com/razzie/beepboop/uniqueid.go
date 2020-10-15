package beepboop

import (
	"fmt"
	"time"

	"github.com/tjarratt/babble"
)

// UniqueID ...
func UniqueID() string {
	i := uint16(time.Now().UnixNano())
	babbler := babble.NewBabbler()
	return fmt.Sprintf("%s-%x", babbler.Babble(), i)
}
