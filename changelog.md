# Changelog

## Unreleased

- Discover local HDHomeRun devices and list channels in human-readable or JSON output.
- Record native MPEG-TS until interrupted, with tuner-busy retries, automatic reconnects, and idle-sleep inhibition where available.
- Keep one file per uninterrupted capture, with live status or NDJSON and a persistent session log.

Verified on hardware: five-hour recording, controlled stream interruption and reconnect, and playback of a partial left after a forced process kill. Full-day and power-loss results are not yet recorded.
