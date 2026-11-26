# hdrec
A foreground Go recorder for HDHomeRun tuners. It writes native MPEG-TS without FFmpeg and retries busy tuners and lost connections until you stop it.

## Quick start

From a checkout with Go 1.22 or newer:

```sh
go build ./cmd/hdrec
./hdrec devices
./hdrec channels
./hdrec channels --json
./hdrec record 7.1 --out-dir ./recordings
```

Replace `7.1` with an exact channel number from your lineup. If discovery finds no devices or more than one, choose a tuner with `--device http://TUNER_HOST` or set `HDREC_DEVICE`. An explicit URL is never replaced with a different tuner.

Press **Ctrl-C** to stop and finalize. The tool tries to inhibit idle sleep while running and warns if it can't. Check the machine's sleep and lid settings before a long recording.

## Files and monitoring

Each run creates a new directory under `./recordings` (or `--out-dir`). One uninterrupted stretch produces one `.ts` file, regardless of duration. During capture the file ends in `.ts.partial`; a reconnect starts a new part. A large file needs a filesystem that supports it and enough free disk space. A reconnect cannot recover missed broadcasts.

In a terminal, events scroll above a live status line. For automation use `./hdrec record 7.1 --json` for newline-delimited JSON. If live output is blocked or its consumer goes away, recording continues. From another terminal, follow `tail -f /path/to/session/session.ndjson` for lifecycle events and check whether the active partial is growing. The session log retains the final outcome if live output could not deliver it.

Exit status: `0` = clean stop with captured data, `2` = captured data with known missed airtime, `1` = fatal error or no captured data. A crash may leave a `.ts.partial`; after confirming the recorder has stopped, try opening it directly in mpv. A process-kill test left a playable partial, but power-loss durability has not been tested. The tool does not restart itself.

See [changelog.md](changelog.md) for release notes. Licensed under [MIT](LICENSE).
