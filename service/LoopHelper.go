package service

import "time"

func intervalSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return time.Second
	}
	return time.Duration(seconds) * time.Second
}
