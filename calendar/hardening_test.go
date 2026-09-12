package calendar

import (
	"encoding/json"
	"github.com/pw0rld/macos-agenda"
	"math"
	"testing"
	"time"
)

func TestCreateSelectionAndZeroCoordinates(t *testing.T) {
	start := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	in := CreateEventInput{Title: "Review", StartDate: start, EndDate: start.Add(time.Hour), CalendarID: "calendar-exact", StructuredLocation: &eventkit.StructuredLocation{Title: "Equator", Latitude: 0, Longitude: 12}}
	b, err := marshalCreateInput(in)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["calendarID"] != "calendar-exact" {
		t.Fatal(string(b))
	}
	loc := out["structuredLocation"].(map[string]any)
	if lat, ok := loc["latitude"]; !ok || lat != float64(0) {
		t.Fatal("zero latitude was lost")
	}
	in.Calendar = "Work"
	if _, err := marshalCreateInput(in); err == nil {
		t.Fatal("ambiguous name/ID accepted")
	}
}

func TestRejectInvalidMutation(t *testing.T) {
	start := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	for _, change := range []func(*CreateEventInput){
		func(i *CreateEventInput) { i.Title = "" },
		func(i *CreateEventInput) { i.Notes = "bad\x00text" },
		func(i *CreateEventInput) { i.CalendarID = "\n" },
		func(i *CreateEventInput) { i.EndDate = i.StartDate },
		func(i *CreateEventInput) { i.TimeZone = "Unknown/Zone" },
		func(i *CreateEventInput) { i.StructuredLocation = &eventkit.StructuredLocation{Latitude: math.NaN()} },
	} {
		in := CreateEventInput{Title: "Review", StartDate: start, EndDate: start.Add(time.Hour)}
		change(&in)
		if _, err := marshalCreateInput(in); err == nil {
			t.Errorf("invalid input accepted: %+v", in)
		}
	}
}

func TestRejectInvalidQuerySelection(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Hour)
	for _, opts := range [][]ListOption{{WithCalendarID("")}, {WithCalendar("")}, {WithCalendars(nil)}, {WithCalendar("Work"), WithCalendarID("id")}, {WithCalendar("one\ntwo")}} {
		if err := validateQuery(start, end, applyOptions(opts)); err == nil {
			t.Fatal("invalid scope accepted")
		}
	}
}

func TestContainerSourceID(t *testing.T) {
	b, err := marshalCreateCalendarInput(CreateCalendarInput{Title: "Work", SourceID: "source-exact"})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	json.Unmarshal(b, &out)
	if out["sourceID"] != "source-exact" {
		t.Fatal(string(b))
	}
	if _, err := marshalCreateCalendarInput(CreateCalendarInput{Title: "Work", Source: "iCloud", SourceID: "id"}); err == nil {
		t.Fatal("conflicting source accepted")
	}
}

func TestConferenceFromPublicFields(t *testing.T) {
	url := "https://zoom.us/j/123"
	notes := "Join https://meet.google.com/abc-defg-hij"
	e := convertRawEvent(rawEvent{URL: &url, Notes: &notes})
	if e.ConferenceURL != url {
		t.Fatal(e.ConferenceURL)
	}
	e = convertRawEvent(rawEvent{Notes: &notes})
	if e.ConferenceURL != "https://meet.google.com/abc-defg-hij" {
		t.Fatal(e.ConferenceURL)
	}
}
