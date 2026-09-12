# Migration from upstream v0.15.0

The project has been renamed from `pw0rld/go-eventkit` to `pw0rld/macos-agenda`. Update existing fork imports and Git remotes to the new name. The root Go package remains named `eventkit` for the underlying framework types. The executable is `agenda`; build with `make build` to include macOS privacy metadata.

The module path is now `github.com/pw0rld/macos-agenda`. Update import paths and your module requirement together. Pin a reviewed fork commit or a future fork release; do not assume existing upstream tags contain these changes.

| Previous API/behavior | Maintained fork |
|---|---|
| Reminder or event prefix lookup | Complete ID only |
| First matching container name | Unique name required; prefer CalendarID/ListID |
| WithListID treated as a name | Exact identifier matching |
| CreateEventInput.Calendar / UpdateEventInput.Calendar | Still supported; added CalendarID, mutually exclusive with name |
| CreateReminderInput.ListName / UpdateReminderInput.ListName | Still supported; added ListID, mutually exclusive with name |
| CreateCalendarInput.Source / CreateListInput.Source | Still supported for unique eligible sources; added SourceID |
| Flagged, Tags, WithTags | Removed; no native flag/tag support |
| IsShared, SharedToMe, IsOwnedByMe | Removed; no optimistic private-metadata fallback |
| Private reminder URL attachment | Public URL property only; native UI presentation may differ |
| AttendeeInput, write Attendees, TravelTime, SelfStatus | Removed; public read-only attendee metadata remains |
| AttendeeWritesSupported, RSVPSupported, AvailabilitySupported, RespondToInvitation, RequestAvailability, PendingInvitations | Removed |
| AvailabilityType, AvailabilitySpan, Invitation, ErrUnsupportedFeature | Removed with scheduling APIs |
| WatchChanges | Removed; use explicit bounded queries |
| DeleteEvents / DeleteReminders | Removed; resolve and authorize each individual deletion |
| Native errors may unwind through cgo | Public entry points catch Objective-C exceptions |
| Invalid times accepted or arithmetic wraps | Invalid input and overflow return errors |

Unneeded private-framework research notes, obsolete planning documents, duplicated mock tests, and real-data mutation demos were removed. Git history retains them. Core JSON/date/recurrence tests remain, with new validation and synthetic native regressions.

Date parsing keeps ordinary existing syntax, but rejects invalid time suffixes, out-of-range minutes/default hours and excessively large relative counts. Relative counts are capped at 1,000,000. Exact dates used for EventKit mutations must be nonzero and in years 1–9999.

For safe retries, query the target after uncertain provider errors. A successful EventKit save is not a guarantee that the remote provider has synchronized the change.
