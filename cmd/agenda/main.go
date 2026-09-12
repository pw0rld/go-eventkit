package main

import (
	"encoding/json"
	"fmt"
	"github.com/pw0rld/macos-agenda/calendar"
	"github.com/pw0rld/macos-agenda/internal/cli"
	"github.com/pw0rld/macos-agenda/reminders"
	"os"
)

var version = "dev"

func main() { os.Exit(platformRun(run)) }

func run() int {
	args := os.Args[1:]
	var err error
	switch {
	case len(args) == 1 && (args[0] == "version" || args[0] == "--version"):
		err = json.NewEncoder(os.Stdout).Encode(map[string]string{"name": "macos-agenda", "version": version})
	case len(args) == 1 && args[0] == "status":
		err = json.NewEncoder(os.Stdout).Encode(permissionStatus())
	case len(args) > 0 && args[0] == "authorize":
		if len(args) != 2 || (args[1] != "calendar" && args[1] != "reminders") {
			err = &cli.UsageError{Err: fmt.Errorf("usage: agenda authorize calendar|reminders")}
		} else {
			if args[1] == "calendar" {
				_, err = calendar.New()
			} else {
				_, err = reminders.New()
			}
			if err == nil {
				err = json.NewEncoder(os.Stdout).Encode(permissionStatus())
			}
		}
	default:
		err = cli.Run(args, os.Stdin, os.Stdout, cli.Execute)
	}
	if err != nil {
		json.NewEncoder(os.Stderr).Encode(map[string]any{"status": "error", "error": err.Error(), "exitCode": cli.ExitCode(err)})
	}
	return cli.ExitCode(err)
}
