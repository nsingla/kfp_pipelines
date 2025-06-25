// Copyright 2018-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
)

func ToRunDetailsFromPipelineSpec(pipelineSpec interface{}, runID string) *run_model.V2beta1RunDetails {
	logger.Log("Converting Pipeline Spec to run details")
	parsedRunDetails := &run_model.V2beta1RunDetails{}
	pipelineSpecMap := pipelineSpec.(map[string]interface{})
	specs, ok := pipelineSpecMap["pipeline_specs"]
	var specsMap map[string]interface{}
	if !ok {
		specsMap = pipelineSpecMap
	} else {
		specsMap = specs.(map[string]interface{})
	}
	root := pipelineSpecMap["root"].(map[string]interface{})
	components := pipelineSpecMap["components"].(map[string]interface{})
	tasks := root["dag"].(map[string]interface{})["tasks"].(map[string]interface{})
	executors := specsMap["deploymentSpec"].(map[string]interface{})["executors"].(map[string]interface{})
	parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, getTaskDetailsForComponent(runID, tasks, components)...)
	parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, createTaskDetail(runID, "root", tasks, "", components))
	parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, createTaskDetail(runID, "root-driver", tasks, "", components))
	for range executors {
		parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, createTaskDetail(runID, "executor", tasks, "", make(map[string]interface{})))
	}
	return parsedRunDetails
}

func getTaskDetailsForComponent(runID string, tasks map[string]interface{}, components map[string]interface{}) []*run_model.V2beta1PipelineTaskDetail {
	var parsedTaskDetails []*run_model.V2beta1PipelineTaskDetail
	for _, task := range tasks {
		taskMap := task.(map[string]interface{})
		taskName := taskMap["taskInfo"].(map[string]interface{})["name"].(string)
		parsedTaskDetails = append(parsedTaskDetails, createTaskDetail(runID, taskName, taskMap, "", components))
		parsedTaskDetails = append(parsedTaskDetails, createTaskDetail(runID, taskName+"-driver", taskMap, "", components))
		// Process nested tasks if this is a DAG component
		componentName := taskMap["componentRef"].(map[string]interface{})["name"].(string)
		if component, compExists := components[componentName].(map[string]interface{}); compExists && component["dag"] != nil {
			nestedTasks := getTaskDetailsForComponent(runID, component["dag"].(map[string]interface{})["tasks"].(map[string]interface{}), make(map[string]interface{}))
			parsedTaskDetails = append(parsedTaskDetails, nestedTasks...)
		}
	}
	return parsedTaskDetails
}

// createTaskDetail creates a V2beta1PipelineTaskDetail from a TaskYAML
func createTaskDetail(runID string, taskName string, task map[string]interface{}, parentTaskID string, components map[string]interface{}) *run_model.V2beta1PipelineTaskDetail {
	now := time.Now().UTC()

	taskDetail := &run_model.V2beta1PipelineTaskDetail{
		RunID:        runID,
		DisplayName:  taskName,
		ParentTaskID: parentTaskID,
		CreateTime:   strfmt.DateTime(now),
		StartTime:    strfmt.DateTime(now),
		StateHistory: []*run_model.V2beta1RuntimeStatus{},
		Inputs:       make(map[string]run_model.V2beta1ArtifactList),
		Outputs:      make(map[string]run_model.V2beta1ArtifactList),
		ChildTasks:   []*run_model.PipelineTaskDetailChildTask{},
	}

	// Set executor detail if component exists
	componentRef, componentRefExists := task["componentRef"]
	if componentRefExists {
		componentRefMap := componentRef.(map[string]interface{})
		componentRefName := componentRefMap["name"]
		for _, component := range components {
			componentMap := component.(map[string]interface{})
			componentName := componentMap["name"]
			if componentName == componentRefName {
				taskDetail.ExecutorDetail = &run_model.V2beta1PipelineTaskExecutorDetail{
					MainJob:                   componentName.(string),
					PreCachingCheckJob:        "",
					FailedMainJobs:            []string{},
					FailedPreCachingCheckJobs: []string{},
				}
			}
		}
	}

	// Process inputs
	taskInputs, taskInputExists := task["inputs"]
	if taskInputExists {
		taskInputsMap := taskInputs.(map[string]interface{})
		taskInputParameters := taskInputsMap["parameters"].(map[string]interface{})
		if len(taskInputParameters) > 0 {
			for paramName, paramValue := range taskInputParameters {
				// Convert parameter value to artifact list
				artifactList := run_model.V2beta1ArtifactList{
					ArtifactIds: []string{},
				}

				// Create a simple artifact representation
				if _, ok := paramValue.(map[string]interface{}); ok {
					artifactList.ArtifactIds = append(artifactList.ArtifactIds, fmt.Sprintf("%s-%s", taskName, paramName))
				}

				taskDetail.Inputs[paramName] = artifactList
			}
		}
	}

	// Process parameter iterator for loop tasks
	if task["parameterIterator"] != nil {
		// Add loop-specific metadata
		taskDetail.ExecutorDetail = &run_model.V2beta1PipelineTaskExecutorDetail{
			MainJob:                   "loop-executor",
			PreCachingCheckJob:        "",
			FailedMainJobs:            []string{},
			FailedPreCachingCheckJobs: []string{},
		}
	}

	return taskDetail
}
