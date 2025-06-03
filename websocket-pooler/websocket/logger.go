package websocket

import (
	stdlog "log" // Import with a different name
)

// This file provides a package-level logger for the websocket package
// It can be extended to use a more sophisticated logging system if needed
var log = stdlog.New(stdlog.Writer(), "[websocket] ", stdlog.LstdFlags)
