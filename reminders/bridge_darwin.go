//go:build darwin

package reminders

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
	"github.com/pw0rld/go-eventkit/internal/validate"
	"strings"
	"sync"
	"unsafe"
)

var bridgeMu sync.Mutex

func resultErr(res C.ek_result_t) error {
	if res.error != nil {
		msg := C.GoString(res.error)
		C.ek_rem_free(res.error)
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

// New creates a new Reminders [Client] and requests reminders access.
//
// On first call, macOS displays a TCC prompt requesting reminders access.
// Returns [ErrAccessDenied] if the user denies access.
// Returns [ErrUnsupported] on non-darwin platforms.
func New() (*Client, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	res := C.ek_rem_request_access()
	if res.error != nil {
		err := resultErr(res)
		if strings.Contains(err.Error(), "access denied") {
			return nil, fmt.Errorf("%w: %s", ErrAccessDenied, err)
		}
		return nil, err
	}
	C.ek_rem_free(res.result)
	return &Client{}, nil
}

// Lists returns all reminder lists across all accounts (iCloud, Exchange, etc.).
func (c *Client) Lists() ([]List, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	res := C.ek_rem_fetch_lists()
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return parseListsJSON(C.GoString(res.result))
}

// Reminders returns reminders matching the given filter options.
// With no options, returns all reminders across all lists.
// Options can filter by list, completion status, search query, and due date range.
func (c *Client) Reminders(opts ...ListOption) ([]Reminder, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	o := applyOptions(opts)
	if err := validateQuery(o); err != nil {
		return nil, err
	}
	strictID := C.int(0)
	if o.listIDSet {
		strictID = 1
	}

	var cList, cCompleted, cSearch, cBefore, cAfter *C.char

	if o.listName != "" {
		cList = C.CString(o.listName)
		defer C.free(unsafe.Pointer(cList))
	} else if o.listID != "" {
		cList = C.CString(o.listID)
		defer C.free(unsafe.Pointer(cList))
	}
	if o.completed != nil {
		if *o.completed {
			cCompleted = C.CString("true")
		} else {
			cCompleted = C.CString("false")
		}
		defer C.free(unsafe.Pointer(cCompleted))
	}
	if o.search != "" {
		cSearch = C.CString(o.search)
		defer C.free(unsafe.Pointer(cSearch))
	}
	if o.dueBefore != nil {
		s := o.dueBefore.UTC().Format("2006-01-02T15:04:05.000Z")
		cBefore = C.CString(s)
		defer C.free(unsafe.Pointer(cBefore))
	}
	if o.dueAfter != nil {
		s := o.dueAfter.UTC().Format("2006-01-02T15:04:05.000Z")
		cAfter = C.CString(s)
		defer C.free(unsafe.Pointer(cAfter))
	}

	res := C.ek_rem_fetch_reminders(cList, cCompleted, cSearch, cBefore, cAfter, strictID)
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return parseRemindersJSON(C.GoString(res.result))
}

// Reminder returns a single reminder by ID.
// Requires the complete identifier returned by the store; prefixes are rejected.
// Returns [ErrNotFound] if no reminder matches.
func (c *Client) Reminder(id string) (*Reminder, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_rem_get_reminder(cID)
	if res.error != nil {
		return nil, resultErr(res)
	}
	defer C.ek_rem_free(res.result)
	return parseReminderJSON(C.GoString(res.result))
}

// CreateReminder creates a new reminder and returns it with its assigned ID.
// The reminder is saved to the EventKit store immediately.
func (c *Client) CreateReminder(input CreateReminderInput) (*Reminder, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	for _, rule := range input.RecurrenceRules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("reminders: invalid recurrence rule: %w", err)
		}
	}

	jsonStr, err := marshalCreateInput(input)
	if err != nil {
		return nil, err
	}

	cJSON := C.CString(jsonStr)
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_rem_create_reminder(cJSON)
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return parseReminderJSON(C.GoString(res.result))
}

// UpdateReminder updates an existing reminder and returns the updated version.
// Only non-nil fields in the input are modified. Returns [ErrNotFound] if the
// reminder does not exist.
func (c *Client) UpdateReminder(id string, input UpdateReminderInput) (*Reminder, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if input.RecurrenceRules != nil {
		for _, rule := range *input.RecurrenceRules {
			if err := rule.Validate(); err != nil {
				return nil, fmt.Errorf("reminders: invalid recurrence rule: %w", err)
			}
		}
	}

	jsonStr, err := marshalUpdateInput(input)
	if err != nil {
		return nil, err
	}

	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	cJSON := C.CString(jsonStr)
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_rem_update_reminder(cID, cJSON)
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return parseReminderJSON(C.GoString(res.result))
}

// DeleteReminder permanently deletes a reminder by ID.
// Returns [ErrNotFound] if the reminder does not exist.
func (c *Client) DeleteReminder(id string) error {
	if err := validate.ID(id); err != nil {
		return err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_rem_delete_reminder(cID)
	if res.error != nil {
		return fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return nil
}

// CreateList creates a new reminder list and returns it with its assigned ID.
// The list is saved to the EventKit store immediately.
func (c *Client) CreateList(input CreateListInput) (*List, error) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if input.Source == "" && input.SourceID == "" {
		return nil, fmt.Errorf("reminders: source is required")
	}
	jsonStr, err := marshalCreateListInput(input)
	if err != nil {
		return nil, err
	}

	cJSON := C.CString(jsonStr)
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_rem_create_list(cJSON)
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)

	jsonResp := C.GoString(res.result)
	lists, err := parseListsJSON("[" + jsonResp + "]")
	if err != nil {
		return nil, err
	}
	if len(lists) == 0 {
		return nil, fmt.Errorf("reminders: unexpected empty response")
	}
	return &lists[0], nil
}

// UpdateList updates an existing reminder list and returns the updated version.
// Only non-nil fields in the input are modified.
// Returns [ErrNotFound] if the list does not exist.
// Returns [ErrImmutable] if the list is immutable.
func (c *Client) UpdateList(id string, input UpdateListInput) (*List, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	jsonStr, err := marshalUpdateListInput(input)
	if err != nil {
		return nil, err
	}

	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	cJSON := C.CString(jsonStr)
	defer C.free(unsafe.Pointer(cJSON))

	res := C.ek_rem_update_list(cID, cJSON)
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, ErrImmutable) {
			return nil, ErrImmutable
		}
		return nil, fmt.Errorf("reminders: %w", err)
	}
	defer C.ek_rem_free(res.result)

	jsonResp := C.GoString(res.result)
	lists, err := parseListsJSON("[" + jsonResp + "]")
	if err != nil {
		return nil, err
	}
	if len(lists) == 0 {
		return nil, fmt.Errorf("reminders: unexpected empty response")
	}
	return &lists[0], nil
}

// DeleteList permanently removes a reminder list and all its reminders.
// Returns [ErrNotFound] if the list does not exist.
// Returns [ErrImmutable] if the list is immutable.
func (c *Client) DeleteList(id string) error {
	if err := validate.ID(id); err != nil {
		return err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_rem_delete_list(cID)
	if res.error != nil {
		err := resultErr(res)
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, ErrImmutable) {
			return ErrImmutable
		}
		return fmt.Errorf("reminders: %w", err)
	}
	defer C.ek_rem_free(res.result)
	return nil
}

// CompleteReminder marks a reminder as completed and returns the updated version.
// Sets [Reminder.Completed] to true and [Reminder.CompletionDate] to now.
// Returns [ErrNotFound] if the reminder does not exist.
func (c *Client) CompleteReminder(id string) (*Reminder, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_rem_complete_reminder(cID)
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return parseReminderJSON(C.GoString(res.result))
}

// UncompleteReminder marks a reminder as incomplete and returns the updated version.
// Sets [Reminder.Completed] to false and clears [Reminder.CompletionDate].
// Returns [ErrNotFound] if the reminder does not exist.
func (c *Client) UncompleteReminder(id string) (*Reminder, error) {
	if err := validate.ID(id); err != nil {
		return nil, err
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	res := C.ek_rem_uncomplete_reminder(cID)
	if res.error != nil {
		return nil, fmt.Errorf("reminders: %w", resultErr(res))
	}
	defer C.ek_rem_free(res.result)
	return parseReminderJSON(C.GoString(res.result))
}
