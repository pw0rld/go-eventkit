//go:build darwin

package calendar

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework EventKit -framework Foundation -framework AppKit -framework CoreLocation
#include "bridge_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"github.com/pw0rld/macos-agenda/internal/validate"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var bridgeMu sync.Mutex

func resultErr(res C.ek_result_t) error {
	if res.error != nil {
		msg := C.GoString(res.error)
		C.ek_cal_free(res.error)
		if strings.Contains(msg, "ambiguous") {
			return fmt.Errorf("%w: %s", ErrSelection, msg)
		}
		if strings.Contains(msg, "not found") {
			return fmt.Errorf("%w: %s", ErrNotFound, msg)
		}
		if strings.Contains(msg, "immutable") {
			return fmt.Errorf("%w: %s", ErrImmutable, msg)
		}
		return errors.New(msg)
	}
	return errors.New("unknown error")
}

func isNotFound(err error) bool {
	return err != nil && errors.Is(err, ErrNotFound)
}

// New creates a new Calendar [Client] and requests calendar access.
//
// On first call, macOS displays a TCC prompt requesting calendar access.
// Returns [ErrAccessDenied] if the user denies access.
// Returns [ErrUnsupported] on non-darwin platforms.
func New() (*Client, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	res := C.ek_cal_request_access()
	if res.error != nil {
		err := resultErr(res)
		if strings.Contains(err.Error(), "denied") {
			return nil, ErrAccessDenied
		}
		return nil, fmt.Errorf("calendar: %s", err)
	}
	C.ek_cal_free(res.result)
	return &Client{}, nil
}

// Calendars returns all calendars for events across all accounts
// (iCloud, Google, Exchange, local, subscribed, birthdays).
func (c *Client) Calendars() ([]Calendar, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	res := C.ek_cal_fetch_calendars()
	if res.error != nil {
		return nil, fmt.Errorf("calendar: %w", resultErr(res))
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	return parseCalendarsJSON(jsonStr)
}

// Events returns events within the given time range.
// EventKit requires a bounded date range — this method cannot fetch all events.
// Options can filter by calendar name, calendar ID, or search query.
func (c *Client) Events(start, end time.Time, opts ...ListOption) ([]Event, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	o := applyOptions(opts)
	if err := validateQuery(start, end, o); err != nil {
		return nil, err
	}
	strictID := C.int(0)
	if o.calendarIDSet {
		strictID = 1
	}

	cStart := C.CString(start.UTC().Format("2006-01-02T15:04:05.000Z"))
	defer C.free(unsafe.Pointer(cStart))
	cEnd := C.CString(end.UTC().Format("2006-01-02T15:04:05.000Z"))
	defer C.free(unsafe.Pointer(cEnd))

	var cCalID *C.char
	if o.calendarID != "" {
		cCalID = C.CString(o.calendarID)
		defer C.free(unsafe.Pointer(cCalID))
	} else if len(o.calendarNames) > 0 {
		cCalID = C.CString(strings.Join(o.calendarNames, "\n"))
		defer C.free(unsafe.Pointer(cCalID))
	}

	var cSearch *C.char
	if o.searchQuery != "" {
		cSearch = C.CString(o.searchQuery)
		defer C.free(unsafe.Pointer(cSearch))
	}

	res := C.ek_cal_fetch_events(cStart, cEnd, cCalID, cSearch, strictID)
	if res.error != nil {
		return nil, fmt.Errorf("calendar: %w", resultErr(res))
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	return parseEventsJSON(jsonStr)
}

// Event returns a single event by its stable event identifier
// (EKEvent.eventIdentifier). Returns [ErrNotFound] if no event matches.
func (c *Client) Event(id string) (*Event, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_cal_get_event(cID)
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("calendar: %w", err)
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	return parseEventJSON(jsonStr)
}

// CreateEvent creates a new calendar event and returns it with its assigned ID.
// The event is saved to the EventKit store immediately.
func (c *Client) CreateEvent(input CreateEventInput) (*Event, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	for _, rule := range input.RecurrenceRules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("calendar: invalid recurrence rule: %w", err)
		}
	}

	jsonBytes, err := marshalCreateInput(input)
	if err != nil {
		return nil, fmt.Errorf("calendar: failed to marshal input: %w", err)
	}

	cJSON := C.CString(string(jsonBytes))
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_cal_create_event(cJSON)
	if res.error != nil {
		return nil, fmt.Errorf("calendar: %w", resultErr(res))
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	return parseEventJSON(jsonStr)
}

// UpdateEvent updates an existing event and returns the updated version.
// Only non-nil fields in the input are modified. The span parameter controls
// whether the change applies to just this occurrence or all future occurrences
// of a recurring event. Returns [ErrNotFound] if the event does not exist.
func (c *Client) UpdateEvent(id string, input UpdateEventInput, span Span) (*Event, error) {
	if err := validateSpan(span); err != nil {
		return nil, err
	}
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if input.RecurrenceRules != nil {
		for _, rule := range *input.RecurrenceRules {
			if err := rule.Validate(); err != nil {
				return nil, fmt.Errorf("calendar: invalid recurrence rule: %w", err)
			}
		}
	}

	jsonBytes, err := marshalUpdateInput(input)
	if err != nil {
		return nil, fmt.Errorf("calendar: failed to marshal input: %w", err)
	}

	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	cJSON := C.CString(string(jsonBytes))
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_cal_update_event(cID, cJSON, C.int(span))
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("calendar: %w", err)
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	return parseEventJSON(jsonStr)
}

// DeleteEvent permanently removes an event.
// The span parameter controls whether the deletion applies to just this
// occurrence or all future occurrences of a recurring event.
// Returns [ErrNotFound] if the event does not exist.
func (c *Client) DeleteEvent(id string, span Span) error {
	if err := validateSpan(span); err != nil {
		return err
	}
	if err := validate.ID(id); err != nil {
		return err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_cal_delete_event(cID, C.int(span))
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("calendar: %w", err)
	}
	C.ek_cal_free(res.result)
	return nil
}

// CreateCalendar creates a new calendar and returns it with its assigned ID.
// The calendar is saved to the EventKit store immediately.
func (c *Client) CreateCalendar(input CreateCalendarInput) (*Calendar, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if input.Source == "" && input.SourceID == "" {
		return nil, fmt.Errorf("calendar: source is required")
	}
	jsonBytes, err := marshalCreateCalendarInput(input)
	if err != nil {
		return nil, fmt.Errorf("calendar: failed to marshal input: %w", err)
	}

	cJSON := C.CString(string(jsonBytes))
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_cal_create_calendar(cJSON)
	if res.error != nil {
		return nil, fmt.Errorf("calendar: %w", resultErr(res))
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	cals, err := parseCalendarsJSON("[" + jsonStr + "]")
	if err != nil {
		return nil, err
	}
	if len(cals) == 0 {
		return nil, fmt.Errorf("calendar: unexpected empty response")
	}
	return &cals[0], nil
}

// UpdateCalendar updates an existing calendar and returns the updated version.
// Only non-nil fields in the input are modified.
// Returns [ErrNotFound] if the calendar does not exist.
// Returns [ErrImmutable] if the calendar is immutable.
func (c *Client) UpdateCalendar(id string, input UpdateCalendarInput) (*Calendar, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	jsonBytes, err := marshalUpdateCalendarInput(input)
	if err != nil {
		return nil, fmt.Errorf("calendar: failed to marshal input: %w", err)
	}

	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	cJSON := C.CString(string(jsonBytes))
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_cal_update_calendar(cID, cJSON)
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, ErrImmutable) {
			return nil, ErrImmutable
		}
		return nil, fmt.Errorf("calendar: %w", err)
	}
	defer C.ek_cal_free(res.result)

	jsonStr := C.GoString(res.result)
	cals, err := parseCalendarsJSON("[" + jsonStr + "]")
	if err != nil {
		return nil, err
	}
	if len(cals) == 0 {
		return nil, fmt.Errorf("calendar: unexpected empty response")
	}
	return &cals[0], nil
}

// DeleteCalendar permanently removes a calendar and all its events.
// Returns [ErrNotFound] if the calendar does not exist.
// Returns [ErrImmutable] if the calendar is immutable.
func (c *Client) DeleteCalendar(id string) error {
	if err := validate.ID(id); err != nil {
		return err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_cal_delete_calendar(cID)
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, ErrImmutable) {
			return ErrImmutable
		}
		return fmt.Errorf("calendar: %w", err)
	}
	C.ek_cal_free(res.result)
	return nil
}
