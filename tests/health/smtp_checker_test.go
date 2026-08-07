package health

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/health/checkers"
	test "github.com/chainedpixel/ordo-factus/tests"
)

// fakeSMTP responds to a minimal handshake. authOK toggles whether AUTH
// PLAIN is accepted, letting us simulate bad credentials.
func fakeSMTP(t *testing.T, authOK bool) (host, port string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				w := bufio.NewWriter(c)
				r := bufio.NewReader(c)
				write := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }

				write("220 fake.local ESMTP")
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					trimmed := strings.TrimRight(line, "\r\n")
					upper := strings.ToUpper(trimmed)
					switch {
					case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
						write("250-fake.local")
						write("250-AUTH PLAIN LOGIN")
						write("250 SIZE 10240000")
					case strings.HasPrefix(upper, "AUTH PLAIN"):
						if authOK {
							write("235 2.7.0 Authentication successful")
						} else {
							write("535 5.7.8 Authentication credentials invalid")
							return
						}
					case upper == "QUIT":
						write("221 bye")
						return
					case upper == "NOOP":
						write("250 OK")
					default:
						write("250 OK")
					}
				}
			}(conn)
		}
	}()
	h, p, _ := net.SplitHostPort(ln.Addr().String())
	return h, p, func() { _ = ln.Close() }
}

func TestSMTPChecker_UpWhenNotConfigured(t *testing.T) {
	test.TestMain(t)
	config.SMTP.Host = ""

	got := checkers.NewSMTPChecker().Check()
	if got.Status != constants.StatusUp {
		t.Fatalf("expected UP, got %q", got.Status)
	}
	if !strings.Contains(strings.ToLower(got.Details), "no configurado") &&
		!strings.Contains(strings.ToLower(got.Details), "not configured") {
		t.Errorf("details should mention not configured, got %q", got.Details)
	}
}

func TestSMTPChecker_UpWhenAuthSucceeds(t *testing.T) {
	test.TestMain(t)
	host, port, stop := fakeSMTP(t, true)
	defer stop()

	config.SMTP.Host = host
	config.SMTP.Port = port
	config.SMTP.TLS = false
	config.SMTP.Username = "user"
	config.SMTP.Password = "good"
	defer func() { config.SMTP.Host = "" }()

	got := checkers.NewSMTPChecker().Check()
	if got.Status != constants.StatusUp {
		t.Fatalf("expected UP, got %q (details=%q)", got.Status, got.Details)
	}
}

func TestSMTPChecker_DownWhenAuthFails(t *testing.T) {
	test.TestMain(t)
	host, port, stop := fakeSMTP(t, false)
	defer stop()

	config.SMTP.Host = host
	config.SMTP.Port = port
	config.SMTP.TLS = false
	config.SMTP.Username = "user"
	config.SMTP.Password = "bad"
	defer func() { config.SMTP.Host = "" }()

	got := checkers.NewSMTPChecker().Check()
	if got.Status != constants.StatusDown {
		t.Fatalf("expected DOWN with bad credentials, got %q (details=%q)", got.Status, got.Details)
	}
	if !strings.Contains(strings.ToLower(got.Details), "credenciales") &&
		!strings.Contains(strings.ToLower(got.Details), "credentials") {
		t.Errorf("details should mention credentials cleanly, got %q", got.Details)
	}
	for _, leak := range []string{"535", "5.7.8", "google.com", "gsmtp"} {
		if strings.Contains(got.Details, leak) {
			t.Errorf("details leaks raw provider info %q: %s", leak, got.Details)
		}
	}
}

func TestSMTPChecker_DownWhenHostUnreachable(t *testing.T) {
	test.TestMain(t)
	config.SMTP.Host = "127.0.0.1"
	config.SMTP.Port = strconv.Itoa(1)
	config.SMTP.TLS = false
	defer func() { config.SMTP.Host = "" }()

	got := checkers.NewSMTPChecker().Check()
	if got.Status != constants.StatusDown {
		t.Fatalf("expected DOWN, got %q (details=%q)", got.Status, got.Details)
	}
	if !strings.Contains(strings.ToLower(got.Details), "conectar") &&
		!strings.Contains(strings.ToLower(got.Details), "reach") {
		t.Errorf("details should mention connectivity cleanly, got %q", got.Details)
	}
}
