// Standalone regression test. All store data below is synthetic; no EKEventStore is created.
#if TEST_REMINDERS
#include "../../reminders/bridge_darwin.m"
#else
#include "../../calendar/bridge_darwin.m"
#endif
#include <assert.h>

@interface FakeCalendar : NSObject
@property NSString* title;
@property NSString* calendarIdentifier;
@end
@implementation FakeCalendar
@end

@interface FakeStore : NSObject
@property NSArray* calendars;
@property NSDictionary* items;
- (NSArray*)calendarsForEntityType:(EKEntityType)type;
- (id)calendarItemWithIdentifier:(NSString*)identifier;
@end
@implementation FakeStore
- (NSArray*)calendarsForEntityType:(EKEntityType)type { return self.calendars; }
- (id)calendarItemWithIdentifier:(NSString*)identifier { return self.items[identifier]; }
@end

@interface FakeReminder : NSObject
@end
@implementation FakeReminder
- (BOOL)isKindOfClass:(Class)cls { return cls == [EKReminder class] || [super isKindOfClass:cls]; }
@end

int main(void) {
 @autoreleasepool {
  FakeCalendar* a=[FakeCalendar new]; a.title=@"Work"; a.calendarIdentifier=@"id-a";
  FakeCalendar* b=[FakeCalendar new]; b.title=@"work"; b.calendarIdentifier=@"id-b";
  FakeStore* fixture=[FakeStore new]; fixture.calendars=@[a,b];
  EKEventStore* store=(EKEventStore*)fixture;
#if TEST_REMINDERS
  assert(find_list_by_name(store,@"Work") == nil);
  assert(find_list_by_id(store,@"id-b") == (id)b);
  fixture.items=@{@"ABC-1":[FakeReminder new],@"ABC-2":[FakeReminder new]};
  assert(find_reminder_by_id(store,@"ABC") == nil);
  assert(find_reminder_by_id(store,@"") == nil);
  assert(find_reminder_by_id(store,@"ABC-2") == fixture.items[@"ABC-2"]);
#else
  assert(find_calendar_by_name(store,@"Work") == nil);
  assert(find_calendar_by_id(store,@"id-b") == (id)b);
#endif
  puts("native target selection: PASS (synthetic store)");
 }
 return 0;
}
