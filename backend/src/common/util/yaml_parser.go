package util

import (
	"fmt"
	"os"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

// LoadPipelineSpecFromYAML loads a pipeline spec from a YAML file path
func LoadPipelineSpecFromYAML(path string) (*pipelinespec.PipelineSpec, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	// Convert YAML -> JSON, then use protojson to honor proto field names
	jsonBytes, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	var spec pipelinespec.PipelineSpec
	um := protojson.UnmarshalOptions{
		DiscardUnknown: true, // tolerate extra fields
	}
	if err := um.Unmarshal(jsonBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PipelineSpec (protojson): %w", err)
	}
	return &spec, nil
}
