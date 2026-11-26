package login

import (
	"errors"
	"github.com/fosrl/cli/internal/api"
	"testing"
	"time"
)

func TestApprovalRetriesThrottlingThenExchanges(t *testing.T) {
	clock := time.Unix(1000, 0)
	calls := 0
	var waits []time.Duration
	token, err := awaitDeviceApproval(clock.Add(time.Minute), func() (*api.DeviceWebAuthPollResponse, string, error) {
		calls++
		switch calls {
		case 1:
			return nil, "", &api.DevicePollRateLimitError{Delay: 7 * time.Second}
		case 2:
			return nil, "", &api.DevicePollRateLimitError{}
		case 3:
			return &api.DeviceWebAuthPollResponse{}, "", nil
		default:
			return &api.DeviceWebAuthPollResponse{Verified: true, Token: "test-token"}, "", nil
		}
	}, func() time.Time { return clock }, func(d time.Duration) { waits = append(waits, d); clock = clock.Add(d) })
	if err != nil || token != "test-token" || calls != 4 {
		t.Fatalf("exchange failed: calls=%d err=%v", calls, err)
	}
	if len(waits) != 3 || waits[0] != 7*time.Second || waits[1] != 4*time.Second || waits[2] != time.Second {
		t.Fatalf("waits=%v", waits)
	}
}
func TestApprovalStopsAtExpiryDespiteRepeated429(t *testing.T) {
	clock := time.Unix(1000, 0)
	calls := 0
	_, err := awaitDeviceApproval(clock.Add(5*time.Second), func() (*api.DeviceWebAuthPollResponse, string, error) {
		calls++
		return nil, "", &api.DevicePollRateLimitError{Delay: time.Hour}
	}, func() time.Time { return clock }, func(d time.Duration) { clock = clock.Add(d) })
	if err == nil || calls != 1 || !clock.Equal(time.Unix(1005, 0)) {
		t.Fatalf("expiry not enforced: calls=%d err=%v", calls, err)
	}
}
func TestApprovalDoesNotRetryPermanentFailure(t *testing.T) {
	failure := errors.New("unauthorized")
	clock := time.Now()
	_, err := awaitDeviceApproval(clock.Add(time.Minute), func() (*api.DeviceWebAuthPollResponse, string, error) { return nil, "", failure }, func() time.Time { return clock }, func(time.Duration) { t.Fatal("unexpected retry") })
	if !errors.Is(err, failure) {
		t.Fatal("permanent failure lost")
	}
}
func TestApprovalRejectsLateOrEmptyToken(t *testing.T) {
	for _, late := range []bool{false, true} {
		clock := time.Unix(1000, 0)
		deadline := clock.Add(time.Second)
		_, err := awaitDeviceApproval(deadline, func() (*api.DeviceWebAuthPollResponse, string, error) {
			if late {
				clock = deadline
			}
			return &api.DeviceWebAuthPollResponse{Verified: true}, "", nil
		}, func() time.Time { return clock }, func(time.Duration) { t.Fatal("unexpected wait") })
		if err == nil {
			t.Fatal("accepted expired or empty token")
		}
	}
}
