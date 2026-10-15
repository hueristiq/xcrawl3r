package input

import "os"

func HasStdin() (has bool) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return
	}

	isPipe := (stat.Mode() & os.ModeCharDevice) == 0
	isFIFO := (stat.Mode() & os.ModeNamedPipe) != 0

	has = isPipe || isFIFO

	return
}
