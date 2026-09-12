#import <EventKit/EventKit.h>
int agenda_authorization(int reminder) {
 return (int)[EKEventStore authorizationStatusForEntityType:reminder ? EKEntityTypeReminder : EKEntityTypeEvent];
}
