package main

import "testing"

func TestDelayedSchedule(t *testing.T) {
	tests := []struct {
		name, donor string
		hours       int
		want        string
	}{
		{name: "receipt after six hours", donor: "donor-17", hours: 6, want: "0 6 * * *"},
		{name: "reminder after one hour", donor: "volunteer-3", hours: 1, want: "0 1 * * *"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := delayedSchedule(followUp{Donor: tt.donor, Hours: tt.hours})
			if err != nil || got != tt.want {
				t.Fatalf("schedule = %q, err = %v; want %q", got, err, tt.want)
			}
		})
	}
}
