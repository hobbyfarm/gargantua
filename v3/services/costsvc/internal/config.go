package costservice

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
)

var (
	DefaultConfigPath = "/etc"
	DefaultConfigName = "cost-config.json"
)

type GroupVersionResourceJSON struct {
	schema.GroupVersionResource
}

func ParseConfig() []schema.GroupVersionResource {
	file := fmt.Sprintf("%s/%s", DefaultConfigPath, DefaultConfigName)

	content, err := os.ReadFile(file)
	if err != nil {
		glog.Fatalf("Error reading config %s: %s", file, err.Error())
	}

	var resources []GroupVersionResourceJSON

	if err = json.Unmarshal(content, &resources); err != nil {
		glog.Fatalf("Error parsing config %s: %s", file, err.Error())
	}

	out := make([]schema.GroupVersionResource, len(resources))
	for i, resource := range resources {
		out[i] = resource.GroupVersionResource
	}

	return out
}
