package uid

import (
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

func RemoveUIDPublic(uid types.UID) types.UID {
	return types.UID(strings.TrimSuffix(string(uid), "-p"))
}
