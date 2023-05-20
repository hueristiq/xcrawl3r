package configuration

import "github.com/logrusorgru/aurora/v3"

const (
	NAME        string = "xcrawl3r"
	VERSION     string = "0.0.0"
	DESCRIPTION string = "A CLI utility to recursively crawl webpages."
)

var (
	BANNER = aurora.Sprintf(
		aurora.BrightBlue(`
                             _ _____      
__  _____ _ __ __ ___      _| |___ / _ __ 
\ \/ / __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
 >  < (__| | | (_| |\ V  V /| |___) | |   
/_/\_\___|_|  \__,_| \_/\_/ |_|____/|_| %s

%s
`).Bold(),
		aurora.BrightYellow("v"+VERSION).Bold(),
		aurora.BrightGreen(DESCRIPTION).Italic(),
	)
)
