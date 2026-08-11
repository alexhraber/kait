package main

import (
	"os"
	"runtime"
)

func defaultBuildkiteAgentBinary() string {
	if runtime.GOOS == "darwin" {
		for _, path := range []string{
			"/opt/homebrew/bin/buildkite-agent",
			"/usr/local/bin/buildkite-agent",
		} {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		return "buildkite-agent"
	}
	return "/buildkite/bin/buildkite-agent"
}
