# Maintenance rules

- Scope: public EventKit calendar/reminder CRUD, validation, alarms and recurrence.
- Do not introduce private Apple frameworks/selectors, shell execution, telemetry or direct network clients into the library.
- Native writes must resolve exact IDs or reject ambiguous names. Do not silently broaden an empty explicit query scope.
- Keep cgo internal, ARC enabled, and all native store calls serialized per package.
- Run `make check` for code changes. These checks use synthetic data only.
- Do not run tests against real calendar/reminder data unless the user explicitly asks for that and provides a disposable target.
- Preserve MIT attribution. Document breaking public API changes in MIGRATION.md.
