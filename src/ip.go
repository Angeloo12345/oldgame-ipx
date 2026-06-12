package main

import (
	"encoding/binary"
	"net"
	"strconv"
)

func (transport *IPXTransport) setAddress(address string) {
	host, portStr, _ := net.SplitHostPort(address)
	ip := net.ParseIP(host)

	if len(ip) == 16 {
		transport.Host = binary.BigEndian.Uint32(ip[12:16])
	} else {
		transport.Host = binary.BigEndian.Uint32(ip)
	}

	port, _ := strconv.Atoi(portStr)
	transport.Port = uint16(port)
}

func (transport *IPXTransport) Address() string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, transport.Host)
	return net.JoinHostPort(ip.String(), strconv.Itoa(int(transport.Port)))
}
