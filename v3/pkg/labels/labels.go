package labels

import (
	"strings"
)

const (
	SettingScope           = "hobbyfarm.io/setting-scope"
	SettingWeight          = "hobbyfarm.io/setting-weight"
	SettingGroup           = "hobbyfarm.io/setting-group"
	AccessCodeLabel        = "hobbyfarm.io/accesscode"
	OneTimeAccessCodeLabel = "hobbyfarm.io/otac"
	ScheduledEventLabel    = "hobbyfarm.io/scheduledevent"
	SessionLabel           = "hobbyfarm.io/session"
	UserLabel              = "hobbyfarm.io/user"
	RBACManagedLabel       = "rbac.hobbyfarm.io/managed"
	EnvironmentLabel       = "hobbyfarm.io/environment"
	VirtualMachineTemplate = "hobbyfarm.io/virtualmachinetemplate"
)

func DotEscapeLabel(label string) string {
	return strings.ReplaceAll(label, ".", "\\.")
}

func UpdateCategoryLabels(labels map[string]string, oldCategories []string, newCategories []string) map[string]string {
	for _, category := range oldCategories {
		labels["category-"+category] = "false"
	}
	for _, category := range newCategories {
		labels["category-"+category] = "true"
	}
	return labels
}
