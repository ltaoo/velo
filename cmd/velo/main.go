package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: velo <command> [options]")
		fmt.Fprintln(os.Stderr, "commands: build")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		platform := fs.String("platform", "", "target platform: darwin, windows, linux, all (default: current OS)")
		outDir := fs.String("out", "dist", "output directory")
		fs.Parse(os.Args[2:])

		projectPath := "."
		if fs.NArg() > 0 {
			projectPath = fs.Arg(0)
		}

		if err := runBuild(projectPath, *platform, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
