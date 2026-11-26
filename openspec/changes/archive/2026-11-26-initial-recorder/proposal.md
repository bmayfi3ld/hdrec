## Why
A long unattended HDHomeRun recording should survive temporary reception failures and make missed airtime visible without requiring a DVR service.

## What Changes
- Discover tuners locally, list their channels, and record an exact channel until interrupted.
- Preserve native stream parts, retry transient failures, and inhibit idle sleep where supported.
- Offer live terminal status, machine-readable events, and a persistent session history.

## Capabilities
### New Capabilities
- `foreground-recording`: select a tuner, capture and recover a session, and report its outcome.

### Modified Capabilities
None.

## Impact
New standalone Go CLI for local tuners and local storage. No FFmpeg runtime requirement, background service, scheduling, or changes to existing applications.
