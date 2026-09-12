package cli

import (
	"fmt"
	"github.com/pw0rld/macos-agenda/calendar"
	"github.com/pw0rld/macos-agenda/reminders"
)

func Execute(req Request) (any, error) {
	if req.Resource == "calendars" || req.Resource == "events" {
		c, err := calendar.New()
		if err != nil {
			return nil, err
		}
		if req.Resource == "calendars" {
			return c.Calendars()
		}
		switch req.Action {
		case "list":
			var opts []calendar.ListOption
			if req.CalendarID != "" {
				opts = append(opts, calendar.WithCalendarID(req.CalendarID))
			}
			if req.Search != "" {
				opts = append(opts, calendar.WithSearch(req.Search))
			}
			return c.Events(req.From, req.To, opts...)
		case "get":
			return c.Event(req.ID)
		case "create":
			return c.CreateEvent(*req.Payload.(*calendar.CreateEventInput))
		case "update":
			return c.UpdateEvent(req.ID, *req.Payload.(*calendar.UpdateEventInput), req.Span)
		case "delete":
			err := c.DeleteEvent(req.ID, req.Span)
			return map[string]string{"deletedID": req.ID}, err
		}
	} else if req.Resource == "lists" || req.Resource == "reminders" {
		c, err := reminders.New()
		if err != nil {
			return nil, err
		}
		if req.Resource == "lists" {
			return c.Lists()
		}
		switch req.Action {
		case "list":
			var opts []reminders.ListOption
			if req.ListID != "" {
				opts = append(opts, reminders.WithListID(req.ListID))
			}
			if req.Search != "" {
				opts = append(opts, reminders.WithSearch(req.Search))
			}
			if req.Completed != "all" {
				opts = append(opts, reminders.WithCompleted(req.Completed == "true"))
			}
			return c.Reminders(opts...)
		case "get":
			return c.Reminder(req.ID)
		case "create":
			return c.CreateReminder(*req.Payload.(*reminders.CreateReminderInput))
		case "update":
			return c.UpdateReminder(req.ID, *req.Payload.(*reminders.UpdateReminderInput))
		case "complete":
			return c.CompleteReminder(req.ID)
		case "uncomplete":
			return c.UncompleteReminder(req.ID)
		case "delete":
			err := c.DeleteReminder(req.ID)
			return map[string]string{"deletedID": req.ID}, err
		}
	}
	return nil, fmt.Errorf("unsupported request")
}
