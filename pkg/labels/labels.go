package labels

import "strings"

const (
	SettingScope = "hobbyfarm.io/setting-scope"
)

func DotEscapeLabel(label string) string {
	return strings.ReplaceAll(label, ".", "\\.")
}
