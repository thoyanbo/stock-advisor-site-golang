package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var levelOrder = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
var currentLevel = 1

func SetLevel(level string) {
	if v, ok := levelOrder[level]; ok {
		currentLevel = v
	}
}

func emit(level string, event string, meta map[string]interface{}) {
	if levelOrder[level] < currentLevel {
		return
	}
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"event":     event,
	}
	for k, v := range meta {
		entry[k] = v
	}
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"level":"error","event":"log_marshal_failed","error":"%s"}`+"\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(data))
}

func Debug(event string, meta map[string]interface{}) { emit("debug", event, meta) }
func Info(event string, meta map[string]interface{})  { emit("info", event, meta) }
func Warn(event string, meta map[string]interface{})  { emit("warn", event, meta) }
func Error(event string, meta map[string]interface{}) { emit("error", event, meta) }
