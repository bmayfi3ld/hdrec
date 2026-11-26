package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/bmayfi3ld/hdrec/pkg/hdhomerun"
	"github.com/bmayfi3ld/hdrec/pkg/power"
	"github.com/bmayfi3ld/hdrec/pkg/record"
)

func holdPower(ctx context.Context, inhibit func(context.Context) (io.Closer, error), warn func(string)) func() {
	h, e := inhibit(ctx)
	if e != nil {
		warn("Could not prevent idle sleep; disable idle sleep manually: " + e.Error())
		return func() {}
	}
	return func() { _ = h.Close() }
}

func capture(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	jsonRequested := false
	for _, arg := range args {
		if arg == "--json" || arg == "-json" || arg == "--json=true" || arg == "-json=true" {
			jsonRequested = true
		}
	}
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonMode := fs.Bool("json", false, "print newline-delimited JSON")
	device := fs.String("device", "", "tuner URL")
	dir := fs.String("out-dir", "recordings", "output directory")
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		args = append(append([]string{}, args[1:]...), args[0])
	}
	parseErr := fs.Parse(args)
	const usage = "Usage: hdrec record CHANNEL [--device URL] [--out-dir DIR] [--json]"
	if parseErr == flag.ErrHelp && !jsonRequested {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	report, finish := liveOutput(stdout, *jsonMode || jsonRequested, fs.Arg(0))
	defer finish()
	fail := func(code string, e error) int {
		report(record.Event{Kind: "error", Code: code, Message: e.Error()})
		return 1
	}
	if parseErr != nil {
		return fail("invalid_args", fmt.Errorf("%v. %s", parseErr, usage))
	}
	if fs.NArg() != 1 {
		return fail("invalid_args", fmt.Errorf("%s", usage))
	}
	d, e := selectDevice(ctx, *device)
	if e != nil {
		return fail("device_selection", e)
	}
	c, e := hdhomerun.New(d.URL)
	if e != nil {
		return fail("device_selection", e)
	}
	defer holdPower(ctx, power.Inhibit, func(msg string) {
		report(record.Event{Kind: "warning", Code: "sleep_inhibitor_unavailable", Message: msg})
	})()
	started := false
	r, e := record.Run(ctx, c, record.Request{Channel: fs.Arg(0), OutputDir: *dir, Device: d.URL}, func(event record.Event) {
		if event.Kind == "session_started" {
			started = true
		}
		report(event)
	})
	if e != nil && !started {
		return fail("storage_error", e)
	}
	return r.ExitCode(e)
}
