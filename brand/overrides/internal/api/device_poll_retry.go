package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DevicePollRateLimitError is retryable only within the device code lifetime.
type DevicePollRateLimitError struct{ Delay time.Duration }

func (*DevicePollRateLimitError) Error() string { return "device authorization polling rate limited" }
func devicePollRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		// A larger value already exceeds the maximum five-minute login budget.
		if seconds > 300 {
			return 5 * time.Minute
		}
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		return 0
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		return date.Sub(now)
	}
	return 0
}
