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

package api

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/kubeflow/pipelines/backend/test/v2/api/constants"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
)

// ####################################################################################################################################################################
// ################################################################### SET AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	logger.Log("################### Setup before each Pipeline Upload test #####################")
	expectedPipeline = new(model.V2beta1Pipeline)
	expectedPipeline.CreatedAt = testStartTime
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Upload and Verify Pipeline Run >", Label("Positive", "E2E", S1, FullRegression), func() {

	/* Critical Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", func() {
		var pipelineDir = "valid"
		criticalPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range criticalPipelineFiles[0:1] {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				pipelineFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFile)
				logger.Log("Uploading pipeline file %s", pipelineFile)
				uploadedPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFile, &pipelineGeneratedName)
				logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
				uploadedPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
				pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFilePath)
				logger.Log("Create run for pipeline with id: '%s' and name: '%s'", uploadedPipeline.PipelineID, uploadedPipeline.DisplayName)
				uploadedPipelineRun := createPipelineRun(&uploadedPipeline.PipelineID, &uploadedPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
				logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s", uploadedPipelineRun.RunID, uploadedPipeline.PipelineID)
				utils.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING}, nil)
				logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s is now RUNNING", uploadedPipelineRun.RunID, uploadedPipeline.PipelineID)
				timeToWait := time.Duration(300)
				utils.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStateSKIPPED, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateCANCELED}, &timeToWait)
				specs := utils.ReadYamlFile(pipelineFilePath).(map[string]interface{})
				pipelineSpecs, platformSpecs := utils.DeserializeSpecs(specs)
				logger.Log("%v - %v", pipelineSpecs.PipelineInfo, platformSpecs.Platforms)
				capturePodLogsForUnsuccessfulTasks(uploadedPipelineRun.RunID)
			})
		}
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func capturePodLogsForUnsuccessfulTasks(runID string) {
	logger.Log("Fetching updated pipeline run details for run with id=%s", runID)
	runDetails := utils.GetPipelineRun(runClient, &runID)
	logger.Log("Updated pipeline run details")
	var failedTasks []string
	sort.Slice(runDetails.RunDetails.TaskDetails, func(i, j int) bool {
		return time.Time(runDetails.RunDetails.TaskDetails[i].EndTime).After(time.Time(runDetails.RunDetails.TaskDetails[j].EndTime)) // Sort Tasks by End Time in descending order
	})
	for _, task := range runDetails.RunDetails.TaskDetails {
		switch task.State {
		case run_model.V2beta1RuntimeStateSUCCEEDED:
			{
				logger.Log("SUCCEEDED - Task %s for run %s has finished successfully", task.DisplayName, task.RunID)
			}
		case run_model.V2beta1RuntimeStateRUNNING:
			{
				logger.Log("RUNNING - Task %s for Run %s is running", task.DisplayName, task.RunID)

			}
		case run_model.V2beta1RuntimeStateSKIPPED:
			{
				logger.Log("SKIPPED - Task %s for Run %s skipped", task.DisplayName, task.RunID)
			}
		case run_model.V2beta1RuntimeStateCANCELED:
			{
				logger.Log("CANCELED - Task %s for Run %s canceled", task.DisplayName, task.RunID)
			}
		default:
			{
				logger.Log("%s - Task %s for Run %s did not complete successfully", task.State, task.DisplayName, runID)
				for _, childTask := range task.ChildTasks {
					podName := childTask.PodName
					if podName != "" {
						logger.Log("Capturing pod logs for task %s, with pod name %s", task.DisplayName, podName)
						podLog := utils.ReadPodLogs(k8Client, *namespace, podName, nil, &testStartTimeUTC, podLogLimit)
						logger.Log("Pod logs captured for pod %s", task.DisplayName, podName)
						logger.Log("Attaching pod logs to the report")
						AddReportEntry(fmt.Sprintf("Failing '%s' Component Log", task.DisplayName), podLog)
						logger.Log("Attached pod logs to the report")
					}
				}
				failedTasks = append(failedTasks, task.DisplayName)
			}
		}
	}
	if len(failedTasks) > 0 {
		logger.Log("Found failed tasks: %v", failedTasks)
		Fail(fmt.Sprintf("Test failed due to failing tasks: %s", strings.Join(failedTasks, "\n")))
	}
}
