## Context
Greenfield Go module; the device supplies a lineup and an HTTP stream. Confirm actual media and busy responses on supported hardware. See proposal.md for motivation.

## Goals / Non-Goals
Keep capture independent of rendering and accept cancellation at every blocking network operation. No transcoding, scheduling, service installation, automatic failover, or crash relaunch.

## Decisions
- Use native IPv4 UDP discovery across active interfaces with a two-second window; deduplicate by device ID. Use the advertised lineup and stream URLs rather than constructing channel paths.
- Copy verified MPEG-TS directly with bounded memory. FFmpeg adds an unnecessary failure boundary for native capture. Rotate on whole transport packets, not keyframes; do not reconnect on ordinary rotation.
- Bound dial/header waits and watchdog stalled reads instead of imposing a whole-session timeout. Retry transient failures with jittered exponential backoff from 1 to 30 seconds, resetting after sustained reception. Close each failed response. Distinguish device-specific busy from other service errors.
- Create a unique directory under ./recordings or --out-dir. Periodically sync active bytes; flush, sync, close, and rename before announcing file_closed. Persist lifecycle events to session.ndjson. Never append new reception to an interrupted part.
- One set of recorder counters and events feeds every view. Refresh terminal metrics once per second and NDJSON metrics every ten seconds. Decouple terminal redraws from reads; keep event order and use bounded buffering.
- Use caffeinate on macOS, systemd-inhibit on Linux where available, and a Windows execution-state request on a locked OS thread. Tie the inhibitor lifetime to the session.

### Package boundaries (proposed Go signatures)
```go
// cmd/hdrec/main.go: install signal context; exit with cli.Run result.
// pkg/cli: select device, choose renderer, report outcomes.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int

// pkg/hdhomerun
type Device struct { ID, URL string }
type Channel struct { Number, Name string }
func Discover(ctx context.Context) ([]Device, error)
func New(baseURL string) (*Client, error)
func (c *Client) Channels(ctx context.Context) ([]Channel, error)
func (c *Client) Open(ctx context.Context, number string) (io.ReadCloser, error)

// pkg/record: session lifecycle, files, retries, events and summary.
type Source interface { Open(context.Context, string) (io.ReadCloser, error) }
type Request struct { Channel, OutputDir string }
func Run(ctx context.Context, src Source, req Request, report func(Event)) (Result, error)

// pkg/power: close handle to release inhibitor.
func Inhibit(ctx context.Context) (io.Closer, error)
```

## Risks / Trade-offs
- Lost reception → report observed gaps; retry cannot restore missed airtime.
- Power loss → finalized parts remain separate; the last partial may be truncated.
- Broadcast blocked → offer direct URL selection, not a subnet scan.
- Lid closure and power policy → warn about limits; do not promise to wake a machine.
