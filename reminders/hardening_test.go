package reminders

import (
	"encoding/json"
	"testing"
	"time"
)

func TestExactListSelection(t *testing.T) {
	in := CreateReminderInput{Title: "Prepare slides", ListID: "list-exact"}
	data, err := marshalCreateInput(in)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		t.Fatal(err)
	}
	if out["listID"] != "list-exact" {
		t.Fatal(data)
	}
	in.ListName = "Work"
	if _, err := marshalCreateInput(in); err == nil {
		t.Fatal("conflicting scope accepted")
	}
	id := "list-exact"
	data, err = marshalUpdateInput(UpdateReminderInput{ListID: &id})
	if err != nil {
		t.Fatal(err)
	}
	out = map[string]any{}
	json.Unmarshal([]byte(data), &out)
	if out["listID"] != id {
		t.Fatal(data)
	}
	for _, opts := range [][]ListOption{{WithListID("")}, {WithList(" ")}, {WithListID("id"), WithList("Work")}} {
		if err := validateQuery(applyOptions(opts)); err == nil {
			t.Fatal("invalid filter accepted")
		}
	}
	if err := validateQuery(applyOptions([]ListOption{WithListID("id")})); err != nil {
		t.Fatal(err)
	}
}

func TestReminderValidation(t *testing.T) {
	for _, in := range []CreateReminderInput{{Title: ""}, {Title: "bad\x00title"}, {Title: "x", Priority: 10}, {Title: "x", ListID: "\n"}} {
		if _, err := marshalCreateInput(in); err == nil {
			t.Errorf("invalid input accepted: %+v", in)
		}
	}
	now := time.Now()
	alarms := []Alarm{}
	if _, err := marshalUpdateInput(UpdateReminderInput{RemindMeDate: &now, Alarms: &alarms}); err == nil {
		t.Fatal("conflicting alarm updates accepted")
	}
	data, err := marshalCreateListInput(CreateListInput{Title: "Work", SourceID: "source-exact"})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	json.Unmarshal([]byte(data), &out)
	if out["sourceID"] != "source-exact" {
		t.Fatal(data)
	}
}
