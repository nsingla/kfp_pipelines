package common

import (
	"fmt"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/types/known/structpb"
)

// Options contain driver options
type Options struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	Run *apiv2beta1.Run
	// required, Component spec
	Component *pipelinespec.ComponentSpec
	// required
	ParentTask *apiv2beta1.PipelineTaskDetail
	DriverAPI  DriverAPI

	// optional, iteration index. -1 means not an iteration.
	IterationIndex int

	// optional, required only by root DAG driver
	RuntimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	Namespace     string

	// optional, required by non-root drivers
	Task *pipelinespec.PipelineTaskSpec

	// optional, required only by container driver
	Container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec

	// optional, allows to specify kubernetes-specific executor config
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig

	// optional, required only if the {{$.pipeline_job_resource_name}} placeholder is used or the run uses a workspace
	RunName string
	// optional, required only if the {{$.pipeline_job_name}} placeholder is used
	RunDisplayName   string
	PipelineLogLevel string
	PublishLogs      string
	CacheDisabled    bool
	DriverType       string
	TaskName         string // the original name of the task, used for input resolution
	PodName          string
	PodUID           string
}

// Identifying information used for error messages
func (o Options) Info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.Run.GetRunId())
	if o.Task.GetTaskInfo().GetName() != "" {
		msg = msg + fmt.Sprintf(", taskDisplayName=%q", o.Task.GetTaskInfo().GetName())
	}
	if o.TaskName != "" {
		msg = msg + fmt.Sprintf(", taskName=%q", o.TaskName)
	}
	if o.Task.GetComponentRef().GetName() != "" {
		msg = msg + fmt.Sprintf(", component=%q", o.Task.GetComponentRef().GetName())
	}
	if o.ParentTask != nil {
		msg = msg + fmt.Sprintf(", dagExecutionID=%v", o.ParentTask.GetParentTaskId())
	}
	if o.IterationIndex >= 0 {
		msg = msg + fmt.Sprintf(", iterationIndex=%v", o.IterationIndex)
	}
	if o.RuntimeConfig != nil {
		msg = msg + ", runtimeConfig" // this only means runtimeConfig is not empty
	}
	if o.Component.GetImplementation() != nil {
		msg = msg + ", componentSpec" // this only means componentSpec is not empty
	}
	if o.KubernetesExecutorConfig != nil {
		msg = msg + ", KubernetesExecutorConfig" // this only means KubernetesExecutorConfig is not empty
	}
	return msg
}

const pipelineChannelPrefix = "pipelinechannel--"

// ParameterNameToIOFields parses a pipeline channel name into its key,
// It also extracts producerTaskName, and producerKey if the channel has
// a Task producer. Note that a channel does not always have a task
// producer, such as in the case for raw parameterIterator.
func ParameterNameToIOFields(name string,
) (key string, producerTaskName string, producerKey string, err error) {
	if !strings.HasPrefix(name, pipelineChannelPrefix) {
		return name, "", "", nil
	}

	// Remove prefix
	nameWithoutPrefix := strings.TrimPrefix(name, pipelineChannelPrefix)

	// Split remaining string by last dash to separate producer task name and key
	parts := strings.Split(nameWithoutPrefix, "-")
	if len(parts) < 2 {
		return "", "", "",
			fmt.Errorf("invalid pipeline channel name: %s", name)
	}

	// This is a parameterIterator Pipeline Channel, i.e. a LoopArgument
	// There are two cases here
	if strings.HasPrefix(nameWithoutPrefix, "loop-item") {
		// Case 1. It is a Raw parameterIterator in which case
		// its format is "pipelinechannel--loop-item-param-{code}"
		// The declared LoopArgument parameter, for cases like
		// nested parellelFor loops.

		if len(parts) != 4 || parts[2] != "param" {
			return "", "", "",
				fmt.Errorf("invalid pipeline channel name for raw LoopArgument: %s", name)
		}
		key = strings.Join(parts[:4], "-")
		return key, "", "", nil

	} else if strings.HasSuffix(nameWithoutPrefix, "loop-item") {
		// Case 2. If it is an inputParameter parameterIterator, (i.e., it has a producer)
		// Then its format is "pipelinechannel--{producerTaskName}-{producerKey}-loop-item"'
		// Note that {producerTaskName} can have dashes, but the last dash separates the
		// Task Name from the Key.

		if len(parts) < 4 {
			return "", "", "",
				fmt.Errorf("invalid pipeline channel name for inputParameter LoopArgument: %s", name)
		}
		producerKey = parts[len(parts)-3]
		taskName := strings.Join(parts[:len(parts)-3], "-")
		return "", taskName, producerKey, nil
	}

	key = parts[len(parts)-1]
	producerTaskName = strings.Join(parts[:len(parts)-1], "-")
	producerKey = key

	return key, producerTaskName, producerKey, nil
}

func IsPipelineChannel(name string) bool {
	return strings.HasPrefix(name, "pipelinechannel--")
}

func IsLoopArgument(name string) bool {
	// Remove prefix
	nameWithoutPrefix := strings.TrimPrefix(name, pipelineChannelPrefix)
	return strings.HasSuffix(nameWithoutPrefix, "loop-item") || strings.HasPrefix(nameWithoutPrefix, "loop-item")
}

func IsRuntimeIterationTask(task *apiv2beta1.PipelineTaskDetail) bool {
	return task.Type == apiv2beta1.PipelineTaskDetail_RUNTIME && task.TypeAttributes != nil && task.TypeAttributes.IterationIndex != nil
}

func ConvertArtifactsToArtifactList(artifacts []*apiv2beta1.Artifact) (*pipelinespec.ArtifactList, error) {
	var runtimeArtifacts []*pipelinespec.RuntimeArtifact
	for _, artifact := range artifacts {
		runtimeArtifact, err := ConvertArtifactToRuntimeArtifact(artifact)
		if err != nil {
			return nil, err
		}
		runtimeArtifacts = append(runtimeArtifacts, runtimeArtifact)
	}
	return &pipelinespec.ArtifactList{
		Artifacts: runtimeArtifacts,
	}, nil
}

func ConvertArtifactToRuntimeArtifact(
	artifact *apiv2beta1.Artifact,
) (*pipelinespec.RuntimeArtifact, error) {
	if artifact.GetName() == "" && artifact.GetUri() == "" {
		return nil, fmt.Errorf("artifact name or uri cannot be empty")
	}
	runtimeArtifact := &pipelinespec.RuntimeArtifact{
		Name:       artifact.GetName(),
		ArtifactId: artifact.GetArtifactId(),
		Type: &pipelinespec.ArtifactTypeSchema{
			Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
				SchemaTitle: artifact.Type.String(),
			},
		},
	}
	if artifact.GetUri() != "" {
		runtimeArtifact.Uri = artifact.GetUri()
	}
	if artifact.GetMetadata() != nil {
		runtimeArtifact.Metadata = &structpb.Struct{
			Fields: artifact.GetMetadata(),
		}
	}
	return runtimeArtifact, nil
}
