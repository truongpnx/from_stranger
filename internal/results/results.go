package results

import "time"

func Remaining(expiresAt time.Time) time.Duration {
	return time.Until(expiresAt)
}
