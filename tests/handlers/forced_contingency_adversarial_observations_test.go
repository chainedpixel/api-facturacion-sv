package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

// TestAdversarialForcedHeader_HostileValuesNeverEscapeTheAllowedTypes checks that no header value produces a type
// outside 0, 1 and 5.
func TestAdversarialForcedHeader_HostileValuesNeverEscapeTheAllowedTypes(t *testing.T) {
	withForcedContingencyConfig(t, true, "00")

	values := []string{
		"5\t", "\t5", "5\r", "5\n", "5\x00", "\x005", "5 5", "5,1", "1,5", "5;1", "5 ", " 5", "5​",
		"５", "٥", "५", "5０", "-0", "+0", "-5", "-1", "0x5", "0b101", "5e0", "5.", ".5", "5_0",
		"1 ", "01", "001", "+1", "+5", "05", "005", "99999999999999999999", "-99999999999999999999",
		"9223372036854775813", "18446744073709551621", strings.Repeat("9", 10000), strings.Repeat("5", 10000),
		strings.Repeat(" ", 10000) + "5", "5" + strings.Repeat(" ", 10000), "%35", "&#53;", "five", "FIVE", "NaN",
		"Infinity", "true", "null", "<script>alert(5)</script>", "5\"", "'5'", "\"5\"", "[5]", "{\"a\":5}",
	}
	for _, value := range values {
		value := value
		name := value
		if len(name) > 30 {
			name = name[:30] + "..."
		}
		t.Run(strconv.Quote(name), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice", nil)
			req.Header.Set(helpers.ForcedContingencyHeader, value)
			got := utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req))
			assert.Contains(t, []int{0, 1, 5}, got, "header %q produced %d", value, got)
		})
	}
}

// TestAdversarialForcedHeader_NonCanonicalAndRepeatedHeadersOverRealHTTP sends the header through a real HTTP server
// with a lowercase name and repeated lines.
func TestAdversarialForcedHeader_NonCanonicalAndRepeatedHeadersOverRealHTTP(t *testing.T) {
	withForcedContingencyConfig(t, true, "00")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := helpers.RequestContextWithForcedContingency(r)
		_, _ = w.Write([]byte(strconv.Itoa(utils.ForcedContingencyType(ctx))))
	}))
	defer server.Close()

	send := func(t *testing.T, header http.Header) string {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, server.URL, nil)
		require.NoError(t, err)
		req.Header = header
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		buffer := new(bytes.Buffer)
		_, err = buffer.ReadFrom(resp.Body)
		require.NoError(t, err)
		return buffer.String()
	}

	cases := []struct {
		name   string
		header http.Header
		want   string
	}{
		{"lowercase name", http.Header{"x-force-contingency-type": {"5"}}, "5"},
		{"uppercase name", http.Header{"X-FORCE-CONTINGENCY-TYPE": {"1"}}, "1"},
		{"repeated lines first wins", http.Header{"X-Force-Contingency-Type": {"5", "1"}}, "5"},
		{"repeated lines first invalid", http.Header{"X-Force-Contingency-Type": {"abc", "5"}}, "0"},
		{"repeated lines first empty", http.Header{"X-Force-Contingency-Type": {"", "5"}}, "0"},
		{"similar header name", http.Header{"X-Force-Contingency": {"5"}}, "0"},
		{"suffix header name", http.Header{"X-Force-Contingency-Type-2": {"5"}}, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, send(t, tc.header))
		})
	}
}

// TestAdversarialForcedHeader_ConcurrentRequestsDoNotLeakTypes runs many requests with different headers and checks
// every context carries only its own type.
func TestAdversarialForcedHeader_ConcurrentRequestsDoNotLeakTypes(t *testing.T) {
	withForcedContingencyConfig(t, true, "00")

	headers := []string{"1", "5", " 5 ", "abc", "", "3", "0", "-1"}
	expected := []int{1, 5, 5, 0, 0, 0, 0, 0}

	var wg sync.WaitGroup
	failures := make(chan string, 4000)
	for i := 0; i < 2000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			index := i % len(headers)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice", nil)
			req.Header.Set(helpers.ForcedContingencyHeader, headers[index])
			got := utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req))
			if got != expected[index] {
				failures <- headers[index]
			}
			if utils.ForcedContingencyType(req.Context()) != 0 {
				failures <- "original context mutated"
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Errorf("leaked or wrong type for header %q", failure)
	}
}

// TestAdversarialForcedHeader_ToggledConfigNeverForces checks that toggling the configuration between requests is
// honored on every request, never cached.
func TestAdversarialForcedHeader_ToggledConfigNeverForces(t *testing.T) {
	withForcedContingencyConfig(t, true, "00")
	value := "5"
	req := forcedContingencyRequest(&value)
	assert.Equal(t, 5, utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req)))

	withForcedContingencyConfig(t, true, "01")
	assert.Zero(t, utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req)))

	withForcedContingencyConfig(t, false, "00")
	assert.Zero(t, utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req)))

	withForcedContingencyConfig(t, true, "00")
	assert.Equal(t, 5, utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req)))

	for _, ambient := range []string{"0", "000", " 00", "00 ", "01", "1", "", "ZZ"} {
		withForcedContingencyConfig(t, true, ambient)
		assert.Zero(t, utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req)), "ambient %q", ambient)
	}
}
