package cache

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/cache"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

func newPrefixedStore(t *testing.T, prefix string) (ports.CacheManager, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	store, err := cache.NewRedisTokenCache(&config.RedisConfig{Host: server.Host(), Port: server.Port(), Prefix: prefix}, nil)
	require.NoError(t, err)
	return store, server
}

// TestRedisKeyPrefixNormalization verifies the configured prefix always ends with a single separator.
func TestRedisKeyPrefixNormalization(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"   ":      "",
		":":        "",
		"app":      "app:",
		"app:":     "app:",
		"app::":    "app:",
		" app-v2 ": "app-v2:",
	}
	for raw, expected := range cases {
		assert.Equal(t, expected, (&config.RedisConfig{Prefix: raw}).KeyPrefix(), "prefix %q", raw)
	}
}

// TestRedisTokenCacheStoresKeysUnderPrefix verifies every operation reads and writes namespaced keys.
func TestRedisTokenCacheStoresKeysUnderPrefix(t *testing.T) {
	test.TestMain(t)
	store, server := newPrefixedStore(t, "app-v2")

	require.NoError(t, store.Set("hacienda:token:abc", []byte("value"), time.Minute))
	assert.True(t, server.Exists("app-v2:hacienda:token:abc"))
	assert.False(t, server.Exists("hacienda:token:abc"))

	got, err := store.Get("hacienda:token:abc")
	require.NoError(t, err)
	assert.Equal(t, "value", got)

	require.NoError(t, store.RPush("metrics:a:durations", []byte("1")))
	length, err := store.LLen("metrics:a:durations")
	require.NoError(t, err)
	assert.Equal(t, int64(1), length)
	assert.True(t, server.Exists("app-v2:metrics:a:durations"))

	require.NoError(t, store.Set("token:xyz", []byte("{}"), time.Minute))
	require.NoError(t, store.Delete("xyz"))
	assert.False(t, server.Exists("app-v2:token:xyz"))
}

// TestRedisTokenCacheIgnoresUnprefixedLegacyKeys verifies keys written without the prefix are invisible to the adapter.
func TestRedisTokenCacheIgnoresUnprefixedLegacyKeys(t *testing.T) {
	test.TestMain(t)
	store, server := newPrefixedStore(t, "app-v2")

	require.NoError(t, server.Set("hacienda:token:legacy", "stale"))
	require.NoError(t, server.Set("metrics:legacy:durations", "stale"))

	_, err := store.Get("hacienda:token:legacy")
	assert.Error(t, err)

	require.NoError(t, store.RPush("metrics:current:durations", []byte("1")))
	keys, err := store.ScanKeys("metrics:*:durations")
	require.NoError(t, err)
	assert.Equal(t, []string{"metrics:current:durations"}, keys)
}

// TestRedisTokenCacheWithoutPrefixKeepsKeys verifies an empty prefix preserves the original key layout.
func TestRedisTokenCacheWithoutPrefixKeepsKeys(t *testing.T) {
	test.TestMain(t)
	store, server := newPrefixedStore(t, "")

	require.NoError(t, store.Set("hacienda:token:abc", []byte("value"), time.Minute))
	assert.True(t, server.Exists("hacienda:token:abc"))
}
