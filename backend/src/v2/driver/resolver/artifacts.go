package resolver

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

func resolveArtifacts(ctx context.Context, opts common.Options) ([]ArtifactMetadata, *int, error) {
	var artifacts []ArtifactMetadata

	for name, artifactSpec := range opts.Task.GetInputs().GetArtifacts() {
		v, ioType, err := resolveInputArtifact(ctx, opts, name, artifactSpec, opts.ParentTask.Inputs.GetArtifacts())
		if err != nil {
			return nil, nil, err
		}

		am := ArtifactMetadata{
			Key:               name,
			InputArtifactSpec: artifactSpec,
			ArtifactIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
				Artifacts: v.Artifacts,
				Type:      ioType,
				Producer:  &apiv2beta1.IOProducer{TaskName: opts.ParentTask.Name},
			},
		}
		if opts.IterationIndex >= 0 {
			am.ArtifactIO.Producer.Iteration = util.Int64Pointer(int64(opts.IterationIndex))
		}
		artifacts = append(artifacts, am)
	}

	var iterationCount *int

	if opts.Task.GetArtifactIterator() != nil {
		iterator := opts.Task.GetArtifactIterator()
		// This should be the key input into the for loop task
		iteratorInputDefinitionKey := iterator.GetItemInput()
		// Used to look up the Artifact from the resolved list
		// The key here should map to a ArtifactMetadata.Key that
		// was resolved in the prior loop.
		sourceInputArtifactKey := iterator.GetItems().GetInputArtifact()
		artifactIO, err := findArtifactByIOKey(sourceInputArtifactKey, artifacts)
		if err != nil {
			return nil, nil, err
		}
		artifacts = append(artifacts, ArtifactMetadata{
			Key: iteratorInputDefinitionKey,
			ArtifactIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
				Artifacts: artifactIO.Artifacts,
				Type:      apiv2beta1.IOType_ITERATOR_INPUT,
				Producer:  artifactIO.Producer,
			},
			ArtifactIterator: iterator,
		})
		count := len(artifactIO.Artifacts)
		iterationCount = &count
	}

	return artifacts, iterationCount, nil
}

func resolveInputArtifact(
	ctx context.Context,
	opts common.Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, apiv2beta1.IOType, error) {
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		artifactIO, err := resolveArtifactComponentInputParameter(opts, artifactSpec, inputArtifacts)
		if err != nil {
			return nil, apiv2beta1.IOType_COMPONENT_INPUT, artifactError(err)
		}
		return artifactIO, apiv2beta1.IOType_COMPONENT_INPUT, nil
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		artifact, err := resolveUpstreamArtifacts(ctx, opts, artifactSpec)
		if err != nil {
			return nil, apiv2beta1.IOType_TASK_OUTPUT_INPUT, err
		}
		return artifact, apiv2beta1.IOType_TASK_OUTPUT_INPUT, nil
	default:
		return nil, apiv2beta1.IOType_UNSPECIFIED, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

func resolveArtifactComponentInputParameter(
	opts common.Options,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifactsIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	key := artifactSpec.GetComponentInputArtifact()
	if key == "" {
		return nil, fmt.Errorf("empty component input")
	}

	for _, artifactIO := range inputArtifactsIO {
		ioKey := artifactIO.GetArtifactKey()
		if key == ioKey {
			if !common.IsLoopArgument(key) {
				return artifactIO, nil
			}
			if artifactIO.Producer != nil && artifactIO.Producer.Iteration != nil && *artifactIO.Producer.Iteration == int64(opts.IterationIndex) {
				return artifactIO, nil
			}
			return artifactIO, nil
		}
	}
	return nil, fmt.Errorf("failed to find input artifact %s", key)

}

func resolveUpstreamArtifacts(ctx context.Context,
	opts common.Options,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
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
	if opts.IterationIndex >= 0 {
		producerTaskUniqueName = getTaskNameWithIterationIndex(producerTaskUniqueName, int64(opts.IterationIndex))
	}
	// producerTask is the specific task guaranteed to have the output artifact
	// producerTaskUniqueName may look something like "task_name_a_dag_id_1_idx_0"
	producerTask := tasks[producerTaskUniqueName]
	if producerTask == nil {
		return nil, fmt.Errorf("producerTask task %s not found", producerTaskUniqueName)
	}
	outputArtifactKey := artifactSpec.GetTaskOutputArtifact().GetOutputArtifactKey()

	outputArtifacts := producerTask.GetOutputs().GetArtifacts()
	artifactIO, err := findArtifactByProducerKeyInList(outputArtifactKey, outputArtifacts)
	if err != nil {
		return nil, err
	}
	if artifactIO == nil {
		return nil, fmt.Errorf("output artifact %s not found", outputArtifactKey)
	}
	return artifactIO, nil
}

// generateUniqueTaskName generates a unique task name for a given task.
func generateUniqueTaskName(task, parentTask *apiv2beta1.PipelineTaskDetail) (string, error) {
	if task == nil || task.Name == "" || parentTask == nil {
		return "", fmt.Errorf("parenttask and task can't be nil and task name cannot be empty")
	}
	taskName := getTaskNameWithTaskID(task.Name, parentTask.TaskId)
	if common.IsRuntimeIterationTask(task) {
		if task.TypeAttributes == nil || task.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getTaskNameWithIterationIndex(taskName, *task.TypeAttributes.IterationIndex)
	} else if common.IsRuntimeIterationTask(parentTask) {
		if parentTask.TypeAttributes == nil || parentTask.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getTaskNameWithIterationIndex(taskName, *parentTask.TypeAttributes.IterationIndex)
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

// getSubTasks creates a map of all subtasks under currentTask.
// The keys of the map are formed by concatenating the task name with the task id.
// If the task is a runtime iteration task, then the key is formed by concatenating
// the task name with the iteration index.
// So you may end up with a map of the form:
//
//	{
//	  "task_name_a_dag_id_1_idx_0": {
//	    ...
//	  },
//	  "task_name_a_dag_id_1_idx_1": {
//	    ...
//	  },
//	  "task_name_b_dag_id_2": {
//	    ...
//	  },
//	},
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

func getTaskNameWithIterationIndex(taskName string, iterationIndex int64) string {
	return fmt.Sprintf("%s_idx_%d", taskName, iterationIndex)
}

func getTaskNameWithTaskID(taskName, taskID string) string {
	return fmt.Sprintf("%s_%s", taskName, taskID)
}

func findArtifactByProducerKeyInList(
	producerKey string,
	artifactsIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	for _, artifactIO := range artifactsIO {
		if artifactIO.GetArtifactKey() == producerKey {
			return artifactIO, nil
		}
	}
	return nil, fmt.Errorf("artifact with producer key %s not found", producerKey)
}
