// +build darwin

package quic

import "golang.org/x/sys/unix"

const msgTypeIPTOS = unix.IP_RECVTOS

const (
	ipv4RECVPKTINFO = 0x1a
	ipv6RECVPKTINFO = 0x3d
)

const (
	msgTypeIPv4PKTINFO = 0x1a
	msgTypeIPv6PKTINFO = 0x2e
)
