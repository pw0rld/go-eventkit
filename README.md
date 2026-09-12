# go-eventkit

A small macOS Calendar and Reminders library maintained at `github.com/pw0rld/go-eventkit`.
It uses Go + cgo + Apple's **public EventKit APIs**. No private frameworks, shell commands, telemetry, direct network clients, or third-party Go modules are used by the library.

This fork intentionally makes breaking API changes from upstream v0.15.0. See [MIGRATION.md](MIGRATION.md).

## Supported scope

- List calendars and reminder lists, including their account/source IDs.
- Query events by date range; query reminders by list ID, status, text and due date.
- Create/update/delete individual events and reminders; complete/uncomplete reminders.
- Create/update/delete calendar containers and reminder lists.
- Event locations, notes, public URL fields, alarms and recurrence rules.
- Reminder priorities, due dates, alarms and recurrence rules.
- Conference links derived from public event URL/location/notes text.
- Strict date and numeric validation before entering the native framework.

Not included: native reminder tags/flags, private sharing metadata, private URL attachments, attendee invitations, RSVP, server free/busy queries, notification watchers and bulk deletion.

## Requirements and installation

- macOS 14+ for the maintained runtime (the older authorization fallback remains for compilation compatibility).
- Go 1.24.5+ and Apple Command Line Tools to build. No full Xcode installation is required.
- Calendar and/or Reminders privacy access for the application launching the binary.

For local development, clone this fork and use a Go module `replace` directive pointing to the checkout. Until a fork release is published, pin a reviewed **fork commit**, rather than using an upstream tag or `@latest`.

```go
import (
    "github.com/pw0rld/go-eventkit/calendar"
    "github.com/pw0rld/go-eventkit/reminders"
)
```

`New()` requests the corresponding macOS permission; it may show a system dialog. The library does not bypass denied permissions. Permission requests have a 60-second deadline; reminder fetches have a 30-second deadline.

## Use exact IDs

Discover the ID first, then use it for writes. Titles are not stable identities. Name-based selection is supported only when it resolves uniquely. Specifying both a name and an ID is an error. Explicitly empty query selectors fail instead of expanding the query to all calendars.

```go
client, err := calendar.New()
if err != nil { return err }
calendars, err := client.Calendars()
if err != nil { return err }
// Select the intended account and calendar from calendars; retain its ID.
_ = calendars

start, err := time.Parse(time.RFC3339, "2026-09-15T09:00:00+08:00")
if err != nil { return err }
event, err := client.CreateEvent(calendar.CreateEventInput{
    Title: "Team meeting",
    CalendarID: selectedCalendarID,
    StartDate: start,
    EndDate: start.Add(time.Hour),
    TimeZone: "Asia/Shanghai",
    Notes: "Prepare the agenda",
})
if err != nil { return err }
_ = event.ID // Keep the complete ID for later updates/deletion.
```

The snippets run inside your application's function; import `time` and supply `selectedCalendarID` after selecting your account. A reminder example:

```go
client, err := reminders.New()
if err != nil { return err }
item, err := client.CreateReminder(reminders.CreateReminderInput{
    Title: "Prepare slides",
    ListID: selectedListID,
    DueDate: &due,
    RemindMeDate: &alarm,
    Priority: reminders.PriorityHigh,
})
if err != nil { return err }
_, err = client.CompleteReminder(item.ID)
return err
```

Supply `selectedListID`, `due` and `alarm` explicitly. Query with `reminders.WithListID(selectedListID)`. Reminder IDs and event lookups are exact; truncated prefixes are unsupported.

## Operational behavior

- Omitting a write destination uses the system default. Automated callers should always set an explicit ID.
- Ordinary reminder edits and list changes are submitted in one public EventKit save. Moves that EventKit rejects, including some shared-list moves, return an error; there is no private fallback or delete/recreate workaround.
- URL is the public `EKCalendarItem.URL` property. It may not appear in the native Reminders app's special URL attachment field.
- Saving to iCloud/Exchange/shared calendars still uses normal system synchronization. Editing/deleting an existing meeting can have account-provider notification effects even though this fork cannot add invitees or issue RSVP calls.
- `SpanThisEvent` and `SpanFutureEvents` are explicit. The API resolves an event by EventKit ID; it does not offer an occurrence-date selector. Do not assume an ID alone uniquely selects an arbitrary later occurrence of a series.
- Native calls are serialized per package and Objective-C exceptions are converted into errors. The library does not provide transactions across multiple API calls. An error after a successful system commit (for example output serialization failing) may still require re-querying before retrying a create.
- Due dates use the system calendar's date components. There is no separate date-only or per-reminder timezone API; callers should verify the displayed due time when working across timezones.
- Destructive operations are explicit library calls. An agent-facing wrapper should show the resolved target and require appropriate user authorization; this library is not an authorization boundary.

## Checks

```sh
make check
```

This runs Go tests, vet, synthetic native selector regressions and a Linux stub build. It does **not** request EventKit permissions or read/write real calendar data. Native fixtures instantiate fake stores and test the actual static selector helpers. Real iCloud/Exchange behavior still needs testing with a dedicated disposable calendar/list before production use.

## License

MIT, retaining the original copyright and license. Upstream project: [BRO3886/go-eventkit](https://github.com/BRO3886/go-eventkit).
