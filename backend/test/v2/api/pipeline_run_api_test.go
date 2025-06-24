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
	"strings"
	"time"

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
	"gopkg.in/yaml.v2"
)

// ####################################################################################################################################################################
// ################################################################### CLASS VARIABLES ################################################################################
// ####################################################################################################################################################################

var createdPipelineRun *run_model.V2beta1Run
var createdPipeline *pipeline_upload_model.V2beta1Pipeline
var err error
var runName string
var runDescription string

// ####################################################################################################################################################################
// ################################################################### SETUP AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	runName = "API Test Run - " + randomName
	runDescription = "API Test Run"
	logger.Log("Setting up Pipeline Run Tests")
	logger.Log("Create a hello-world pipeline")
	var pipelineDir = "valid"
	createdPipeline, err = uploadPipeline(pipelineDir, "parameters_simple.yaml", &pipelineGeneratedName)
	Expect(err).NotTo(HaveOccurred())
	createdPipelines = append(createdPipelines, createdPipeline)
})

var _ = AfterEach(func() {
	logger.Log("Tearing down Pipeline Run Tests")
	logger.Log("Deleting created Pipeline Runs")
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Run >", Label("Positive", "PipelineRun", S1), func() {

	/* Critical Positive Scenarios of uploading a pipeline file */
	Context("Create a pipeline run and verify created argo workflows >", Label(Smoke), func() {

		It(fmt.Sprintf("Create a run for '%s' pipeline and verify the argo workflow", helloWorldPipelineFileName), func() {
			createPipelineVersions, _, _, err := utils.ListPipelineVersions(pipelineClient, createdPipeline.PipelineID)
			Expect(err).NotTo(HaveOccurred())
			createdPipelineRun = createPipelineRunFromPipeline(&createdPipeline.PipelineID, &createPipelineVersions[0].PipelineVersionID, nil)
			expectedPipelineRun := createExpectedPipelineRun(&createdPipeline.PipelineID, &createPipelineVersions[0].PipelineVersionID, nil, false)
			matcher.MatchPipelineRunShallow(createdPipelineRun, expectedPipelineRun)
			createdPipelineRunFromDB, err := runClient.Get(&run_params.RunServiceGetRunParams{
				RunID: createdPipelineRun.RunID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipelineRunFromDB).To(Equal(createdPipelineRun))
			expectedPipelineRunDetails := utils.ToRunDetailsFromPipelineSpec(createPipelineVersions[0].PipelineSpec, createdPipelineRun.RunID)
			fmt.Println("Expected Pipeline Run Details:")
			yamlString, _ := yaml.Marshal(&expectedPipelineRunDetails)
			fmt.Println("Waiting for run details to be available")
			timeout := time.After(30 * time.Second)
			for createdPipelineRunFromDB.RunDetails == nil {

				time.Sleep(1 * time.Second)
				createdPipelineRunFromDB, _ = runClient.Get(&run_params.RunServiceGetRunParams{
					RunID: createdPipelineRun.RunID,
				})
				select {
				case <-timeout:
					Fail("Timeout waiting for run details to be available for runId=" + createdPipelineRun.RunID)
				default:
					if createdPipelineRunFromDB.RunDetails != nil {
						if len(createdPipelineRunFromDB.RunDetails.TaskDetails) < len(expectedPipelineRunDetails.TaskDetails) {
							logger.Log("Not all tasks for the run %s have been generated, tasks=%d/%d", createdPipelineRun.RunID, len(createdPipelineRunFromDB.RunDetails.TaskDetails), len(expectedPipelineRunDetails.TaskDetails))
							createdPipelineRunFromDB.RunDetails = nil
							continue
						} else {
							logger.Log("All %d/%d tasks for the run %s have been generated", len(createdPipelineRunFromDB.RunDetails.TaskDetails), len(expectedPipelineRunDetails.TaskDetails), createdPipelineRun.RunID)
						}
					}
				}
			}

			fmt.Println(string(yamlString))
			yamlString, _ = yaml.Marshal(&createdPipelineRunFromDB)
			fmt.Println("Created Pipeline Run Details:")
			fmt.Println(string(yamlString))
			matcher.MatchPipelineRunDetails(createdPipelineRunFromDB.RunDetails, expectedPipelineRunDetails)
		})
	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

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
	expectedRun := createPipelineRunPayload(&createdPipeline.PipelineID, pipelineVersionID, experimentID)
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
