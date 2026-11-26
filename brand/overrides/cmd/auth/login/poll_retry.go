package login

import (
	"errors"
	"fmt"
	"github.com/fosrl/cli/internal/api"
	"time"
)

// Inject the clock and wait so retry/expiry behavior can be tested without sleeps.
func awaitDeviceApproval(deadline time.Time, poll func() (*api.DeviceWebAuthPollResponse, string, error), now func() time.Time, wait func(time.Duration)) (string, error) {
	retry := 2 * time.Second
	for {
		if !now().Before(deadline) {
			return "", fmt.Errorf("device authorization expired; please log in again")
		}
		response, message, err := poll()
		delay := time.Second
		if err != nil {
			var limited *api.DevicePollRateLimitError
			if !errors.As(err, &limited) {
				return "", fmt.Errorf("failed to poll device web auth: %w", err)
			}
			delay = retry
			if limited.Delay > delay {
				delay = limited.Delay
			}
			if retry < 30*time.Second {
				retry *= 2
				if retry > 30*time.Second {
					retry = 30 * time.Second
				}
			}
		} else {
			retry = 2 * time.Second
			// Never accept a result returned after the authorization deadline.
			if !now().Before(deadline) {
				return "", fmt.Errorf("device authorization expired; please log in again")
			}
			if response != nil && response.Verified {
				if response.Token == "" {
					return "", fmt.Errorf("verification succeeded but no token received")
				}
				return response.Token, nil
			}
			if message == "Code expired" || message == "Code not found" {
				return "", fmt.Errorf("code expired or not found; please log in again")
			}
		}
		remaining := deadline.Sub(now())
		if remaining <= 0 {
			continue
		}
		if delay > remaining {
			delay = remaining
		}
		wait(delay)
	}
}
