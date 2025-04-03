package stdio

import (
	"os"
)

func HasStdIn() (hasStdIn bool) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		hasStdIn = false

		return
	}

	isPipedFromChrDev := (stat.Mode() & os.ModeCharDevice) == 0
	isPipedFromFIFO := (stat.Mode() & os.ModeNamedPipe) != 0

	hasStdIn = isPipedFromChrDev || isPipedFromFIFO

	return
}
