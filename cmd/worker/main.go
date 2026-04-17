// Command worker is a placeholder entry point for background jobs (timer
// reminders, scheduled aggregations) that later changes will introduce.
// Kept as a buildable no-op so `go build ./...` and Makefile targets work
// from day one.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "timetrak worker: no jobs defined in the MVP; exiting")
}
