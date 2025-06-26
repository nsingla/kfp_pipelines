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
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"sigs.k8s.io/yaml"

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
var experimentName string
var runName string
var runDescription string

// ####################################################################################################################################################################
// ################################################################### SETUP AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	logger.Log("Setting up Pipeline Run Tests")
	runName = "API Test Run - " + randomName
	runDescription = "API Test Run"
	experimentName = "API Test Experiment - " + randomName
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Run >", Label("Positive", "PipelineRun", FullRegression, S1), func() {

	type TestParams struct {
		pipelineDirectory    string
		pipelineCacheEnabled bool
	}

	testParams := []TestParams{
		{pipelineDirectory: "valid", pipelineCacheEnabled: true},
		{pipelineDirectory: "valid", pipelineCacheEnabled: false},
	}

	/* Critical pipelines */
	Context("Create a valid pipeline and verify the created run >", func() {
		for _, param := range testParams {
			pipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, param.pipelineDirectory))
			for _, pipelineFile := range pipelineFiles {
				It(fmt.Sprintf("Create a '%s' pipeline with cacheEnabled=%t and verify run", pipelineFile, param.pipelineCacheEnabled), func() {
					configuredPipelineSpecFile := configureCacheSettingAndGetPipelineFile(param.pipelineDirectory, pipelineFile, param.pipelineCacheEnabled)
					createdPipeline := uploadAPipeline(configuredPipelineSpecFile, &pipelineGeneratedName)
					pipelineRuntimeInputs := getPipelineRunTimeInputs(configuredPipelineSpecFile)
					createPipelineRunAndVerify(&createdPipeline.PipelineID, nil, pipelineRuntimeInputs)
				})
			}
			It(fmt.Sprintf("Create a '%s' pipeline with cacheEnabled=%t, create an experiement and verify run with associated experiment", pipelineFiles[0], param.pipelineCacheEnabled), Label(Smoke), func() {
				pipelineFile := pipelineFiles[0]
				configuredPipelineSpecFile := configureCacheSettingAndGetPipelineFile(param.pipelineDirectory, pipelineFile, param.pipelineCacheEnabled)
				createdExperiment := createExperiment(experimentName)
				createdPipeline := uploadAPipeline(configuredPipelineSpecFile, &pipelineGeneratedName)
				pipelineRuntimeInputs := getPipelineRunTimeInputs(configuredPipelineSpecFile)
				createPipelineRunAndVerify(&createdPipeline.PipelineID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
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

func configureCacheSettingAndGetPipelineFile(pipelineDirectory string, pipelineFileName string, cacheEnabled bool) string {
	pipelineSpecsFromFile := utils.PipelineSpecFromFile(pipelineFilesRootDir, pipelineDirectory, pipelineFileName)
	if _, platformSpecExists := pipelineSpecsFromFile["platform_spec"]; platformSpecExists {
		pipelineSpecs := pipelineSpecsFromFile["pipeline_spec"].(map[string]interface{})
		configurePipelineCacheSettings(&pipelineSpecs, cacheEnabled)

	} else {
		configurePipelineCacheSettings(&pipelineSpecsFromFile, cacheEnabled)
	}
	unmarshalledPipelineSpec, unmarshallErr := yaml.Marshal(pipelineSpecsFromFile)
	Expect(unmarshallErr).NotTo(HaveOccurred(), "Failed to unmarshall pipeline spec")
	newPipelineFilePath := utils.CreateTempFile(unmarshalledPipelineSpec)
	return newPipelineFilePath
}

func uploadAPipeline(pipelineFile string, pipelineName *string) *pipeline_upload_model.V2beta1Pipeline {
	logger.Log("Create a pipeline")
	pipelineUploadParams.SetName(pipelineName)
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	createdPipeline, err = pipelineUploadClient.UploadFile(pipelineFile, pipelineUploadParams)
	Expect(err).NotTo(HaveOccurred(), "Failed to upload pipeline")
	createdPipelines = append(createdPipelines, createdPipeline)
	return createdPipeline
}

func createExperiment(experimentName string) *experiment_model.V2beta1Experiment {
	logger.Log("Create an experiment with name %s", experimentName)
	createExperimentParams := experiment_params.NewExperimentServiceCreateExperimentParams()
	createExperimentParams.Body = &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
	}
	createdExperiment, experimentErr := experimentClient.Create(createExperimentParams)
	Expect(experimentErr).NotTo(HaveOccurred(), "Failed to create experiment")
	createdExperimentIds = append(createdExperimentIds, createdExperiment.ExperimentID)
	return createdExperiment
}

func createPipelineRunAndVerify(pipelineID *string, experimentID *string, pipelineInputMap map[string]interface{}) {
	createdPipelines = append(createdPipelines, createdPipeline)
	createPipelineVersions, _, _, listPipelineVersionErr := utils.ListPipelineVersions(pipelineClient, *pipelineID)
	Expect(listPipelineVersionErr).NotTo(HaveOccurred(), "Failed to list pipeline versions for pipeline with id="+*pipelineID)
	createdPipelineRun = createPipelineRun(pipelineID, &createPipelineVersions[0].PipelineVersionID, experimentID, pipelineInputMap)
	createdRunIds = append(createdRunIds, createdPipelineRun.RunID)
	expectedPipelineRun := createExpectedPipelineRun(&createdPipeline.PipelineID, &createPipelineVersions[0].PipelineVersionID, experimentID, pipelineInputMap, false)
	matcher.MatchPipelineRuns(createdPipelineRun, expectedPipelineRun)
	createdPipelineRunFromDB, createRunError := runClient.Get(&run_params.RunServiceGetRunParams{
		RunID: createdPipelineRun.RunID,
	})
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to get run with Id="+createdPipelineRun.RunID)

	// Making the fields that can be different but we don't care about equal to stabilize tests
	matcher.MatchPipelineRuns(createdPipelineRun, createdPipelineRunFromDB)
}

func createPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", pipelineID, pipelineVersionID)
	createRunRequest := &run_params.RunServiceCreateRunParams{Body: createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams)}
	createdRun, createRunError := runClient.Create(createRunRequest)
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to create run for pipeline with id="+*pipelineID)
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

func createPipelineRunPayload(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
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
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: inputParams,
		},
	}
}

func createExpectedPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}, archived bool) *run_model.V2beta1Run {
	expectedRun := createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams)
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
		Expect(expError).NotTo(HaveOccurred(), "Failed to list experiments")
		for _, experiment := range experminents {
			if strings.ToLower(experiment.DisplayName) == "default" {
				expectedRun.ExperimentID = experiment.ExperimentID
			}
		}
	}
	return expectedRun
}

func configurePipelineCacheSettings(pipelineSpec *map[string]interface{}, cacheEnabled bool) {
	if pipelineTasks, pipelineTasksExists := (*pipelineSpec)["root"].(map[string]interface{})["dag"].(map[string]interface{})["tasks"]; pipelineTasksExists {
		for _, pipelineTask := range pipelineTasks.(map[string]interface{}) {
			task := pipelineTask.(map[string]interface{})
			cachingOptionMap := make(map[string]interface{})
			if cacheEnabled {
				cachingOptionMap["enableCache"] = true
			}
			task["cachingOptions"] = cachingOptionMap
		}
	}
}

func getPipelineRunTimeInputs(pipelineSpecFile string) map[string]interface{} {
	pipelineSpec := utils.ReadYamlFile(pipelineSpecFile).(map[string]interface{})
	pipelineInputMap := make(map[string]interface{})
	if pipelineInputDef, pipelineInputParamsExists := pipelineSpec["root"].(map[string]interface{})["inputDefinitions"]; pipelineInputParamsExists {
		if pipelineInput, pipelineInputExists := pipelineInputDef.(map[string]interface{})["parameters"]; pipelineInputExists {
			for input, value := range pipelineInput.(map[string]interface{}) {
				valueMap := value.(map[string]interface{})
				_, defaultValExists := valueMap["defaultValue"]
				optional, optionalExists := valueMap["isOptional"]
				if optionalExists && optional.(bool) {
					continue
				}
				if !defaultValExists || !optionalExists {
					valueType := valueMap["parameterType"].(string)
					switch valueType {
					case "NUMBER_INTEGER":
						pipelineInputMap[input] = rand.Int()
					case "STRING":
						pipelineInputMap[input] = utils.GetRandomString(20)
					case "STRUCT":
						pipelineInputMap[input] = map[string]interface{}{
							"A": strconv.FormatFloat(rand.Float64(), 'g', -1, 64),
							"B": strconv.FormatFloat(rand.Float64(), 'g', -1, 64),
						}
					case "LIST":
						pipelineInputMap[input] = []string{utils.GetRandomString(20)}
					case "BOOLEAN":
						pipelineInputMap[input] = true
					default:
						pipelineInputMap[input] = utils.GetRandomString(20)
					}
				}

			}
		}
	}
	return pipelineInputMap
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
