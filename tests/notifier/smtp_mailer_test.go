package notifier

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	test "github.com/chainedpixel/ordo-factus/tests"
)

type fakeSMTPServer struct {
	listener net.Listener
	data     strings.Builder
	mu       sync.Mutex
	done     chan struct{}
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSMTPServer{listener: ln, done: make(chan struct{})}
	go s.accept()
	return s
}

func (s *fakeSMTPServer) addr() string { return s.listener.Addr().String() }

func (s *fakeSMTPServer) close() { _ = s.listener.Close() }

func (s *fakeSMTPServer) sent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.String()
}

func (s *fakeSMTPServer) accept() {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	defer close(s.done)

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	write := func(line string) { _, _ = w.WriteString(line + "\r\n"); _ = w.Flush() }

	write("220 fake.local ESMTP")
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if inData {
			s.mu.Lock()
			s.data.WriteString(line)
			s.mu.Unlock()
			if trimmed == "." {
				inData = false
				write("250 OK")
			}
			continue
		}
		switch {
		case strings.HasPrefix(strings.ToUpper(trimmed), "EHLO"), strings.HasPrefix(strings.ToUpper(trimmed), "HELO"):
			write("250-fake.local")
			write("250 SIZE 10240000")
		case strings.HasPrefix(strings.ToUpper(trimmed), "MAIL FROM"):
			write("250 OK")
		case strings.HasPrefix(strings.ToUpper(trimmed), "RCPT TO"):
			write("250 OK")
		case strings.ToUpper(trimmed) == "DATA":
			write("354 Send data")
			inData = true
		case strings.ToUpper(trimmed) == "QUIT":
			write("221 bye")
			return
		case strings.ToUpper(trimmed) == "RSET":
			write("250 OK")
		default:
			write("250 OK")
		}
	}
}

func setupSMTPConfig(t *testing.T, host string, port int) {
	t.Helper()
	test.TestMain(t)
	config.SMTP.Host = host
	config.SMTP.Port = strconv.Itoa(port)
	config.SMTP.From = "Ordo <noreply@local>"
	config.SMTP.TLS = false
	config.Server.AdminEmail = "admin@test.local"
}

func TestSMTPMailerSkipsWhenHostEmpty(t *testing.T) {
	test.TestMain(t)
	config.SMTP.Host = ""
	config.Server.AdminEmail = "admin@test.local"
	m := email.NewSMTPMailer()
	if err := m.SendAdminAlert(context.Background(), "s", "<b>x</b>", "x"); err != nil {
		t.Fatalf("expected graceful skip, got %v", err)
	}
}

func TestSMTPMailerSkipsWhenAdminEmpty(t *testing.T) {
	test.TestMain(t)
	config.SMTP.Host = "localhost"
	config.Server.AdminEmail = ""
	config.SMTP.From = "from@x"
	m := email.NewSMTPMailer()
	if err := m.SendAdminAlert(context.Background(), "s", "<b>x</b>", "x"); err != nil {
		t.Fatalf("expected graceful skip, got %v", err)
	}
}

func TestSMTPMailerSendsThroughFakeServer(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	host, portStr, _ := net.SplitHostPort(srv.addr())
	port, _ := strconv.Atoi(portStr)
	setupSMTPConfig(t, host, port)

	m := email.NewSMTPMailer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SendAdminAlert(ctx, "Asunto de prueba", "<p>HTML body</p>", "Plain body"); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case <-srv.done:
	case <-time.After(2 * time.Second):
	}

	got := srv.sent()
	for _, want := range []string{"Asunto de prueba", "HTML body", "Plain body"} {
		if !strings.Contains(got, want) {
			t.Errorf("server data missing %q\n--- got ---\n%s", want, got)
		}
	}
}
