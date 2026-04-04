package telemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/codila125/musica/internal/logger"
)

type Field struct {
	Key   string
	Value any
}

var (
	mu       sync.Mutex
	counters = map[string]int64{}
)

func Count(name string) {
	mu.Lock()
	counters[name]++
	mu.Unlock()
}

func Snapshot() map[string]int64 {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]int64, len(counters))
	for k, v := range counters {
		out[k] = v
	}
	return out
}

func Event(op string, fields ...Field) {
	parts := []string{"op=" + op}
	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", f.Key, f.Value))
	}
	logger.Get().Info("%s", strings.Join(parts, " "))
}

func Timed(op string, fields ...Field) func(extra ...Field) {
	start := time.Now()
	return func(extra ...Field) {
		all := append([]Field{}, fields...)
		all = append(all, extra...)
		all = append(all, Field{Key: "duration_ms", Value: time.Since(start).Milliseconds()})
		Event(op, all...)
	}
}

func CountersField() Field {
	s := Snapshot()
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", k, s[k]))
	}
	return Field{Key: "counters", Value: strings.Join(parts, ",")}
}
