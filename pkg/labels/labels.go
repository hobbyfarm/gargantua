package labels

import "strings"

const (
	SettingScope  = "hobbyfarm.io/setting-scope"
	SettingWeight = "hobbyfarm.io/setting-weight"
)

func DotEscapeLabel(label string) string {
	return strings.ReplaceAll(label, ".", "\\.")
}
