// Package localbus centralizes local Bus host defaults shared by CLI modules.
package localbus

import (
	"net"
	"os"
	"strconv"
	"strings"
)

// DefaultHost is the loopback host used when BUS_HOST is unset.
// Used by: HostFromEnv and modules that need to display the fallback host.
const DefaultHost = "127.0.0.1"

// HostFromEnv returns BUS_HOST when set, otherwise DefaultHost.
// Used by: local service bind defaults and derived local Bus API URLs.
func HostFromEnv() string {
	if host := strings.TrimSpace(os.Getenv("BUS_HOST")); host != "" {
		return host
	}
	return DefaultHost
}

// Addr returns a host:port bind address using BUS_HOST when set.
// Used by: Bus service command default --addr/--listen values.
func Addr(port int) string {
	return net.JoinHostPort(HostFromEnv(), strconv.Itoa(port))
}

// HTTPURL returns an HTTP URL using BUS_HOST when set.
// Used by: local Bus client URL defaults.
func HTTPURL(port int, path string) string {
	path = strings.TrimSpace(path)
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "http://" + net.JoinHostPort(HostFromEnv(), strconv.Itoa(port)) + path
}
