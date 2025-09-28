package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestContainerComponentInputs(t *testing.T) {
	// Setup test environment
	testSetup := NewTestSetup(t)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	assert.NotNil(t, run)

	opts := Options{
		PipelineName: "test-pipeline",
		Run:          run,
		Component: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{},
				},
			},
		},
		ParentTask:     nil,
		ParentTaskID:   "",
		DriverAPI:      testSetup.DriverAPI,
		IterationIndex: -1,
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"string_input": structpb.NewStringValue("test-input1"),
				"number_input": structpb.NewNumberValue(42.5),
				"bool_input":   structpb.NewBoolValue(true),
				"null_input":   structpb.NewNullValue(),
				"list_input": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
					structpb.NewStringValue("value1"),
					structpb.NewNumberValue(42),
					structpb.NewBoolValue(true),
				}}),
				"map_input": structpb.NewStructValue(&structpb.Struct{
					Fields: map[string]*structpb.Value{
						"key1": structpb.NewStringValue("value1"),
						"key2": structpb.NewNumberValue(42),
						"key3": structpb.NewListValue(&structpb.ListValue{
							Values: []*structpb.Value{
								structpb.NewStringValue("nested1"),
								structpb.NewStringValue("nested2"),
							},
						}),
					},
				}),
			},
		},
		Namespace:                "test-namespace",
		Task:                     &pipelinespec.PipelineTaskSpec{},
		Container:                nil,
		KubernetesExecutorConfig: &kubernetesplatform.KubernetesExecutorConfig{},
		PipelineLogLevel:         "1",
		PublishLogs:              "false",
		CacheDisabled:            false,
		DriverType:               "CONTAINER",
		TaskName:                 "",
		PodName:                  "system-dag-driver",
		PodUID:                   "some-uid",
	}

	// Execute RootDAG
	execution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)

}
