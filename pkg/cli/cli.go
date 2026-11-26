package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bmayfi3ld/hdrec/pkg/hdhomerun"
	"io"
	"os"
)

func selectDevice(ctx context.Context, override string) (hdhomerun.Device, error) {
	return hdhomerun.Select(ctx, override, os.Getenv("HDREC_DEVICE"), hdhomerun.Discover)
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: hdrec devices | channels | record CHANNEL"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	if len(args) > 0 && args[0] == "record" {
		return capture(ctx, args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "channels" {
		fs := flag.NewFlagSet("channels", flag.ContinueOnError)
		fs.SetOutput(stderr)
		j := fs.Bool("json", false, "print JSON")
		override := fs.String("device", "", "tuner URL")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
			return 1
		}
		d, e := selectDevice(ctx, *override)
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		c, e := hdhomerun.New(d.URL)
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		cs, e := c.Channels(ctx)
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		if *j {
			if e = json.NewEncoder(stdout).Encode(cs); e != nil {
				return 1
			}
		} else {
			fmt.Fprintln(stdout, "NUMBER\tNAME")
			for _, s := range cs {
				fmt.Fprintf(stdout, "%s\t%s\n", s.Number, s.Name)
			}
		}
		return 0
	}
	if len(args) > 0 && args[0] == "devices" {
		fs := flag.NewFlagSet("devices", flag.ContinueOnError)
		fs.SetOutput(stderr)
		j := fs.Bool("json", false, "print JSON")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
			return 1
		}
		ds, e := hdhomerun.Discover(ctx)
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		if *j {
			if e = json.NewEncoder(stdout).Encode(ds); e != nil {
				return 1
			}
		} else {
			fmt.Fprintln(stdout, "DEVICE ID\tURL")
			for _, d := range ds {
				fmt.Fprintf(stdout, "%s\t%s\n", d.ID, d.URL)
			}
			if len(ds) == 0 {
				fmt.Fprintln(stderr, "No devices found; use channels --device http://HOST.")
			}
		}
		return 0
	}
	fmt.Fprintln(stderr, usage)
	return 1
}
