package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func logEvent(level, event string, fields map[string]string) {
	entry := map[string]any{
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"component": "kaite",
		"event":     event,
	}
	for key, value := range fields {
		entry[key] = value
	}
	data, _ := json.Marshal(entry)
	_, _ = fmt.Fprintln(os.Stderr, string(data))
}
