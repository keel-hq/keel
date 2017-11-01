package policies

import (
	"github.com/keel-hq/keel/types"
)

// GetPolicy - gets policy
func GetPolicy(labels map[string]string) types.PolicyType {
	for k, v := range labels {
		switch k {
		case types.KeelPolicyLabel:
			return types.ParsePolicy(v)
		case "keel.observer/policy":
			return types.ParsePolicy(v)
		}
	}

	return types.PolicyTypeNone
}

// GetTriggerPolicy - checks for trigger label, if not set - returns
// default trigger type
func GetTriggerPolicy(labels map[string]string) types.TriggerType {
	trigger, ok := labels[types.KeelTriggerLabel]
	if ok {
		return types.ParseTrigger(trigger)
	}
	return types.TriggerTypeDefault
}
