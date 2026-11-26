## Context
Greenfield Go module; the device supplies a lineup and an HTTP stream. Confirm actual media and busy responses on supported hardware. See proposal.md for motivation.

## Goals / Non-Goals
Keep capture independent of rendering and accept cancellation at every blocking network operation. No transcoding, scheduling, service installation, automatic failover, or crash relaunch.

## Decisions
- Use native IPv4 UDP discovery across active interfaces with a two-second window; deduplicate by device ID. Use the advertised lineup and stream URLs rather than constructing channel paths.
- Copy verified MPEG-TS directly with bounded memory. FFmpeg adds an unnecessary failure boundary for native capture. Keep one file per uninterrupted stretch of reception, regardless of duration. Finalize it on disconnect or graceful stop; open the next part only when media resumes.
- Bound dial/header waits and watchdog stalled reads instead of imposing a whole-session timeout. Retry transient failures with jittered exponential backoff from 1 to 30 seconds, resetting after sustained reception. Close each failed response. Distinguish device-specific busy from other service errors.
- Create a unique directory under ./recordings or --out-dir. Periodically sync active bytes; flush, sync, close, and rename before announcing file_closed. Persist lifecycle events to session.ndjson. Never append new reception to an interrupted part.
- One set of recorder counters and events feeds every view. Refresh terminal metrics once per second and NDJSON metrics every ten seconds. Decouple live output from reads. Preserve lifecycle event order in session.ndjson, use bounded best-effort live output, and coalesce metrics so a resumed terminal sees current status. Operators can follow session.ndjson from a second terminal and check whether the active partial is growing; no attach command.
- Use caffeinate on macOS, systemd-inhibit on Linux where available, and a Windows execution-state request on a locked OS thread. Tie the inhibitor lifetime to the session.

### Recording terminal layout

```text
18:00:01  Opened part-000001.ts.partial
19:42:08  Connection lost; saved part-000001.ts
19:42:11  Reconnected after 3s; opened part-000002.ts.partial
RECORDING  7.1 | 01:42:15 elapsed | 2 files | 6.3 GiB | 8.8 Mb/s | 1 reconnect
```

Only the last line redraws. Clear it to print an event, then redraw it below the event. Events scroll into normal terminal history; no fixed-height pane is needed. On narrow terminals, drop bitrate and reconnect count before core metrics. With `--json`, append one JSON event per line without redraws.

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
