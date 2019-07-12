package mediaserver

import (
	native "github.com/notedit/media-server-go/wrapper"
)

func init() {
	native.MediaServerEnableLog(false)
	native.MediaServerInitialize()
}

// EnableLog log or not
func EnableLog(flag bool) {
	native.MediaServerEnableLog(flag)
}

// EnableDebug debug or not
func EnableDebug(flag bool) {
	native.MediaServerEnableDebug(flag)
}

// EnableUltraDebug ultra debug
func EnableUltraDebug(flag bool) {
	native.MediaServerEnableUltraDebug(flag)
}

// SetPortRange set min max port
func SetPortRange(minPort, maxPort int) bool {
	return native.MediaServerSetPortRange(minPort, maxPort)
}
