package calendar

import (
	"fmt"
	"github.com/pw0rld/macos-agenda"
	"github.com/pw0rld/macos-agenda/internal/validate"
	"time"
)

// Validate checks a create request without requesting macOS access.
func (in CreateEventInput) Validate() error { return validateCreate(in) }

// Validate checks an update payload; effective dates are also checked against the stored event on save.
func (in UpdateEventInput) Validate() error { return validateUpdate(in) }

func validateSpan(s Span) error {
	if s != SpanThisEvent && s != SpanFutureEvents {
		return fmt.Errorf("invalid recurrence span")
	}
	return nil
}

func validateQuery(start, end time.Time, o listOptions) error {
	if err := validate.Range(start, end); err != nil {
		return err
	}
	if o.calendarIDSet {
		if len(o.calendarNames) > 0 {
			return fmt.Errorf("conflicting calendar filters")
		}
		if err := validate.ID(o.calendarID); err != nil {
			return err
		}
	}
	for _, name := range o.calendarNames {
		if err := validate.ID(name); err != nil {
			return err
		}
	}
	return validate.Text(o.searchQuery)
}

func validateLocation(loc *eventkit.StructuredLocation) error {
	if loc == nil {
		return nil
	}
	if err := validate.Text(loc.Title); err != nil {
		return err
	}
	return validate.Coordinates(loc.Latitude, loc.Longitude, loc.Radius)
}

func validateCreate(in CreateEventInput) error {
	if err := validate.ID(in.Title); err != nil {
		return fmt.Errorf("title: %w", err)
	}
	if err := validate.Range(in.StartDate, in.EndDate); err != nil {
		return err
	}
	if err := validate.Selection(in.Calendar, in.CalendarID); err != nil {
		return err
	}
	for _, s := range []string{in.Location, in.Notes, in.URL} {
		if err := validate.Text(s); err != nil {
			return err
		}
	}
	if in.TimeZone != "" {
		if _, err := time.LoadLocation(in.TimeZone); err != nil {
			return err
		}
	}
	for _, r := range in.RecurrenceRules {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	return validateLocation(in.StructuredLocation)
}

func validateUpdate(in UpdateEventInput) error {
	for _, s := range []*string{in.Title, in.Location, in.Notes, in.URL, in.TimeZone, in.Calendar, in.CalendarID} {
		if s != nil {
			if err := validate.Text(*s); err != nil {
				return err
			}
		}
	}
	if in.Title != nil {
		if err := validate.ID(*in.Title); err != nil {
			return err
		}
	}
	if in.Calendar != nil && in.CalendarID != nil {
		return fmt.Errorf("specify calendar name or ID, not both")
	}
	for _, s := range []*string{in.Calendar, in.CalendarID} {
		if s != nil {
			if err := validate.ID(*s); err != nil {
				return err
			}
		}
	}
	for _, t := range []*time.Time{in.StartDate, in.EndDate} {
		if t != nil {
			if err := validate.Time(*t); err != nil {
				return err
			}
		}
	}
	if in.StartDate != nil && in.EndDate != nil {
		if err := validate.Range(*in.StartDate, *in.EndDate); err != nil {
			return err
		}
	}
	if in.TimeZone != nil && *in.TimeZone != "" {
		if _, err := time.LoadLocation(*in.TimeZone); err != nil {
			return err
		}
	}
	if in.RecurrenceRules != nil {
		for _, r := range *in.RecurrenceRules {
			if err := r.Validate(); err != nil {
				return err
			}
		}
	}
	return validateLocation(in.StructuredLocation)
}
