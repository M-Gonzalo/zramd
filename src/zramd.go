package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"zramd/src/util"
	"zramd/src/zram"

	"github.com/alexflint/go-arg"
)

type startCmd struct {
	Algorithm      string  `arg:"-a,env" default:"zstd" placeholder:"A" help:"zram compression algorithm"`
	MaxSizeMB      int     `arg:"-m,--max-size,env" default:"8192" placeholder:"M" help:"maximum total MB of swap to allocate"`
	MaxSizePercent float32 `arg:"-r,--max-ram,env" default:"0.5" placeholder:"P" help:"maximum percentage of RAM allowed to use"`
	SwapPriority   int     `arg:"-p,--priority,env" default:"10" placeholder:"N" help:"swap priority"`
}

type stopCmd struct {
}

type args struct {
	Start *startCmd `arg:"subcommand:start" help:"load zram module and setup swap devices"`
	Stop  *stopCmd  `arg:"subcommand:stop" help:"stop swap devices and unload zram module"`
}

func initializeZram(cmd *startCmd) error {
	if zram.IsLoaded() {
		return errors.New("The zram module is already loaded")
	}
	numCPU := runtime.NumCPU()
	if err := zram.LoadModule(numCPU); err != nil {
		return err
	}
	return nil
}

func deinitializeZram() error {
	return nil
}

func run() int {
	var args args
	parser := arg.MustParse(&args)

	switch {
	case args.Start != nil:
		if args.Start.Algorithm == "zstd" && !util.IsZstdSupported() {
			parser.Fail("The zstd algorithm is not supported on kernels < 4.19")
		}
		if args.Start.MaxSizePercent < 0.05 || args.Start.MaxSizePercent > 1 {
			parser.Fail("--max-ram must be a value between 0.05 and 1")
		}
		if !util.IsRoot() {
			fmt.Fprintf(os.Stderr, "Root privileges are required\n")
			return 1
		}
		err := initializeZram(args.Start)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return 1
		}
		return 0

	case args.Stop != nil:
		if !util.IsRoot() {
			fmt.Fprintf(os.Stderr, "Root privileges are required\n")
			return 1
		}
		err := deinitializeZram()
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return 1
		}
		return 0
	}

	return 0
}

func main() {
	code := run()
	os.Exit(code)
}
