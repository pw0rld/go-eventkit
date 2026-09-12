package reminders

import (
	"fmt"
	"github.com/pw0rld/go-eventkit/internal/validate"
	"time"
)

func validateQuery(o *listOptions) error {
	if o.listIDSet && o.listNameSet {
		return fmt.Errorf("conflicting list filters")
	}
	if o.listIDSet {
		if err := validate.ID(o.listID); err != nil {
			return err
		}
	}
	if o.listNameSet {
		if err := validate.ID(o.listName); err != nil {
			return err
		}
	}
	if err := validate.Text(o.search); err != nil {
		return err
	}
	for _, t := range []*time.Time{o.dueBefore, o.dueAfter} {
		if t != nil {
			if err := validate.Time(*t); err != nil {
				return err
			}
		}
	}
	if o.dueBefore != nil && o.dueAfter != nil && o.dueBefore.Before(*o.dueAfter) {
		return fmt.Errorf("invalid due-date range")
	}
	return nil
}

func validateAlarms(alarms []Alarm) error {
	for _, a := range alarms {
		if a.AbsoluteDate != nil {
			if err := validate.Time(*a.AbsoluteDate); err != nil {
				return err
			}
		}
		if a.AbsoluteDate != nil && (a.RelativeOffset != 0 || a.Location != nil) {
			return fmt.Errorf("alarm must have one trigger kind")
		}
		if a.Location != nil {
			if a.RelativeOffset != 0 {
				return fmt.Errorf("location and relative alarms cannot be combined")
			}
			if err := validate.Text(a.Location.Title); err != nil {
				return err
			}
			if err := validate.Coordinates(a.Location.Latitude, a.Location.Longitude, a.Location.Radius); err != nil {
				return err
			}
			if a.Proximity != ProximityEnter && a.Proximity != ProximityLeave {
				return fmt.Errorf("location alarm requires enter or leave proximity")
			}
		} else if a.Proximity != ProximityNone {
			return fmt.Errorf("proximity requires a location")
		}
	}
	return nil
}

func validateCreate(in CreateReminderInput) error {
	if err := validate.ID(in.Title); err != nil {
		return fmt.Errorf("title: %w", err)
	}
	if err := validate.Selection(in.ListName, in.ListID); err != nil {
		return err
	}
	for _, s := range []string{in.Notes, in.URL} {
		if err := validate.Text(s); err != nil {
			return err
		}
	}
	if in.Priority < 0 || in.Priority > 9 {
		return fmt.Errorf("priority must be 0–9")
	}
	for _, t := range []*time.Time{in.DueDate, in.RemindMeDate} {
		if t != nil {
			if err := validate.Time(*t); err != nil {
				return err
			}
		}
	}
	for _, r := range in.RecurrenceRules {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	return validateAlarms(in.Alarms)
}

func validateUpdate(in UpdateReminderInput) error {
	for _, s := range []*string{in.Title, in.Notes, in.URL, in.ListName, in.ListID} {
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
	if in.ListName != nil && in.ListID != nil {
		return fmt.Errorf("specify list name or ID, not both")
	}
	for _, s := range []*string{in.ListName, in.ListID} {
		if s != nil {
			if err := validate.ID(*s); err != nil {
				return err
			}
		}
	}
	if in.Priority != nil && (*in.Priority < 0 || *in.Priority > 9) {
		return fmt.Errorf("priority must be 0–9")
	}
	for _, t := range []*time.Time{in.DueDate, in.RemindMeDate} {
		if t != nil {
			if err := validate.Time(*t); err != nil {
				return err
			}
		}
	}
	if in.RemindMeDate != nil && in.Alarms != nil {
		return fmt.Errorf("specify remind-me date or replacement alarms, not both")
	}
	if in.RecurrenceRules != nil {
		for _, r := range *in.RecurrenceRules {
			if err := r.Validate(); err != nil {
				return err
			}
		}
	}
	if in.Alarms != nil {
		return validateAlarms(*in.Alarms)
	}
	return nil
}
