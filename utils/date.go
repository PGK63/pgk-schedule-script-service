package utils

import "time"

func GetCurrentDate() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}
