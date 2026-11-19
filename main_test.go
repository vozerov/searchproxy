package main

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"
)

// startProxy serves handle() on a random port in front of target.
func startProxy(t *testing.T, target string) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go handle(c, target)
		}
	}()
	return l.Addr().String()
}

// startBackend answers one line with "pong\n"; closeAfter closes the connection
// right after the reply, like a server dropping an idle keepalive connection.
func startBackend(t *testing.T, closeAfter bool) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				r := bufio.NewReader(c)
				if _, err := r.ReadString('\n'); err != nil {
					return
				}
				_, _ = c.Write([]byte("pong\n"))
				if closeAfter {
					return
				}
				_, _ = io.Copy(io.Discard, r)
			}()
		}
	}()
	return l.Addr().String()
}

func TestBackendCloseReachesClient(t *testing.T) {
	addr := startProxy(t, startBackend(t, true /* closeAfter */))
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if _, err := c.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	got, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("want EOF after backend close, got %v (data %q)", err, got)
	}
	if string(got) != "pong\n" {
		t.Fatalf("got %q, want %q", got, "pong\n")
	}
}

func TestClientHalfCloseKeepsResponse(t *testing.T) {
	addr := startProxy(t, startBackend(t, false /* closeAfter */))
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if _, err := c.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	if err := c.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	got, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("want response and EOF, got %v (data %q)", err, got)
	}
	if string(got) != "pong\n" {
		t.Fatalf("got %q, want %q", got, "pong\n")
	}
}
