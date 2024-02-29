package util

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
)

func GenericUnmarshal[T any](rawObj string, propName string) (T, error) {
	var obj T
	err := json.Unmarshal([]byte(rawObj), &obj)
	if err != nil {
		glog.Errorf("error while unmarshaling %s: %v", propName, err)
		return obj, fmt.Errorf("bad")
	}
	return obj, nil
}
