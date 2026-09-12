GO ?= go
CLANG ?= clang
NATIVE_FLAGS = -fobjc-arc -fblocks -framework EventKit -framework Foundation -framework AppKit -framework CoreLocation

.PHONY: check test native-test vet portable
check: test vet native-test portable

test:
	$(GO) test -timeout 30s ./...

vet:
	$(GO) vet ./...

native-test:
	mkdir -p .build
	$(CLANG) $(NATIVE_FLAGS) tests/native/selectors.m -o .build/calendar-selectors
	$(CLANG) $(NATIVE_FLAGS) -DTEST_REMINDERS=1 tests/native/selectors.m -o .build/reminder-selectors
	.build/calendar-selectors
	.build/reminder-selectors

portable:
	GOOS=linux CGO_ENABLED=0 $(GO) build ./...
