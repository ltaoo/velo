package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ltaoo/velo/buildcfg"
)

func resolveProjectConfig(dir string) (string, error) {
	configPath, legacy, err := buildcfg.ResolveConfigPath(dir)
	if err != nil {
		return "", err
	}
	if legacy {
		fmt.Fprintf(
			os.Stderr,
			"warning: %s is deprecated; rename it to %s\n",
			filepath.Base(configPath),
			buildcfg.ConfigFileName,
		)
	}
	return configPath, nil
}
