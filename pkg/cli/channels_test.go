package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bmayfi3ld/hdrec/pkg/hdhomerun"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTuner(t *testing.T) {
	m := http.NewServeMux()
	s := httptest.NewServer(m)
	defer s.Close()
	m.HandleFunc("/discover.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, `{"LineupURL":%q}`, s.URL+"/lineup") })
	m.HandleFunc("/lineup", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"GuideNumber":"7.10","GuideName":"Example","URL":%q}]`, s.URL+"/stream")
	})
	status := make(chan int, 1)
	m.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		v := <-status
		if v == 503 {
			w.Header().Set("X-HDHomeRun-Error", "805 All Tuners In Use")
		}
		w.WriteHeader(v)
		if v == 200 {
			b := make([]byte, 188*3)
			b[0], b[188], b[376] = 71, 71, 71
			w.Write(b)
		}
	})
	ctx := context.Background()
	for _, j := range []bool{false, true} {
		args := []string{"channels", "--device", s.URL}
		want := "7.10\tExample"
		if j {
			args = append(args, "--json")
			want = `[{"number":"7.10","name":"Example"}]`
		}
		var out, err bytes.Buffer
		if Run(ctx, args, &out, &err) != 0 || !strings.Contains(out.String(), want) {
			t.Fatal(out.String(), err.String())
		}
	}
	c, _ := hdhomerun.New(s.URL)
	if _, e := c.Open(ctx, "7.1"); !errors.Is(e, hdhomerun.ErrPermanent) {
		t.Fatal(e)
	}
	for _, v := range []int{200, 503, 403, 502} {
		status <- v
		r, e := c.Open(ctx, "7.10")
		switch v {
		case 200:
			if e != nil {
				t.Fatal(e)
			}
			r.Close()
		case 503:
			if !errors.Is(e, hdhomerun.ErrBusy) {
				t.Fatal(e)
			}
		case 403:
			if !errors.Is(e, hdhomerun.ErrPermanent) {
				t.Fatal(e)
			}
		case 502:
			if e == nil || errors.Is(e, hdhomerun.ErrBusy) || errors.Is(e, hdhomerun.ErrPermanent) {
				t.Fatal(e)
			}
		}
	}
}
