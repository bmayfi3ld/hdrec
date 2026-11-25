## 1. Device access
- [ ] 1.1 Create the Go module and CLI entrypoint; verify `go build ./cmd/hdrec`.
- [ ] 1.2 Implement discovery and explicit device selection; test zero, one, duplicate and multiple replies and override precedence.
- [ ] 1.3 Implement lineup and stream access; verify exact string selection, listing formats and error classification with a fake HTTP tuner. Confirm stream format and busy response on real hardware without publishing device identifiers.

## 2. Durable capture
- [ ] 2.1 Implement unique sessions, partial files, periodic sync and packet-aligned rotation; test collisions, injected disk failures and rotation on a single open stream.
- [ ] 2.2 Implement busy waits, watchdog and backoff; simulate busy, EOF, silent stall, resumed bytes and cancellation during backoff. Assert gaps, part boundaries and reconnect counts.
- [ ] 2.3 Add session event log, result accounting and shutdown; test clean, incomplete, fatal and empty outcomes including exit codes.
- [ ] 2.4 Add platform-specific sleep inhibition with release on all exits; cross-build for supported platforms and verify warning behavior on acquisition failure.

## 3. Reporting and acceptance
- [ ] 3.1 Render plain, terminal and NDJSON output from the same events; test resize, startup failure, slow rendering and counter agreement across renderers.
- [ ] 3.2 Document discovery fallback, recording, storage, JSON and interruption with generic examples; run documented commands against a test device.
- [ ] 3.3 Run an overnight hardware test with one network interruption. Verify sleep inhibition, playback across parts, disk usage and reported gap. Separately kill a process during writing and verify earlier parts survive.
