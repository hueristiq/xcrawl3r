package configuration

import "github.com/logrusorgru/aurora/v3"

const (
	NAME    = "xcrawl3r"
	VERSION = "0.2.0"
)

var BANNER = aurora.Sprintf(
	aurora.BrightBlue(`
                             _ _____      
__  _____ _ __ __ ___      _| |___ / _ __ 
\ \/ / __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
 >  < (__| | | (_| |\ V  V /| |___) | |   
/_/\_\___|_|  \__,_| \_/\_/ |_|____/|_| 
                                    %s`).Bold(),
	aurora.BrightRed("v"+VERSION).Bold(),
)
