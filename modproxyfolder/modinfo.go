package modproxyfolder

import (
	"time"
)

// Info is the structure of module information for module proxy protocol.
// Ref: https://golang.org/cmd/go/#hdr-Module_proxy_protocol
type Info struct {
	Version string    // version string
	Time    time.Time // commit time
}
