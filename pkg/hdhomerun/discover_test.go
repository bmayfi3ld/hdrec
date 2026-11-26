package hdhomerun

import (
	"context"
	"testing"
)

func TestRepliesAndSelection(t *testing.T) {
	ctx := context.Background()
	base := "http://tuner.example"
	payload := append([]byte{2, 4, 1, 2, 3, 4, 0x2a, byte(len(base))}, base...)
	b := packet(3, payload)
	seen := map[string]Device{}
	for i := 0; i < 2; i++ {
		d, ok := reply(b)
		if !ok || d.URL != base || d.ID != "01020304" {
			t.Fatal(d, ok)
		}
		seen[d.ID] = d
	}
	if len(seen) != 1 {
		t.Fatal(seen)
	}
	b[4] ^= 1
	if _, ok := reply(b); ok {
		t.Fatal("accepted invalid CRC")
	}
	for n := 0; n <= 2; n++ {
		_, err := Select(ctx, "", "", func(context.Context) ([]Device, error) { return make([]Device, n), nil })
		if (err == nil) != (n == 1) {
			t.Fatal(n, err)
		}
	}
	for _, p := range [][3]string{{base, "http://env.example", base}, {"", base, base}} {
		d, e := Select(ctx, p[0], p[1], nil)
		if e != nil || d.URL != p[2] {
			t.Fatal(d, e)
		}
	}
}
