package eventkit

import "testing"

func TestInvalidRecurrenceRanges(t *testing.T) {
	for _, r := range []RecurrenceRule{
		{Frequency: 99, Interval: 1},
		{Frequency: FrequencyWeekly, Interval: 1, DaysOfTheWeek: []RecurrenceDayOfWeek{{DayOfTheWeek: 0}}},
		{Frequency: FrequencyMonthly, Interval: 1, DaysOfTheMonth: []int{32}},
		{Frequency: FrequencyYearly, Interval: 1, MonthsOfTheYear: []int{-1}},
		{Frequency: FrequencyYearly, Interval: 1, DaysOfTheYear: []int{0}},
	} {
		if err := r.Validate(); err == nil {
			t.Errorf("invalid recurrence accepted: %+v", r)
		}
	}
}
