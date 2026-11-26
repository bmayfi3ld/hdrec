# foreground-recording Specification

## Purpose
Enable reliable foreground capture from a selected local HDHomeRun with observable recovery and unambiguous output files.

## Requirements

### Requirement: Select a known device and channel
The CLI SHALL provide devices and channels as tables or JSON arrays, preserve channel numbers as strings, and select by --device, then HDREC_DEVICE, then local discovery.

#### Scenario: Discovery is ambiguous or empty
- **WHEN** discovery does not yield exactly one distinct device
- **THEN** show available choices or a no-result diagnostic and direct URL guidance without opening a stream.

#### Scenario: Explicit target fails
- **WHEN** the explicit device is unavailable or the exact channel is absent
- **THEN** report the problem without substituting another device or channel.

### Requirement: Capture until interrupted
The recorder SHALL begin acquisition immediately, never overwrite prior output, and write one recording file per uninterrupted stretch of reception, regardless of duration.

#### Scenario: Long uninterrupted recording
- **WHEN** reception continues uninterrupted for eight hours and the user presses Ctrl-C
- **THEN** finalize the single active `.ts.partial` as one `.ts` file without creating another part.

#### Scenario: Host cannot inhibit idle sleep
- **WHEN** inhibition is unavailable during a session
- **THEN** warn once, suggest manual power settings, and continue recording.

### Requirement: Recover only from transient failures
The recorder SHALL wait for busy tuners without eviction, detect EOF and stalls, retry the same device with bounded backoff until interrupted, and stop on invalid media, access denial, or local storage errors.

#### Scenario: Data resumes after a stall
- **WHEN** an interrupted stream starts delivering media again
- **THEN** record a reconnect and observed gap and write resumed data to a new part.

#### Scenario: Storage finalization fails
- **WHEN** a write, sync, close or rename fails
- **THEN** stop with the affected path, preserving prior files without announcing successful finalization or retrying the network.

### Requirement: Observable session
The CLI SHALL present lifecycle events above one width-aware live terminal line showing state, elapsed time, total bytes, file count (including active), bitrate and reconnects as space permits. Redirected plain output SHALL have no cursor controls.

#### Scenario: Machine-readable recording
- **WHEN** record --json runs
- **THEN** stdout contains only one timestamped JSON event per line, including startup errors, start/stop, file open/close, retry, error, recovery, and periodic metrics as applicable; no duplicate prose goes to stderr.

#### Scenario: Terminal becomes narrow
- **WHEN** available columns shrink
- **THEN** shorten the live line, dropping secondary metrics first; ordinary event lines remain in terminal scrollback.

#### Scenario: Live output is blocked
- **WHEN** the terminal or JSON consumer stops accepting output
- **THEN** capture SHALL continue without waiting for live output. Live events and metrics MAY be skipped under backpressure; the session's `session.ndjson` SHALL retain lifecycle events and the final outcome independently. A resumed terminal SHALL show current metrics rather than replay stale snapshots.

### Requirement: Account for shutdown and loss
The recorder SHALL honor Ctrl-C and SIGTERM during capture, waits and backoff, release the tuner and inhibitor, finalize writable output, and report totals, directory, reason, receiving time and unavailable time.

#### Scenario: Interrupted session exits
- **WHEN** a session ends
- **THEN** return 0 for clean capture, 2 for captured data with known missed airtime, or 1 for fatal failure or no data; distinguish initial waits from midstream losses in the summary.
