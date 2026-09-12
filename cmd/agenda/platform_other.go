//go:build !darwin

package main

func platformRun(fn func() int) int { return fn() }
func permissionStatus() map[string]string {
	return map[string]string{"calendar": "unsupported", "reminders": "unsupported"}
}
