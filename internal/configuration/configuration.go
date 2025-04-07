package configuration

import (
	"fmt"

	"github.com/logrusorgru/aurora/v4"
)

const (
	NAME    = "xcrawl3r"
	VERSION = "0.2.0"
)

var (
	BANNER = func(au *aurora.Aurora) (banner string) {
		banner = au.Sprintf(
			au.BrightBlue(`
                             _ _____
__  _____ _ __ __ ___      _| |___ / _ __
\ \/ / __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
 >  < (__| | | (_| |\ V  V /| |___) | |
/_/\_\___|_|  \__,_| \_/\_/ |_|____/|_|
                                    %s`).Bold(),
			au.BrightRed("v"+VERSION).Bold().Italic(),
		) + "\n\n"

		return
	}

	DefaultUserAgent = fmt.Sprintf("%s v%s (https://github.com/hueristiq/%s)", NAME, VERSION, NAME)
)
