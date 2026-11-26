## 1. Device access
- [x] 1.1 Create the Go module and CLI entrypoint; verify `go build ./cmd/hdrec`.
- [x] 1.2 Implement discovery and explicit device selection; test zero, one, duplicate and multiple replies and override precedence.
- [x] 1.3 Implement lineup and stream access; verify exact string selection, listing formats and error classification with a fake HTTP tuner. Confirm stream format and busy response on real hardware without publishing device identifiers.

## 2. Durable capture
- [x] 2.1 Implement unique sessions, partial files, periodic sync and finalization; test collisions, injected disk failures and uninterrupted capture beyond 15 minutes producing one finalized file.
- [x] 2.2 Implement busy waits, watchdog and backoff; simulate busy, EOF, silent stall, resumed bytes and cancellation during backoff. Assert that one disconnect followed by resumed media and graceful stop yields exactly two finalized files, one reported gap and one reconnect.
- [x] 2.3 Add session event log, result accounting and shutdown; test clean, incomplete, fatal and empty outcomes including exit codes.
- [x] 2.4 Add platform-specific sleep inhibition with release on all exits; cross-build for supported platforms and verify warning behavior on acquisition failure.

## 3. Reporting and acceptance
- [x] 3.1 Render plain, terminal and NDJSON output from the same events; test resize, startup failure, slow rendering and counter agreement across renderers.
- [x] 3.2 Document discovery fallback, recording, storage, JSON and interruption with generic examples; run documented commands against a test device.
- [x] 3.3 Accept long-run capture and recovery on hardware. Verified a five-hour recording, desktop inhibitor registration, a separate controlled interruption with two decodable parts and a reported gap, and playback of the partial left by a forced process kill. No completed full-day or power-loss test is claimed.
