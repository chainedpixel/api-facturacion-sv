package notifier

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func newRedisFromMini(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	c := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = c.Close() })
	return c, mr
}

func TestCooldownAllowsFirstThenBlocks(t *testing.T) {
	test.TestMain(t)

	rdb, _ := newRedisFromMini(t)
	cd := email.NewRedisCooldown(rdb, 5*time.Minute)

	ok, err := cd.ShouldNotify(context.Background(), "e", "agg")
	if err != nil || !ok {
		t.Fatalf("first call: ok=%v err=%v", ok, err)
	}
	ok, err = cd.ShouldNotify(context.Background(), "e", "agg")
	if err != nil || ok {
		t.Fatalf("second call (cooldown): ok=%v err=%v", ok, err)
	}
}

func TestCooldownExpiresAndAllowsAgain(t *testing.T) {
	test.TestMain(t)

	rdb, mr := newRedisFromMini(t)
	cd := email.NewRedisCooldown(rdb, 100*time.Millisecond)

	ok, _ := cd.ShouldNotify(context.Background(), "e", "agg")
	if !ok {
		t.Fatal("first call should be allowed")
	}
	mr.FastForward(200 * time.Millisecond)

	ok, err := cd.ShouldNotify(context.Background(), "e", "agg")
	if err != nil || !ok {
		t.Fatalf("after TTL: ok=%v err=%v", ok, err)
	}
}

func TestCooldownIsKeyedByEventAndAggregate(t *testing.T) {
	test.TestMain(t)

	rdb, _ := newRedisFromMini(t)
	cd := email.NewRedisCooldown(rdb, 5*time.Minute)

	ok1, _ := cd.ShouldNotify(context.Background(), "e", "branch:1")
	ok2, _ := cd.ShouldNotify(context.Background(), "e", "branch:2")
	if !ok1 || !ok2 {
		t.Fatalf("different aggregates should not collide: %v %v", ok1, ok2)
	}
	okOther, _ := cd.ShouldNotify(context.Background(), "other", "branch:1")
	if !okOther {
		t.Fatal("different event names should not collide")
	}
}
