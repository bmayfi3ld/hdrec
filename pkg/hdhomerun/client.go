package hdhomerun

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var ErrBusy = errors.New("no tuner available")
var ErrPermanent = errors.New("cannot record channel")

type Channel struct {
	Number string `json:"number"`
	Name   string `json:"name"`
}
type Client struct {
	base string
	http *http.Client
}

func New(base string) (*Client, error) {
	if !validURL(base) {
		return nil, fmt.Errorf("invalid tuner URL")
	}
	return &Client{strings.TrimRight(base, "/"), &http.Client{Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, ResponseHeaderTimeout: 10 * time.Second, IdleConnTimeout: 30 * time.Second}}}, nil
}
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	if !validURL(url) {
		return nil, fmt.Errorf("%w: invalid advertised URL", ErrPermanent)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 200 {
		return res, nil
	}
	defer res.Body.Close()
	if res.StatusCode == 503 && strings.EqualFold(res.Header.Get("X-HDHomeRun-Error"), "805 All Tuners In Use") {
		return nil, ErrBusy
	}
	if res.StatusCode >= 500 {
		return nil, fmt.Errorf("tuner returned HTTP %d", res.StatusCode)
	}
	return nil, fmt.Errorf("%w: HTTP %d", ErrPermanent, res.StatusCode)
}
func (c *Client) metadata(ctx context.Context, url string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(out)
}

type station struct{ GuideNumber, GuideName, URL string }

func (c *Client) lineup(ctx context.Context) ([]station, error) {
	var info struct{ LineupURL string }
	if err := c.metadata(ctx, c.base+"/discover.json", &info); err != nil {
		return nil, err
	}
	var stations []station
	err := c.metadata(ctx, info.LineupURL, &stations)
	return stations, err
}
func (c *Client) Channels(ctx context.Context) ([]Channel, error) {
	ss, err := c.lineup(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Channel, 0, len(ss))
	for _, s := range ss {
		out = append(out, Channel{s.GuideNumber, s.GuideName})
	}
	return out, nil
}

type stream struct {
	io.Reader
	io.Closer
}

func (c *Client) Open(ctx context.Context, number string) (io.ReadCloser, error) {
	ss, err := c.lineup(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range ss {
		if s.GuideNumber != number {
			continue
		}
		res, err := c.get(ctx, s.URL)
		if err != nil {
			return nil, err
		}
		r := bufio.NewReader(res.Body)
		first, err := r.Peek(1)
		if err != nil {
			res.Body.Close()
			return nil, err
		}
		if first[0] != 0x47 {
			res.Body.Close()
			return nil, fmt.Errorf("%w: expected MPEG transport stream", ErrPermanent)
		}
		p, err := r.Peek(377)
		if err != nil {
			res.Body.Close()
			return nil, err
		}
		if p[0] != 0x47 || p[188] != 0x47 || p[376] != 0x47 {
			res.Body.Close()
			return nil, fmt.Errorf("%w: expected MPEG transport stream", ErrPermanent)
		}
		return stream{r, res.Body}, nil
	}
	return nil, fmt.Errorf("%w: unknown channel %q; run hdrec channels", ErrPermanent, number)
}
