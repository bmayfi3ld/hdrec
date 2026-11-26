package record

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/bmayfi3ld/hdrec/pkg/hdhomerun"
)

type Source interface {
	Open(context.Context, string) (io.ReadCloser, error)
}
type Request struct{ Channel, OutputDir, Device string }
type Result struct {
	SessionDir  string  `json:"output_dir"`
	Bytes       int64   `json:"bytes"`
	Files       int     `json:"files"`
	Reconnects  int     `json:"reconnects"`
	HadGaps     bool    `json:"had_gaps"`
	Reason      string  `json:"reason"`
	Elapsed     float64 `json:"elapsed_s"`
	Receiving   float64 `json:"receiving_s"`
	Unavailable float64 `json:"unavailable_s"`
	InitialWait float64 `json:"initial_wait_s"`
}
type Event struct {
	Time    time.Time `json:"ts"`
	Kind    string    `json:"event"`
	Path    string    `json:"path,omitempty"`
	Code    string    `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
	Delay   float64   `json:"delay_s,omitempty"`
	Result  *Result   `json:"result,omitempty"`
	State   string    `json:"state,omitempty"`
	Channel string    `json:"channel,omitempty"`
	Device  string    `json:"device,omitempty"`
	Rate    int64     `json:"rate_bps,omitempty"`
	Gap     float64   `json:"gap_s,omitempty"`
}

func Run(ctx context.Context, src Source, req Request, report func(Event)) (Result, error) {
	return run(ctx, src, req, report, 15*time.Second, time.Second)
}

func run(ctx context.Context, src Source, req Request, report func(Event), timeout, base time.Duration) (result Result, err error) {
	started := time.Now()
	dir, err := newSession(req.OutputDir)
	if err != nil {
		return result, err
	}
	result.SessionDir = dir
	log, err := os.OpenFile(filepath.Join(dir, "session.ndjson"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, err
	}
	encoder := json.NewEncoder(log)
	var logErr error
	emit := func(e Event) {
		e.Time = time.Now()
		if logErr == nil && e.Kind != "metrics" {
			logErr = encoder.Encode(e)
		}
		if report != nil {
			report(e)
		}
	}
	emit(Event{Kind: "session_started", Path: dir, Channel: req.Channel, Device: req.Device})
	defer func() {
		result.Elapsed = time.Since(started).Seconds()
		result.Unavailable = max(0, result.Elapsed-result.Receiving)
		if result.Files == 0 {
			result.InitialWait = result.Elapsed
		}
		if logErr != nil {
			result.Reason = "log_error"
			err = errors.Join(err, logErr)
		}
		emit(Event{Kind: "session_stopped", Code: result.Reason, Result: &result})
		err = errors.Join(err, logErr, log.Sync(), log.Close())
	}()
	buf := make([]byte, 64*1024)
	delay := base
	var previousEnd, lastMetric time.Time
	var lastBytes int64
	for ctx.Err() == nil && logErr == nil {
		attempt, cancel := context.WithCancel(ctx)
		watchdog := time.AfterFunc(timeout, cancel)
		stream, readErr := src.Open(attempt, req.Channel)
		var p *part
		var since, lastByte time.Time
		if readErr == nil {
			for attempt.Err() == nil && logErr == nil {
				n, e := stream.Read(buf)
				if n > 0 {
					watchdog.Reset(timeout)
					now := time.Now()
					lastByte = now
					if p == nil {
						p, err = openPart(dir, result.Files+1, now)
						if err != nil {
							break
						}
						if result.Files > 0 {
							result.Reconnects++
							emit(Event{Kind: "reconnected", Gap: now.Sub(previousEnd).Seconds()})
						}
						if result.Files == 0 {
							result.InitialWait = now.Sub(started).Seconds()
						}
						result.Files++
						since = now
						emit(Event{Kind: "file_opened", Path: p.file.Name()})
					}
					var written int
					written, err = p.write(buf[:n], now)
					result.Bytes += int64(written)
					if lastMetric.IsZero() || now.Sub(lastMetric) >= time.Second {
						snapshot := result
						snapshot.Elapsed = now.Sub(started).Seconds()
						snapshot.Receiving += now.Sub(since).Seconds()
						snapshot.Unavailable = max(0, snapshot.Elapsed-snapshot.Receiving)
						var rate int64
						if !lastMetric.IsZero() {
							rate = int64(float64((result.Bytes-lastBytes)*8) / now.Sub(lastMetric).Seconds())
						}
						emit(Event{Kind: "metrics", State: "recording", Result: &snapshot, Rate: rate})
						lastMetric, lastBytes = now, result.Bytes
					}
					if err != nil {
						break
					}
				}
				if e != nil {
					readErr = e
					break
				}
			}
		}
		watchdog.Stop()
		cancel()
		if stream != nil {
			stream.Close()
		}
		if err != nil {
			if p != nil {
				p.file.Close()
			}
			result.Reason = "write_error"
			emit(Event{Kind: "error", Code: result.Reason, Message: err.Error()})
			return result, err
		}
		if p != nil && ctx.Err() != nil {
			emit(Event{Kind: "finalizing"})
		}
		if p != nil {
			result.Receiving += lastByte.Sub(since).Seconds()
			previousEnd = lastByte
			path, e := p.finalize()
			if e != nil {
				result.Reason = "write_error"
				emit(Event{Kind: "error", Code: result.Reason, Path: path, Message: e.Error()})
				return result, e
			}
			emit(Event{Kind: "file_closed", Path: path})
			if time.Since(since) >= 30*time.Second {
				delay = base
			}
		}
		if ctx.Err() != nil || logErr != nil {
			break
		}
		if errors.Is(readErr, hdhomerun.ErrPermanent) {
			result.Reason = "device_error"
			emit(Event{Kind: "error", Code: result.Reason, Message: readErr.Error()})
			return result, readErr
		}
		code := "stream_lost"
		if errors.Is(readErr, hdhomerun.ErrBusy) {
			code = "tuner_busy"
		}
		result.HadGaps = true
		message := "No stream data"
		if readErr != nil {
			message = readErr.Error()
		}
		emit(Event{Kind: "error", Code: code, Message: message})
		wait := delay - delay/4 + time.Duration(rand.Int63n(max(1, int64(delay/4))))
		state := "reconnecting"
		if result.Files == 0 && code == "tuner_busy" {
			state = "waiting_for_tuner"
		}
		emit(Event{Kind: "retry_scheduled", Code: code, Delay: wait.Seconds(), State: state})
		snapshot := result
		snapshot.Elapsed = time.Since(started).Seconds()
		snapshot.Unavailable = max(0, snapshot.Elapsed-snapshot.Receiving)
		emit(Event{Kind: "metrics", State: state, Result: &snapshot})
		lastMetric = time.Time{}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
		delay = min(delay*2, 30*time.Second)
	}
	result.Reason = "user_stop"
	return result, nil
}
