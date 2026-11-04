// Package discovery answers LAN discovery probes over UDP so clients find the server
// without knowing its address: a client broadcasts Request to the server's port and
// the server replies (unicast, to the sender) with a small JSON Reply. The client
// takes the server's IP from the reply's source address.
package discovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"strconv"
)

// Request is the exact datagram payload a client broadcasts.
const Request = "plenka-server:discover"

// Reply is the JSON answer. Port is the HTTP port.
type Reply struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Port    int    `json:"port"`
}

// Server listens on one UDP socket.
type Server struct {
	conn  net.PacketConn
	reply []byte
}

// Listen opens UDP addr (e.g. ":8095") and prepares the reply for HTTP port httpPort.
func Listen(addr, version string, httpPort int) (*Server, error) {
	conn, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return nil, err
	}
	reply, _ := json.Marshal(Reply{Service: "plenka-server", Version: version, Port: httpPort})
	return &Server{conn: conn, reply: reply}, nil
}

// Addr is the bound local address.
func (s *Server) Addr() net.Addr { return s.conn.LocalAddr() }

// Serve answers probes until Close. Other datagrams are ignored, so the socket can't be
// used to reflect traffic: the reply is small and goes only to a matching request.
func (s *Server) Serve() error {
	buf := make([]byte, 512)
	req := []byte(Request)
	for {
		n, from, err := s.conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		if bytes.Equal(bytes.TrimSpace(buf[:n]), req) {
			_, _ = s.conn.WriteTo(s.reply, from)
		}
	}
}

// Close stops Serve.
func (s *Server) Close() error { return s.conn.Close() }

// Port parses a decimal port, returning 0 when invalid.
func Port(s string) int {
	p, err := strconv.Atoi(s)
	if err != nil || p <= 0 || p > 65535 {
		return 0
	}
	return p
}
