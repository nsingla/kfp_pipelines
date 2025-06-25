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
	"strings"

	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	. "github.com/kubeflow/pipelines/backend/test/v2/api/constants"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	matcher "github.com/kubeflow/pipelines/backend/test/v2/api/matcher"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ####################################################################################################################################################################
// ################################################################### CLASS VARIABLES ################################################################################
// ####################################################################################################################################################################

var createdPipelineRun *run_model.V2beta1Run
var createdPipeline *pipeline_upload_model.V2beta1Pipeline
var err error
var runName string
var runDescription string
var createdRunIds []string

// ####################################################################################################################################################################
// ################################################################### SETUP AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	runName = "API Test Run - " + randomName
	runDescription = "API Test Run"
	logger.Log("Setting up Pipeline Run Tests")
})

var _ = AfterEach(func() {
	logger.Log("Tearing down Pipeline Run Tests")
	if len(createdRunIds) > 0 {
		for _, runID := range createdRunIds {
			logger.Log("Deleting run %s", runID)
			deleteRunParams := run_params.NewRunServiceDeleteRunParams()
			deleteRunParams.RunID = runID
			err := runClient.Delete(deleteRunParams)
			if err != nil {
				logger.Log("Failed to delete run %s", runID)
			}
		}
	}
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Run >", Label("Positive", "PipelineRun"), func() {

	/* Critical Positive Scenarios of uploading a pipeline file */
	Context("Create a critical valid pipeline and verify the created run >", Label(Smoke, S1), func() {
		var pipelineDir = "valid/critical"
		criticalPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range criticalPipelineFiles {
			It(fmt.Sprintf("Create a '%s' pipeline and verify run", pipelineFile), func() {
				createPipelineAndVerifyRun(pipelineDir, pipelineFile)
			})
		}
	})

	/* Critical Positive Scenarios of uploading a pipeline file */
	Context("Create a valid pipeline and verify the created run >", Label(FullRegression, S2), func() {
		var pipelineDir = "valid"
		criticalPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range criticalPipelineFiles {
			It(fmt.Sprintf("Create a '%s' pipeline and verify run", pipelineFile), func() {
				createPipelineAndVerifyRun(pipelineDir, pipelineFile)
			})
		}
	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func createPipelineAndVerifyRun(pipelineDir string, pipelineFileName string) {
	logger.Log("Create a pipeline")
	createdPipeline, err = uploadPipeline(pipelineDir, pipelineFileName, &pipelineGeneratedName)
	Expect(err).NotTo(HaveOccurred())
	createdPipelines = append(createdPipelines, createdPipeline)
	createPipelineVersions, _, _, err := utils.ListPipelineVersions(pipelineClient, createdPipeline.PipelineID)
	Expect(err).NotTo(HaveOccurred())
	createdPipelineRun = createPipelineRunFromPipeline(&createdPipeline.PipelineID, &createPipelineVersions[0].PipelineVersionID, nil)
	createdRunIds = append(createdRunIds, createdPipelineRun.RunID)
	expectedPipelineRun := createExpectedPipelineRun(&createdPipeline.PipelineID, &createPipelineVersions[0].PipelineVersionID, nil, false)
	matcher.MatchPipelineRuns(createdPipelineRun, expectedPipelineRun)
	createdPipelineRunFromDB, err := runClient.Get(&run_params.RunServiceGetRunParams{
		RunID: createdPipelineRun.RunID,
	})
	Expect(err).NotTo(HaveOccurred())

	// Making the fields that can be different but we don't care about equal to stabilize tests
	matcher.MatchPipelineRuns(createdPipelineRun, createdPipelineRunFromDB)
}

func createPipelineRunFromPipeline(pipelineID *string, pipelineVersionID *string, experimentID *string) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", pipelineID, pipelineVersionID)
	createRunRequest := &run_params.RunServiceCreateRunParams{Body: createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID)}
	createdRun, err := runClient.Create(createRunRequest)
	Expect(err).NotTo(HaveOccurred())
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

func createPipelineRunPayload(pipelineID *string, pipelineVersionID *string, experimentID *string) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run body")
	return &run_model.V2beta1Run{
		DisplayName:    runName,
		Description:    runDescription,
		ExperimentID:   utils.ParsePointersToString(experimentID),
		ServiceAccount: utils.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        utils.ParsePointersToString(pipelineID),
			PipelineVersionID: utils.ParsePointersToString(pipelineVersionID),
		},
	}
}

func createExpectedPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, archived bool) *run_model.V2beta1Run {
	expectedRun := createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID)
	if !archived {
		expectedRun.StorageState = run_model.V2beta1RunStorageStateAVAILABLE
	} else {
		expectedRun.StorageState = run_model.V2beta1RunStorageStateARCHIVED
	}
	if experimentID == nil {
		logger.Log("Fetch default experiment's experimentId")
		pageSize := int32(1000)
		experminents, expError := experimentClient.ListAll(&experiment_params.ExperimentServiceListExperimentsParams{
			Namespace: namespace,
			PageSize:  &pageSize,
		}, 1000)
		Expect(expError).NotTo(HaveOccurred())
		for _, experiment := range experminents {
			if strings.ToLower(experiment.DisplayName) == "default" {
				expectedRun.ExperimentID = experiment.ExperimentID
			}
		}
	}
	return expectedRun
}

// DO NOT DELETE - When we have the logic to create pending tasks without AWC, we will use the following code
// func waitForTasksAndGetRunDetails(runID string, numberOfExpectedTasks int) *run_model.V2beta1Run {
//	createdPipelineRunFromDB, _ := runClient.Get(&run_params.RunServiceGetRunParams{
//		RunID: runID,
//	})
//	timeout := time.After(30 * time.Second)
//	for createdPipelineRunFromDB.RunDetails == nil {
//		time.Sleep(1 * time.Second)
//		createdPipelineRunFromDB, _ = runClient.Get(&run_params.RunServiceGetRunParams{
//			RunID: runID,
//		})
//		select {
//		case <-timeout:
//			Fail("Timeout waiting for run details to be available for runId=" + runID)
//		default:
//			if createdPipelineRunFromDB.RunDetails != nil {
//				if len(createdPipelineRunFromDB.RunDetails.TaskDetails) < numberOfExpectedTasks {
//					logger.Log("Not all tasks for the run %s have been generated, tasks=%d/%d", runID, len(createdPipelineRunFromDB.RunDetails.TaskDetails), numberOfExpectedTasks)
//					createdPipelineRunFromDB.RunDetails = nil
//					continue
//				} else {
//					logger.Log("All %d/%d tasks for the run %s have been generated", len(createdPipelineRunFromDB.RunDetails.TaskDetails), numberOfExpectedTasks, runID)
//				}
//			}
//		}
//	}
//	return createdPipelineRunFromDB
//}

// DO NOT DELETE - When we have the logic to create pending tasks without AWC, we will use the following code
// func validatePipelineRunDetails(inputPipelineSpec interface{}, runID string) {
//	expectedPipelineRunDetails := utils.ToRunDetailsFromPipelineSpec(inputPipelineSpec, runID)
//	createdPipelineRunFromDB := waitForTasksAndGetRunDetails(runID, len(expectedPipelineRunDetails.TaskDetails))
//	matcher.MatchPipelineRunDetails(createdPipelineRunFromDB.RunDetails, expectedPipelineRunDetails)
//}
