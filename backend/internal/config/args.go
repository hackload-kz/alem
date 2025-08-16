package config

import (
	"flag"
	"fmt"
	"os"
)

type Args struct {
	MigrationsDir string
}

func ParseArgs() *Args {
	args := &Args{}

	flag.StringVar(&args.MigrationsDir, "m", "", "Path to database migrations directory")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	// Parse the flags
	flag.Parse()

	return args
}
