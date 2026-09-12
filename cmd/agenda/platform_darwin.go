//go:build darwin

package main

/*
#cgo LDFLAGS: -framework CoreFoundation -framework EventKit
#include <CoreFoundation/CoreFoundation.h>
#include <objc/objc.h>
// Implemented in platform_darwin.m; status does not request permission.
int agenda_authorization(int reminder);
static void agenda_prepare(void) {
    CFRunLoopSourceContext context = {0};
    CFRunLoopSourceRef source = CFRunLoopSourceCreate(NULL, 0, &context);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopDefaultMode);
    CFRelease(source);
}
static void agenda_pump(void) { CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.02, false); }
*/
import "C"
import "runtime"

func platformRun(fn func() int) int {
	runtime.LockOSThread()
	C.agenda_prepare()
	done := make(chan int, 1)
	go func() { done <- fn() }()
	for {
		select {
		case code := <-done:
			return code
		default:
			C.agenda_pump()
		}
	}
}

func permissionStatus() map[string]string {
	name := func(n int) string {
		switch n {
		case 0:
			return "notDetermined"
		case 1:
			return "restricted"
		case 2:
			return "denied"
		case 3:
			return "fullAccess"
		case 4:
			return "writeOnly"
		default:
			return "unknown"
		}
	}
	return map[string]string{"calendar": name(int(C.agenda_authorization(0))), "reminders": name(int(C.agenda_authorization(1)))}
}
