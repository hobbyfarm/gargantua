package labels

import "strings"

const (
	SettingScope  = "hobbyfarm.io/setting-scope"
	SettingWeight = "hobbyfarm.io/setting-weight"
	SettingGroup  = "hobbyfarm.io/setting-group"
)

func DotEscapeLabel(label string) string {
	return strings.ReplaceAll(label, ".", "\\.")
}
