package policies

import (
	"testing"

	"github.com/keel-hq/keel/types"
)

func TestGetTriggerPolicy(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
		want        types.TriggerType
	}{
		{name: "unset defaults to webhook events", want: types.TriggerTypeDefault},
		{name: "webhook annotation", annotations: map[string]string{types.KeelTriggerLabel: "webhook"}, want: types.TriggerTypeDefault},
		{name: "poll annotation", annotations: map[string]string{types.KeelTriggerLabel: "poll"}, want: types.TriggerTypePoll},
		{name: "webhook label", labels: map[string]string{types.KeelTriggerLabel: "webhook"}, want: types.TriggerTypeDefault},
		{name: "poll label", labels: map[string]string{types.KeelTriggerLabel: "poll"}, want: types.TriggerTypePoll},
		{
			name:        "annotation takes precedence over label",
			labels:      map[string]string{types.KeelTriggerLabel: "poll"},
			annotations: map[string]string{types.KeelTriggerLabel: "webhook"},
			want:        types.TriggerTypeDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTriggerPolicy(tt.labels, tt.annotations); got != tt.want {
				t.Fatalf("GetTriggerPolicy() = %s, want %s", got, tt.want)
			}
		})
	}
}
