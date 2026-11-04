package discovery

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestServeAnswersRequestOnly(t *testing.T) {
	s, err := Listen("127.0.0.1:0", "1.2.3", 8095)
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve()
	defer s.Close()

	c, err := net.DialUDP("udp4", nil, s.Addr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	buf := make([]byte, 512)

	// Unrelated datagram: no answer.
	_, _ = c.Write([]byte("hello"))
	_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := c.Read(buf); err == nil {
		t.Fatal("answered an unrelated datagram")
	}

	_, _ = c.Write([]byte(Request))
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	var r Reply
	if err := json.Unmarshal(buf[:n], &r); err != nil {
		t.Fatal(err)
	}
	if r != (Reply{Service: "plenka-server", Version: "1.2.3", Port: 8095}) {
		t.Fatalf("reply %+v", r)
	}
}

func TestPort(t *testing.T) {
	for in, want := range map[string]int{"8095": 8095, "0": 0, "x": 0, "70000": 0} {
		if got := Port(in); got != want {
			t.Errorf("Port(%q)=%d want %d", in, got, want)
		}
	}
}
