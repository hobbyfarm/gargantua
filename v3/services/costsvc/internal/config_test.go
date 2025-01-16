package costservice

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	originalPath := DefaultConfigPath
	originalName := DefaultConfigName

	defer func() {
		DefaultConfigPath = originalPath
		DefaultConfigName = originalName
	}()

	tests := []struct {
		name  string
		input []GroupVersionResourceJSON
		want  []schema.GroupVersionResource
	}{
		{
			name:  "empty",
			input: []GroupVersionResourceJSON{},
			want:  []schema.GroupVersionResource{},
		},
		{
			name: "non empty",
			input: []GroupVersionResourceJSON{
				{
					GroupVersionResource: schema.GroupVersionResource{
						Group:    "hobbyfarm.io",
						Version:  "v1",
						Resource: "virtualmachines",
					},
				},
				{
					GroupVersionResource: schema.GroupVersionResource{
						Group:    "hobbyfarm.io",
						Version:  "v1",
						Resource: "courses",
					},
				},
			},
			want: []schema.GroupVersionResource{
				{
					Group:    "hobbyfarm.io",
					Version:  "v1",
					Resource: "virtualmachines",
				},
				{
					Group:    "hobbyfarm.io",
					Version:  "v1",
					Resource: "courses",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "cost-config-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temporary file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			// Write some content to the file
			content, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal json: %v", err)
			}
			if _, err := tempFile.Write(content); err != nil {
				t.Fatalf("Failed to write to temporary file: %v", err)
			}

			fullPath := tempFile.Name()
			DefaultConfigPath = filepath.Dir(fullPath)
			DefaultConfigName = filepath.Base(fullPath)

			assert.ElementsMatch(t, ParseConfig(), tt.want)
		})
	}
}
