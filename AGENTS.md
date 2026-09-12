# Maintenance rules

- Scope: public EventKit calendar/reminder CRUD, validation, alarms, recurrence and the agenda CLI.
- Do not introduce private Apple frameworks/selectors, shell execution, telemetry or direct network clients into the library.
- Native writes must resolve exact IDs or reject ambiguous names. Do not silently broaden an empty explicit query scope.
- Keep cgo internal, ARC enabled, and all native store calls serialized per package.
- Run `make check` for code changes. These checks use synthetic data only.
- Run `make build` and test CLI help/dry-run after CLI changes. Never open EventKit stores before validating a request; dry-run and status must not request access.
- Do not run tests against real calendar/reminder data unless the user explicitly asks for that and provides a disposable target.
- Preserve MIT attribution. Document breaking public API changes in MIGRATION.md.
