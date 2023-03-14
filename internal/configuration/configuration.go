package configuration

import (
	"github.com/logrusorgru/aurora/v3"
)

const (
	VERSION     = "v0.0.0"
	DESCRIPTION = "A CLI utility to recursively crawl webpages."
)

var (
	BANNER string = aurora.Sprintf(aurora.BrightBlue(`
 _                                   _ _____
| |__   __ _  ___ _ __ __ ___      _| |___ / _ __
| '_ \ / _`+"`"+` |/ __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
| | | | (_| | (__| | | (_| |\ V  V /| |___) | |
|_| |_|\__, |\___|_|  \__,_| \_/\_/ |_|____/|_|
          |_|                            %s

[> %s <]
`).Bold(),
		aurora.BrightYellow(VERSION).Bold(),
		aurora.BrightGreen(DESCRIPTION).Bold(),
	)
)
