package resolver

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

func resolveInputArtifact(
	ctx context.Context,
	opts common.Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) ([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		artifactIO, err := resolveArtifactComponentInputParameter(opts, artifactSpec, inputArtifacts)
		if err != nil {
			return nil, artifactError(err)
		}
		return []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{artifactIO}, nil
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
	return nil, fmt.Errorf("failed to find input param %s", key)

}

func resolveUpstreamArtifacts(ctx context.Context,
	opts common.Options,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
) ([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {

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
		producerTaskUniqueName = getParallelForTaskName(producerTaskUniqueName, int64(opts.IterationIndex))
	}
	producerTask := tasks[producerTaskUniqueName]
	if producerTask == nil {
		return nil, fmt.Errorf("producerTask task %s not found", producerTaskUniqueName)
	}

	currentTask := producerTask
	outputArtifactKey := artifactSpec.GetTaskOutputArtifact().GetOutputArtifactKey()

	var collectArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
	switch producerTask.GetType() {
	case apiv2beta1.PipelineTaskDetail_RUNTIME:
		outputArtifacts := currentTask.GetOutputs().GetArtifacts()
		artifactIO, err := findArtifactByProducerKeyInList(outputArtifactKey, outputArtifacts)
		if err != nil {
			return nil, err
		}
		if artifactIO == nil {
			return nil, fmt.Errorf("output artifact %s not found", outputArtifactKey)
		}
		collectArtifacts = append(collectArtifacts, artifactIO)
		return collectArtifacts, nil
	case apiv2beta1.PipelineTaskDetail_LOOP:
		if !common.IsPipelineChannel(outputArtifactKey) {
			return nil, fmt.Errorf("loop output artifact %s must be a pipeline channel", outputArtifactKey)
		}

		for _, ioArtifact := range currentTask.GetOutputs().GetArtifacts() {
			if ioArtifact.GetType() == apiv2beta1.IOType_ITERATOR_OUTPUT &&
				ioArtifact.GetArtifactKey() == outputArtifactKey &&
				ioArtifact.GetProducer().TaskName == currentTask.Name {
				collectArtifacts = append(collectArtifacts, ioArtifact)
			}
		}
		if len(collectArtifacts) == 0 {
			return nil, fmt.Errorf("loop output artifacts for key %s not found", outputArtifactKey)
		}
		return collectArtifacts, nil
	case apiv2beta1.PipelineTaskDetail_DAG:
		return nil, fmt.Errorf("task type %s not implemented yet", producerTask.GetType())

	default:
		return nil, fmt.Errorf("task type %s not implemented yet", producerTask.GetType())
	}
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
		taskName = getParallelForTaskName(taskName, *task.TypeAttributes.IterationIndex)
	} else if common.IsRuntimeIterationTask(parentTask) {
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
