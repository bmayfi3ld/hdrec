package hdhomerun

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net"
	"net/url"
	"sort"
	"time"
)

type Device struct {
	ID  string `json:"id"`
	URL string `json:"base_url"`
}

func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

// HDHomeRun packets use network byte order except for their trailing CRC32.
func packet(kind uint16, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint16(b, kind)
	binary.BigEndian.PutUint16(b[2:], uint16(len(payload)))
	copy(b[4:], payload)
	binary.LittleEndian.PutUint32(b[len(b)-4:], crc32.ChecksumIEEE(b[:len(b)-4]))
	return b
}
func reply(b []byte) (d Device, ok bool) {
	if len(b) < 8 || binary.BigEndian.Uint16(b) != 3 || int(binary.BigEndian.Uint16(b[2:])) != len(b)-8 || binary.LittleEndian.Uint32(b[len(b)-4:]) != crc32.ChecksumIEEE(b[:len(b)-4]) {
		return
	}
	p := b[4 : len(b)-4]
	for len(p) > 0 {
		if len(p) < 2 {
			return
		}
		tag, n := p[0], int(p[1])
		p = p[2:]
		if n >= 128 {
			if len(p) == 0 {
				return
			}
			n = n&127 | int(p[0])<<7
			p = p[1:]
		}
		if n > len(p) {
			return
		}
		switch tag {
		case 2:
			if n == 4 {
				d.ID = fmt.Sprintf("%08X", binary.BigEndian.Uint32(p))
			}
		case 0x2a:
			d.URL = string(p[:n])
		}
		p = p[n:]
	}
	return d, d.ID != "" && validURL(d.URL)
}

func Discover(ctx context.Context) ([]Device, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return nil, err
	}
	request := packet(2, []byte{1, 4, 0, 0, 0, 1, 2, 4, 255, 255, 255, 255})
	sent := false
	var lastErr error
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, e := iface.Addrs()
		if e != nil {
			lastErr = e
			continue
		}
		for _, a := range addrs {
			sub, ok := a.(*net.IPNet)
			if !ok || sub.IP.To4() == nil || len(sub.Mask) != 4 {
				continue
			}
			dst := append(net.IP(nil), sub.IP.To4()...)
			for i := range dst {
				dst[i] |= ^sub.Mask[i]
			}
			if _, e = conn.WriteToUDP(request, &net.UDPAddr{IP: dst, Port: 65001}); e != nil {
				lastErr = e
			} else {
				sent = true
			}
		}
	}
	if !sent && lastErr != nil {
		return nil, lastErr
	}
	seen := map[string]Device{}
	buf := make([]byte, 65535)
	for sent {
		n, _, e := conn.ReadFromUDP(buf)
		if e != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if t, ok := e.(net.Error); ok && t.Timeout() {
				break
			}
			return nil, e
		}
		if d, ok := reply(buf[:n]); ok {
			seen[d.ID] = d
		}
	}
	out := make([]Device, 0, len(seen))
	for _, d := range seen {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func Select(ctx context.Context, flagURL, envURL string, discover func(context.Context) ([]Device, error)) (Device, error) {
	s := flagURL
	if s == "" {
		s = envURL
	}
	if s != "" {
		if !validURL(s) {
			return Device{}, fmt.Errorf("invalid device URL: use --device http://HOST")
		}
		return Device{URL: s}, nil
	}
	ds, err := discover(ctx)
	if err != nil {
		return Device{}, err
	}
	if len(ds) == 1 {
		return ds[0], nil
	}
	if len(ds) == 0 {
		return Device{}, fmt.Errorf("no tuners discovered; use --device http://HOST")
	}
	return Device{}, fmt.Errorf("multiple tuners: %v; select one with --device URL", ds)
}
