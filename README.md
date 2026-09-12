# macos-agenda

A macOS Calendar and Reminders CLI (`agenda`) and Go library maintained at `github.com/pw0rld/macos-agenda`.
It uses Go + cgo + Apple's **public EventKit APIs**. No private frameworks, shell commands, telemetry, direct network clients, or third-party Go modules are used by the library.

This independently maintained project makes breaking API changes from upstream go-eventkit v0.15.0. See [MIGRATION.md](MIGRATION.md).

## Build and run the CLI

```sh
make build
./.build/agenda --help
./.build/agenda status
```

The binary is `.build/agenda`, built for the current Mac architecture. The build embeds an Info.plist with Calendar/Reminders usage descriptions and applies a local ad-hoc signature. Keep the binary at a stable path. This is a local build, not a notarized release. Use `make build` rather than plain `go install` so the privacy metadata is included.

`status` only checks permissions; it does not prompt or read calendar content. To grant access, run these from the terminal application you intend to use and respond to macOS's dialogs:

```sh
./.build/agenda authorize calendar
./.build/agenda authorize reminders
./.build/agenda calendars
./.build/agenda lists
```

Authorization may be attributed to the launching app. A Terminal grant does not necessarily grant a different agent host access. The CLI pumps the main run loop while EventKit executes on a worker, allowing normal system callbacks.

Queries return JSON:

```sh
./.build/agenda events list --from 2026-09-14T00:00:00+08:00 --to 2026-09-21T00:00:00+08:00
./.build/agenda reminders list --completed false
```

For writes, use a JSON file or `--input -` for stdin. Preview the supplied examples without accessing either store:

```sh
./.build/agenda events create --input examples/event.json --dry-run
./.build/agenda reminders create --input examples/reminder.json --dry-run
```

Replace the example's `calendarID`/`listID` with the exact ID from `calendars`/`lists`, adjust the dates, then omit `--dry-run` to save. Dry run validates the payload only; it does not establish that the target exists or is writable. Unlike the library's optional default destination, **CLI creation requires an explicit container ID**.

```sh
./.build/agenda reminders update --id COMPLETE_ID --input update.json
./.build/agenda reminders complete --id COMPLETE_ID
./.build/agenda reminders delete --id COMPLETE_ID --confirm-id COMPLETE_ID
```

An update JSON such as `{"notes":""}` clears notes while leaving other fields untouched. `null` means omitted for pointer fields; use empty strings to clear text, `clearDueDate: true` to remove a reminder due date, and `alarms: []` to clear alarms. Empty updates, unknown/duplicate fields, inputs over 1 MiB and nesting over 128 levels are rejected. All date/time fields use RFC3339 with explicit UTC offsets. Advanced duration fields such as `alerts[].relativeOffset` follow Go's JSON representation in nanoseconds; `-900000000000` means 15 minutes before. Recurrence input follows the library types below.

Deletion requires matching `--id` and `--confirm-id`; it is not an interactive confirmation. Resolve and review the target with `get` before deleting. Event update/delete accepts `--span this|future` (default `this`). CLI container operations are limited to listing; container creation/deletion remains a library API.

Success JSON goes to stdout, error JSON to stderr. Exit codes: 0 success, 64 invalid request, 2 denied EventKit access, 1 other failures. Do not blindly retry a create after an uncertain system/provider error.

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

For local development, clone this repository and use a Go module `replace` directive pointing to the checkout. Until a release is published, pin a reviewed **macos-agenda commit**, rather than using an upstream tag or `@latest`.

```go
import (
    "github.com/pw0rld/macos-agenda/calendar"
    "github.com/pw0rld/macos-agenda/reminders"
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
