// Package cli parses requests before opening either EventKit store.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pw0rld/macos-agenda/calendar"
	"github.com/pw0rld/macos-agenda/internal/validate"
	"github.com/pw0rld/macos-agenda/reminders"
)

const Help = `macos-agenda — native macOS Calendar and Reminders CLI

Usage:
  agenda calendars
  agenda lists
  agenda events list --from RFC3339 --to RFC3339 [--calendar-id ID] [--search TEXT]
  agenda reminders list [--list-id ID] [--completed all|true|false] [--search TEXT]
  agenda events get --id ID
  agenda reminders get --id ID
  agenda events create --input FILE|- [--dry-run]
  agenda reminders create --input FILE|- [--dry-run]
  agenda events update --id ID --input FILE|- [--span this|future] [--dry-run]
  agenda reminders update --id ID --input FILE|- [--dry-run]
  agenda reminders complete --id ID [--dry-run]
  agenda reminders uncomplete --id ID [--dry-run]
  agenda events delete --id ID --confirm-id ID [--span this|future] [--dry-run]
  agenda reminders delete --id ID --confirm-id ID [--dry-run]
  agenda status
  agenda authorize calendar|reminders
  agenda version

Input is one JSON object (maximum 1 MiB); times must be RFC3339 with an offset.
Create requires an explicit calendarID/listID. Updates preserve omitted fields.
--dry-run validates request syntax only, without reading the store or checking IDs exist.
Success is JSON on stdout; errors are JSON on stderr. No GUI clicking is used.
`

type Request struct {
	Resource   string        `json:"resource"`
	Action     string        `json:"action"`
	ID         string        `json:"id,omitempty"`
	CalendarID string        `json:"calendarID,omitempty"`
	ListID     string        `json:"listID,omitempty"`
	Search     string        `json:"search,omitempty"`
	Completed  string        `json:"completed,omitempty"`
	From       time.Time     `json:"from,omitempty"`
	To         time.Time     `json:"to,omitempty"`
	Span       calendar.Span `json:"span"`
	DryRun     bool          `json:"dryRun"`
	Payload    any           `json:"payload,omitempty"`
}

type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

func Parse(args []string, stdin io.Reader) (req Request, err error) {
	defer func() {
		if err != nil {
			err = &UsageError{err}
		}
	}()
	if len(args) == 0 {
		return req, fmt.Errorf("missing command; use agenda --help")
	}
	req.Resource = args[0]
	if req.Resource == "calendars" || req.Resource == "lists" {
		if len(args) != 1 {
			return req, fmt.Errorf("this command takes no arguments")
		}
		req.Action = "list"
		return req, nil
	}
	if req.Resource != "events" && req.Resource != "reminders" {
		return req, fmt.Errorf("unknown resource %q", req.Resource)
	}
	if len(args) < 2 {
		return req, fmt.Errorf("missing action")
	}
	req.Action = args[1]
	allowed := map[string]bool{}
	allow := func(names ...string) {
		for _, n := range names {
			allowed[n] = true
		}
	}
	switch req.Action {
	case "list":
		allow("search")
		if req.Resource == "events" {
			allow("from", "to", "calendar-id")
		} else {
			allow("list-id", "completed")
		}
	case "get":
		allow("id")
	case "create":
		allow("input", "dry-run")
	case "update":
		allow("id", "input", "dry-run")
		if req.Resource == "events" {
			allow("span")
		}
	case "delete":
		allow("id", "confirm-id", "dry-run")
		if req.Resource == "events" {
			allow("span")
		}
	case "complete", "uncomplete":
		if req.Resource != "reminders" {
			return req, fmt.Errorf("complete/uncomplete apply only to reminders")
		}
		allow("id", "dry-run")
	default:
		return req, fmt.Errorf("unknown action %q", req.Action)
	}
	fs := flag.NewFlagSet("agenda", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var from, to, input, span, confirm string
	fs.StringVar(&req.ID, "id", "", "complete identifier")
	fs.StringVar(&req.CalendarID, "calendar-id", "", "exact calendar identifier")
	fs.StringVar(&req.ListID, "list-id", "", "exact list identifier")
	fs.StringVar(&req.Search, "search", "", "search text")
	fs.StringVar(&req.Completed, "completed", "all", "completion filter")
	fs.StringVar(&from, "from", "", "range start")
	fs.StringVar(&to, "to", "", "range end")
	fs.StringVar(&input, "input", "", "JSON file or - for stdin")
	fs.StringVar(&span, "span", "this", "recurrence scope")
	fs.StringVar(&confirm, "confirm-id", "", "exact ID to confirm deletion")
	fs.BoolVar(&req.DryRun, "dry-run", false, "validate request only")
	if err = fs.Parse(args[2:]); err != nil {
		return req, err
	}
	if fs.NArg() != 0 {
		return req, fmt.Errorf("unexpected positional arguments")
	}
	fs.Visit(func(f *flag.Flag) {
		if !allowed[f.Name] {
			err = fmt.Errorf("--%s does not apply to %s %s", f.Name, req.Resource, req.Action)
		}
		if (f.Name == "id" || f.Name == "calendar-id" || f.Name == "list-id") && f.Value.String() == "" {
			err = fmt.Errorf("--%s cannot be empty", f.Name)
		}
	})
	if err != nil {
		return req, err
	}
	for _, s := range []string{req.ID, req.CalendarID, req.ListID} {
		if s != "" {
			if err = validate.ID(s); err != nil {
				return req, err
			}
		}
	}
	if err = validate.Text(req.Search); err != nil {
		return req, err
	}
	if req.Action != "list" && req.Action != "create" {
		if err = validate.ID(req.ID); err != nil {
			return req, err
		}
	}
	if span != "this" && span != "future" {
		return req, fmt.Errorf("span must be this or future")
	}
	if span == "future" {
		req.Span = calendar.SpanFutureEvents
	}
	if req.Action == "delete" && !req.DryRun && confirm != req.ID {
		return req, fmt.Errorf("delete requires --confirm-id matching --id")
	}
	if req.Action == "list" && req.Resource == "events" {
		req.From, err = time.Parse(time.RFC3339, from)
		if err != nil {
			return req, fmt.Errorf("--from must be RFC3339 with timezone")
		}
		req.To, err = time.Parse(time.RFC3339, to)
		if err != nil {
			return req, fmt.Errorf("--to must be RFC3339 with timezone")
		}
		if err = validate.Range(req.From, req.To); err != nil {
			return req, err
		}
	}
	if req.Completed != "all" && req.Completed != "true" && req.Completed != "false" {
		return req, fmt.Errorf("completed must be all, true or false")
	}
	if req.Action != "create" && req.Action != "update" {
		return req, nil
	}
	if input == "" {
		return req, fmt.Errorf("--input FILE or --input - is required")
	}
	r := stdin
	if input != "-" {
		f, e := os.Open(input)
		if e != nil {
			return req, e
		}
		defer f.Close()
		r = f
	}
	data, e := io.ReadAll(io.LimitReader(r, 1048577))
	if e != nil {
		return req, e
	}
	if len(data) > 1048576 {
		return req, fmt.Errorf("input exceeds 1 MiB")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return req, fmt.Errorf("input must be a JSON object")
	}
	if err = uniqueKeys(json.NewDecoder(bytes.NewReader(data)), 0); err != nil {
		return req, err
	}
	switch req.Resource + "/" + req.Action {
	case "events/create":
		req.Payload = &calendar.CreateEventInput{}
	case "events/update":
		req.Payload = &calendar.UpdateEventInput{}
	case "reminders/create":
		req.Payload = &reminders.CreateReminderInput{}
	case "reminders/update":
		req.Payload = &reminders.UpdateReminderInput{}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(req.Payload); err != nil {
		return req, err
	}
	var extra any
	if e = dec.Decode(&extra); e != io.EOF {
		return req, fmt.Errorf("input must contain exactly one JSON object")
	}
	if err = req.Payload.(interface{ Validate() error }).Validate(); err != nil {
		return req, err
	}
	switch p := req.Payload.(type) {
	case *calendar.CreateEventInput:
		if err = validate.ID(p.CalendarID); err != nil {
			return req, fmt.Errorf("create requires calendarID")
		}
	case *reminders.CreateReminderInput:
		if err = validate.ID(p.ListID); err != nil {
			return req, fmt.Errorf("create requires listID")
		}
	}
	if req.Action == "update" {
		var fields map[string]json.RawMessage
		json.Unmarshal(data, &fields)
		effective := false
		for _, v := range fields {
			if string(bytes.TrimSpace(v)) != "null" {
				effective = true
			}
		}
		if !effective {
			return req, fmt.Errorf("empty update")
		}
	}
	return req, nil
}

// Duplicate keys (including casing variants) must not silently override an earlier instruction.
func uniqueKeys(d *json.Decoder, depth int) error {
	if depth > 128 {
		return fmt.Errorf("JSON nesting exceeds 128 levels")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return e
			}
			key := strings.ToLower(k.(string))
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = true
			if err := uniqueKeys(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := uniqueKeys(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}

func Run(args []string, stdin io.Reader, stdout io.Writer, execute func(Request) (any, error)) error {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		_, err := io.WriteString(stdout, Help)
		return err
	}
	req, err := Parse(args, stdin)
	if err != nil {
		return err
	}
	if req.DryRun {
		plan := map[string]any{"resource": req.Resource, "action": req.Action, "dryRun": true}
		if req.ID != "" {
			plan["id"] = req.ID
		}
		if req.Payload != nil {
			plan["payload"] = req.Payload
		}
		if req.Resource == "events" && (req.Action == "update" || req.Action == "delete") {
			plan["span"] = req.Span.String()
		}
		return json.NewEncoder(stdout).Encode(map[string]any{"status": "validated", "request": plan})
	}
	result, err := execute(req)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(map[string]any{"status": "success", "data": result})
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var usage *UsageError
	if errors.As(err, &usage) {
		return 64
	}
	if errors.Is(err, calendar.ErrAccessDenied) || errors.Is(err, reminders.ErrAccessDenied) {
		return 2
	}
	return 1
}
