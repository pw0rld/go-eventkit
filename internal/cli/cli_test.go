package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pw0rld/macos-agenda/calendar"
	"github.com/pw0rld/macos-agenda/reminders"
	"strings"
	"testing"
)

func TestDryRunNeverOpensStore(t *testing.T) {
	body := `{"title":"准备 PPT","listID":"exact-list","dueDate":"2026-09-15T18:00:00+08:00"}`
	var out bytes.Buffer
	err := Run([]string{"reminders", "create", "--input", "-", "--dry-run"}, strings.NewReader(body), &out, func(Request) (any, error) { t.Fatal("store opened during dry run"); return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "validated" {
		t.Fatal(out.String())
	}
}

func TestInvalidRequestsNeverReachStore(t *testing.T) {
	tests := []struct {
		args []string
		body string
	}{
		{[]string{"reminders", "delete", "--id", "x"}, ""},
		{[]string{"reminders", "delete", "--id", "x", "--confirm-id", "y"}, ""},
		{[]string{"reminders", "get", "--id", ""}, ""},
		{[]string{"reminders", "list", "--list-id", ""}, ""},
		{[]string{"events", "list", "--from", "2026-09-15", "--to", "2026-09-16"}, ""},
		{[]string{"events", "get", "--id", "x", "--list-id", "y"}, ""},
		{[]string{"reminders", "create", "--input", "-"}, `{"title":"task"}`},
		{[]string{"reminders", "create", "--input", "-"}, `{"title":"task","listID":"x","listID":"y"}`},
		{[]string{"reminders", "create", "--input", "-"}, `{"title":"task","listID":"x","ListID":"y"}`},
		{[]string{"reminders", "create", "--input", "-"}, `{"title":"task","listID":"x","flagged":true}`},
		{[]string{"reminders", "update", "--id", "x", "--input", "-"}, `{}`},
		{[]string{"reminders", "update", "--id", "x", "--input", "-"}, `{"title":null}`},
		{[]string{"reminders", "create", "--input", "-"}, `null`},
		{[]string{"reminders", "create", "--input", "-"}, `{} {}`},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var out bytes.Buffer
			err := Run(tt.args, strings.NewReader(tt.body), &out, func(Request) (any, error) { t.Fatal("invalid request reached store"); return nil, nil })
			if ExitCode(err) != 64 {
				t.Errorf("expected usage error, got %v", err)
			}
		})
	}
}

func TestEventPayloadAndExecution(t *testing.T) {
	body := `{"title":"Review","calendarID":"work-id","startDate":"2026-09-15T09:00:00+08:00","endDate":"2026-09-15T10:00:00+08:00"}`
	var out bytes.Buffer
	calls := 0
	err := Run([]string{"events", "create", "--input", "-"}, strings.NewReader(body), &out, func(r Request) (any, error) {
		calls++
		p := r.Payload.(*calendar.CreateEventInput)
		if p.CalendarID != "work-id" || p.StartDate.UTC().Hour() != 1 {
			t.Fatalf("bad conversion: %+v", p)
		}
		return map[string]string{"id": "created-id"}, nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("%v calls=%d", err, calls)
	}
	if !strings.Contains(out.String(), "created-id") {
		t.Fatal(out.String())
	}
}

func TestUpdatePreservesOmittedFields(t *testing.T) {
	r, err := Parse([]string{"reminders", "update", "--id", "x", "--input", "-"}, strings.NewReader(`{"notes":""}`))
	if err != nil {
		t.Fatal(err)
	}
	p := r.Payload.(*reminders.UpdateReminderInput)
	if p.Notes == nil || *p.Notes != "" || p.Title != nil || p.ListID != nil || p.DueDate != nil {
		t.Fatalf("unexpected update: %+v", p)
	}
}

func TestLimitsAndHelp(t *testing.T) {
	for _, body := range []string{strings.Repeat("x", 1048577), `{"notes":` + strings.Repeat("[", 130) + `0` + strings.Repeat("]", 130) + `}`} {
		if _, err := Parse([]string{"reminders", "create", "--input", "-"}, strings.NewReader(body)); err == nil {
			t.Fatal("oversized/deep input accepted")
		}
	}
	var out bytes.Buffer
	if err := Run([]string{"--help"}, strings.NewReader(""), &out, func(Request) (any, error) { t.Fatal("help opened store"); return nil, nil }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "macos-agenda") {
		t.Fatal(out.String())
	}
}
