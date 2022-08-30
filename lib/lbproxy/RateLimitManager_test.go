package lbproxy

import (
	"reflect"
	"testing"
)

func Test_trimTimestamps(t *testing.T) {
	type args struct {
		ts          []int64
		windowStart int64
	}
	tests := []struct {
		name string
		args args
		want []int64
	}{
		{
			name: "empty",
			args: args{
				ts:          []int64{},
				windowStart: 100,
			},
			want: []int64{},
		},
		{
			name: "single in",
			args: args{
				ts:          []int64{101},
				windowStart: 100,
			},
			want: []int64{101},
		},
		{
			name: "single out",
			args: args{
				ts:          []int64{99},
				windowStart: 100,
			},
			want: []int64{},
		},
		{
			name: "slice",
			args: args{
				ts:          []int64{99, 100, 101},
				windowStart: 100,
			},
			want: []int64{100, 101},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimTimestamps(tt.args.ts, tt.args.windowStart); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trimTimestamps() = %v, want %v", got, tt.want)
			}
		})
	}
}
