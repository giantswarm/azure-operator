package scalestrategy

import "testing"

func TestSafeQuick_GetNodeCount(t *testing.T) {
	tests := []struct {
		name         string
		currentCount int64
		desiredCount int64
		want         int64
	}{
		{
			name:         "Size increased by 2",
			currentCount: 2,
			desiredCount: 4,
			want:         4,
		},
		{
			name:         "Size increased by 20",
			currentCount: 10,
			desiredCount: 20,
			want:         15,
		},
		{
			name:         "Size increased by 20, second run",
			currentCount: 13,
			desiredCount: 20,
			want:         18,
		},
		{
			name:         "Size increased by 20, last run",
			currentCount: 18,
			desiredCount: 20,
			want:         20,
		},
		{
			name:         "Size decreased by 1",
			currentCount: 5,
			desiredCount: 4,
			want:         4,
		},
		{
			name:         "Size decreased by 10",
			currentCount: 20,
			desiredCount: 10,
			want:         10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Staircase{}
			if got := i.GetNodeCount(tt.currentCount, tt.desiredCount); got != tt.want {
				t.Errorf("GetNodeCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
