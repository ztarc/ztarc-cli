package redact

import (
	"encoding/json"
	"fmt"
	newtLogger "github.com/fosrl/newt/logger"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var mu sync.RWMutex
var secrets []string
var output io.Writer = os.Stdout

func Register(values ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, s := range values {
		if s != "" {
			secrets = append(secrets, s)
			b, _ := json.Marshal(s)
			encoded := string(b[1 : len(b)-1])
			if encoded != s {
				secrets = append(secrets, encoded)
			}
		}
		sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	}
}
func String(s string) string {
	mu.RLock()
	defer mu.RUnlock()
	for _, secret := range secrets {
		s = strings.ReplaceAll(s, secret, "[REDACTED]")
	}
	return s
}

type writer struct{}

func (writer) Write(level newtLogger.LogLevel, stamp time.Time, message string) {
	message = String(message)
	mu.RLock()
	defer mu.RUnlock()
	fmt.Fprintf(output, "%s: %s %s\n", level.String(), stamp.UTC().Format(time.RFC3339), message)
}
func InitEngine()              { newtLogger.Init(newtLogger.NewLoggerWithWriter(writer{})) }
func EngineOutput(w io.Writer) { mu.Lock(); output = w; mu.Unlock() }
