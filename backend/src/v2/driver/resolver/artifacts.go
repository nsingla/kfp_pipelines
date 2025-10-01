package resolver

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/structpb"
)

func resolveInputArtifact(
	ctx context.Context,
	opts common.Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*pipelinespec.ArtifactList, error) {
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		return resolveArtifactComponentInputParameter(opts, artifactSpec, inputArtifacts)
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		artifacts, err := resolveUpstreamArtifacts(ctx, opts, artifactSpec)
		if err != nil {
			return nil, err
		}
		return artifacts, nil
	default:
		return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

func resolveArtifactComponentInputParameter(
	opts common.Options,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact) (*pipelinespec.ArtifactList, error) {
	paramName := artifactSpec.GetComponentInputArtifact()
	if paramName == "" {
		return nil, fmt.Errorf("empty component input")
	}
	isPipelineChannel := common.IsPipelineChannel(paramName)

	for _, param := range inputParams {
		generateName, err := common.IOFieldsToPipelineChannelName(param.GetParameterName(), param.GetProducer(), isPipelineChannel)
		if err != nil {
			return nil, err
		}
		if paramName == generateName {
			artifacts := param.GetArtifacts()
			if common.IsLoopArgument(paramName) {
				if len(artifacts) == 0 {
					return nil, fmt.Errorf("loop argument %s must have at least one value", paramName)
				}
				if len(artifacts) <= opts.IterationIndex+1 {
					return nil, fmt.Errorf(
						"loop argument %s has only %d values,"+
							" but index %d is requested, which is out "+
							"of bounds", paramName, len(artifacts), opts.IterationIndex)
				}
				runtimeArtifact, err := convertArtifactToRuntimeArtifact(artifacts[opts.IterationIndex])
				if err != nil {
					return nil, err
				}
				return &pipelinespec.ArtifactList{
					Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
				}, nil
			}
			artifactList, err := convertArtifactsToArtifactList(artifacts)
			if err != nil {
				return nil, err
			}
			return artifactList, nil
		}
	}
	return nil, fmt.Errorf("failed to find input param %s", paramName)

}

func resolveUpstreamArtifacts(ctx context.Context,
	opts common.Options,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec) (*pipelinespec.ArtifactList, error) {

	tasks, err := getSubTasks(opts.ParentTask, opts.Run.Tasks, nil)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return nil, fmt.Errorf("failed to get sub tasks for task %s", opts.ParentTask.Name)
	}
	producerTaskAmbiguousName := artifactSpec.GetTaskOutputArtifact().GetProducerTask()
	if producerTaskAmbiguousName == "" {
		return nil, fmt.Errorf("producerTask task cannot be empty")
	}
	producerTaskUniqueName := getTaskNameWithTaskID(producerTaskAmbiguousName, opts.ParentTask.GetTaskId())
	producerTaskUniqueName = InferIndexedTaskName(producerTaskUniqueName, opts.ParentTask)
	producerTask := tasks[producerTaskUniqueName]
	if producerTask == nil {
		return nil, fmt.Errorf("producerTask task %s not found", producerTaskUniqueName)
	}

	currentTask := producerTask
	outputArtifactKey := artifactSpec.GetTaskOutputArtifact().GetOutputArtifactKey()

	if producerTask.GetType() == apiv2beta1.PipelineTaskDetail_RUNTIME {
		outputArtifacts := currentTask.GetOutputs().GetArtifacts()
		artifacts, err := findArtifactByProducerKeyInList(outputArtifactKey, outputArtifacts)
		if err != nil {
			return nil, err
		}
		if artifacts == nil {
			return nil, fmt.Errorf("output artifact %s not found", outputArtifactKey)
		}
		artifactList, err := convertArtifactsToArtifactList(artifacts)
		if err != nil {
			return nil, err
		}
		return artifactList, nil
	}
	return &pipelinespec.ArtifactList{}, nil
}

// generateUniqueTaskName generates a unique task name for a given task.
func generateUniqueTaskName(task, parentTask *apiv2beta1.PipelineTaskDetail) (string, error) {
	if task == nil || task.Name == "" || parentTask == nil {
		return "", fmt.Errorf("parenttask and task can't be nil and task name cannot be empty")
	}
	taskName := getTaskNameWithTaskID(task.Name, parentTask.TaskId)
	if task.Type == apiv2beta1.PipelineTaskDetail_LOOP_ITERATION {
		if task.TypeAttributes == nil || task.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getParallelForTaskName(taskName, *task.TypeAttributes.IterationIndex)
	} else if parentTask.Type == apiv2beta1.PipelineTaskDetail_LOOP_ITERATION {
		if parentTask.TypeAttributes == nil || parentTask.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getParallelForTaskName(taskName, *parentTask.TypeAttributes.IterationIndex)
	}
	return taskName, nil
}

func getChildTasks(
	tasks []*apiv2beta1.PipelineTaskDetail,
	parentTask *apiv2beta1.PipelineTaskDetail,
) (map[string]*apiv2beta1.PipelineTaskDetail, error) {
	if parentTask == nil {
		return nil, fmt.Errorf("parent task cannot be nil")
	}
	var taskMap = make(map[string]*apiv2beta1.PipelineTaskDetail)
	for _, task := range tasks {
		if task.GetParentTaskId() == parentTask.GetTaskId() {
			taskName, err := generateUniqueTaskName(task, parentTask)
			if err != nil {
				return nil, err
			}
			if taskName == "" {
				return nil, fmt.Errorf("task name cannot be empty")
			}
			taskMap[taskName] = task
		}
	}
	return taskMap, nil
}

func getSubTasks(
	currentTask *apiv2beta1.PipelineTaskDetail,
	allRuntasks []*apiv2beta1.PipelineTaskDetail,
	flattenedTasks map[string]*apiv2beta1.PipelineTaskDetail,
) (map[string]*apiv2beta1.PipelineTaskDetail, error) {
	if flattenedTasks == nil {
		flattenedTasks = make(map[string]*apiv2beta1.PipelineTaskDetail)
	}
	taskChildren, err := getChildTasks(allRuntasks, currentTask)
	if err != nil {
		return nil, fmt.Errorf("failed to get child tasks for task %s: %w", currentTask.Name, err)
	}
	for taskName, task := range taskChildren {
		flattenedTasks[taskName] = task
	}
	for _, task := range taskChildren {
		if task.Type != apiv2beta1.PipelineTaskDetail_RUNTIME {
			flattenedTasks, err = getSubTasks(task, allRuntasks, flattenedTasks)
			if err != nil {
				return nil, err
			}
		}
	}
	return flattenedTasks, nil
}

func getParallelForTaskName(taskName string, iterationIndex int64) string {
	return fmt.Sprintf("%s_idx_%d", taskName, iterationIndex)
}

func getTaskNameWithTaskID(taskName, taskID string) string {
	return fmt.Sprintf("%s_%s", taskName, taskID)
}

func InferIndexedTaskName(producerTaskName string, task *apiv2beta1.PipelineTaskDetail) string {
	// Check if the Task in question is a parallelFor iteration Task. If it is, we need to
	// update the producerTaskName so the downstream task resolves the appropriate index.
	if task.GetType() == apiv2beta1.PipelineTaskDetail_LOOP_ITERATION {
		taskIterationIndex := task.GetTypeAttributes().GetIterationIndex()
		producerTaskName = getParallelForTaskName(producerTaskName, taskIterationIndex)
	}
	return producerTaskName
}

func findArtifactByProducerKeyInList(
	producerKey string,
	artifactsIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) ([]*apiv2beta1.Artifact, error) {
	for _, artifactIO := range artifactsIO {
		if artifactIO.GetProducer().Key == producerKey {
			return artifactIO.Artifacts, nil
		}
	}
	return nil, fmt.Errorf("artifact with producer key %s not found", producerKey)
}

func convertArtifactsToArtifactList(artifacts []*apiv2beta1.Artifact) (*pipelinespec.ArtifactList, error) {
	var runtimeArtifacts []*pipelinespec.RuntimeArtifact
	for _, artifact := range artifacts {
		runtimeArtifact, err := convertArtifactToRuntimeArtifact(artifact)
		if err != nil {
			return nil, err
		}
		runtimeArtifacts = append(runtimeArtifacts, runtimeArtifact)
	}
	return &pipelinespec.ArtifactList{
		Artifacts: runtimeArtifacts,
	}, nil
}

func convertArtifactToRuntimeArtifact(
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
