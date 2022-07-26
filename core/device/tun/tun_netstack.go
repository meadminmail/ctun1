//go:build linux

package tun

import (
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type TUN struct {
	stack.LinkEndpoint

	fd   int
	mtu  uint32
	name string
}
