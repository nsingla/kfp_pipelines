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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"

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
		pipelineCacheEnabled bool
	}

	testParams := []TestParams{
		{pipelineCacheEnabled: true},
		{pipelineCacheEnabled: false},
	}
	pipelineDirectory := "valid"
	pipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

	Context("Create a valid pipeline and verify the created run >", func() {
		for _, param := range testParams {
			for _, pipelineFile := range pipelineFiles {
				It(fmt.Sprintf("Create a '%s' pipeline with cacheEnabled=%t and verify run", pipelineFile, param.pipelineCacheEnabled), func() {
					configuredPipelineSpecFile := configureCacheSettingAndGetPipelineFile(pipelineDirectory, pipelineFile, param.pipelineCacheEnabled)
					createdPipeline := uploadAPipeline(configuredPipelineSpecFile, &pipelineGeneratedName)
					createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
					pipelineRuntimeInputs := getPipelineRunTimeInputs(configuredPipelineSpecFile)
					createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs, true)
					createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs, true)
				})
			}
		}

		It(fmt.Sprintf("Create a '%s' pipeline, create an experiement and verify run with associated experiment", pipelineFiles[0]), Label(Smoke), func() {
			pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
		})
	})

	Context("Create a pipeline run without pipeline ref context >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create a run by specifying direct pipeline version id and not as a pipeline version ref", func() {
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs, false)
			createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs, false)
		})
	})

	Context("Associate a single experiment with multiple pipeline runs >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create an experiment and associate it multiple pipeline runs of the same pipeline", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun1 := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			createdExpectedRunAndVerify(createdPipelineRun1, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)

			createdPipelineRun2 := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			createdExpectedRunAndVerify(createdPipelineRun2, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
		})
		It("Create an experiment and associate it pipeline runs of different pipelines", func() {

			createdExperiment := createExperiment(experimentName)
			createdPipeline1 := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipeline1Version := getPipelineVersions(&createdPipeline1.PipelineID)
			pipeline2Name := pipelineGeneratedName + "2"
			createdPipeline2 := uploadAPipeline(pipelineFile, &pipeline2Name)
			createdPipeline2Version := getPipelineVersions(&createdPipeline2.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun1 := createPipelineRun(&createdPipeline1.PipelineID, &createdPipeline1Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			createdExpectedRunAndVerify(createdPipelineRun1, &createdPipeline1.PipelineID, &createdPipeline1Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)

			createdPipelineRun2 := createPipelineRun(&createdPipeline2.PipelineID, &createdPipeline2Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			createdExpectedRunAndVerify(createdPipelineRun2, &createdPipeline2.PipelineID, &createdPipeline2Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
		})
	})

	Context("Archive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create a pipeline run, archive it and verify that the run state does not change on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := getPipelineRun(&createdPipelineRun.RunID)
			Expect(createdPipelineRun.State).To(Equal(pipelineRunAfterArchive.State))
			Expect(pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})

		It("Create a pipeline run, wait for the run to move to RUNNING, archive it and verify that the run state is still RUNNING on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			waitForRunToBeInState(&createdPipelineRun.RunID, run_model.V2beta1RuntimeStateRUNNING)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := getPipelineRun(&createdPipelineRun.RunID)
			Expect(pipelineRunAfterArchive.State).To(Equal(run_model.V2beta1RuntimeStateRUNNING))
			Expect(pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})
	})

	Context("Unarchive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create a pipeline run, archive it and unarchive it and verify the storage state", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := getPipelineVersions(&createdPipeline.PipelineID)
			pipelineRuntimeInputs := getPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, true)
			archivePipelineRun(&createdPipelineRun.RunID)
			unArchivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterUnArchive := getPipelineRun(&createdPipelineRun.RunID)
			Expect(pipelineRunAfterUnArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateAVAILABLE))
		})
	})

	Context("Create reccurring pipeline run >", func() {
		createExperimentOptions := []bool{true, false}
		for range createExperimentOptions {
			It("Create a Pipeline Run with cron that runs every 5min", func() {
			})
			It("Create a Pipeline Run with cron that runs at a specific time and day", func() {
			})
			It("Create a Pipeline Run with cron that runs on alternate days", func() {
			})
		}
	})

	Context("Create reccurring pipeline run associated with a specific experiment>", func() {
		It("Create a Pipeline Run with cron that runs every 5min", func() {
		})
		It("Create a Pipeline Run with cron that runs at a specific time and day", func() {
		})
		It("Create a Pipeline Run with cron that runs on alternate days", func() {
		})
	})

	Context("Terminate a pipeline run >", func() {
		It("Terminate a run in RUNNING state", func() {
		})
		It("Terminate a run in PENDING state", func() {
		})
		It("Terminate a run in SUCCESSFUL or ERRORED state", func() {
		})
	})

	Context("Get All pipeline run >", func() {
		It("Create a Pipeline Run and validate that it gets returned in the List Runs API call", func() {
		})
	})

	Context("Create reccurring pipeline run >", func() {
		It("Create a Pipeline Run with invalid cron", func() {
		})
	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################
var _ = Describe("Verify Pipeline Run Negative Tests >", Label("Negative", "PipelineRun", FullRegression, S2), func() {
	Context("Unarchive a pipeline run >", func() {
		It("Unarchive a deleted run", func() {
		})
		It("Unarchive a non existent run", func() {
		})
		It("Unarchive an available run", func() {
		})
	})
	Context("Archive a pipeline run >", func() {
		It("Archive a deleted run", func() {
		})
		It("Archive a non existent run", func() {
		})
		It("Archive an already archived run", func() {
		})
	})
	Context("Terminate a pipeline run >", func() {
		It("Terminate a deleted run", func() {
		})
		It("Terminate a non existent run", func() {
		})
		It("Terminate an already terminated run", func() {
		})
	})
	Context("Delete a pipeline run >", func() {
		It("Delete a deleted run", func() {
		})
		It("Delete a non existent run", func() {
		})
	})
	Context("Associate a pipeline run with invalid experiment >", func() {
		It("Associate a run with an archived experiment", func() {
		})
		It("Associate a run with non existent experiment", func() {
		})
		It("Associate a run with deleted experiment", func() {
		})
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func configureCacheSettingAndGetPipelineFile(pipelineDirectory string, pipelineFileName string, cacheEnabled bool) string {
	pipelineSpecsFromFile := utils.PipelineSpecFromFile(pipelineFilesRootDir, pipelineDirectory, pipelineFileName)
	var marshalledPipelineSpecs []byte
	var marshalledPlatformSpecs []byte
	var marshallErr error
	var newPipelineFile *os.File
	if _, platformSpecExists := pipelineSpecsFromFile["platform_spec"]; platformSpecExists {
		pipelineSpecs := pipelineSpecsFromFile["pipeline_spec"].(map[string]interface{})
		pipelineSpecs = *configurePipelineCacheSettings(&pipelineSpecs, cacheEnabled)
		pipelineSpecsFromFile["pipeline_spec"] = pipelineSpecs
		marshalledPipelineSpecs, marshallErr = yaml.Marshal(pipelineSpecsFromFile["pipeline_spec"])
		Expect(marshallErr).NotTo(HaveOccurred())
		marshalledPlatformSpecs, marshallErr = yaml.Marshal(pipelineSpecsFromFile["platform_spec"])
		Expect(marshallErr).NotTo(HaveOccurred())
		newPipelineFile = utils.CreateTempFile([][]byte{marshalledPipelineSpecs, []byte("\n---\n"), marshalledPlatformSpecs})
	} else {
		pipelineSpecsFromFile = *configurePipelineCacheSettings(&pipelineSpecsFromFile, cacheEnabled)
		marshalledPipelineSpecs, marshallErr = yaml.Marshal(pipelineSpecsFromFile)
		Expect(marshallErr).NotTo(HaveOccurred(), "Failed to marshall pipeline spec")
		newPipelineFile = utils.CreateTempFile([][]byte{marshalledPipelineSpecs})
	}
	return newPipelineFile.Name()
}

func uploadAPipeline(pipelineFile string, pipelineName *string) *pipeline_upload_model.V2beta1Pipeline {
	logger.Log("Create a pipeline")
	pipelineUploadParams.SetName(pipelineName)
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	createdPipeline, uploadErr := pipelineUploadClient.UploadFile(pipelineFile, pipelineUploadParams)
	Expect(uploadErr).NotTo(HaveOccurred(), "Failed to upload pipeline")
	createdPipelines = append(createdPipelines, createdPipeline)
	return createdPipeline
}

func getPipelineVersions(pipelineID *string) *pipeline_model.V2beta1PipelineVersion {
	createPipelineVersions, _, _, listPipelineVersionErr := utils.ListPipelineVersions(pipelineClient, *pipelineID)
	Expect(listPipelineVersionErr).NotTo(HaveOccurred(), "Failed to list pipeline versions for pipeline with id="+*pipelineID)
	return createPipelineVersions[0]
}

func createExperiment(experimentName string) *experiment_model.V2beta1Experiment {
	logger.Log("Create an experiment with name %s", experimentName)
	createExperimentParams := experiment_params.NewExperimentServiceCreateExperimentParams()
	createExperimentParams.Body = &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
		Namespace:   *namespace,
	}
	createdExperiment, experimentErr := experimentClient.Create(createExperimentParams)
	Expect(experimentErr).NotTo(HaveOccurred(), "Failed to create experiment")
	createdExperimentIds = append(createdExperimentIds, createdExperiment.ExperimentID)
	return createdExperiment
}

func createdExpectedRunAndVerify(createdPipelineRun *run_model.V2beta1Run, pipelineID *string, pipelineVersionID *string, experimentID *string, pipelineInputMap map[string]interface{}, createPipelineRunWithPipelineRef bool) {
	expectedPipelineRun := createExpectedPipelineRun(pipelineID, pipelineVersionID, experimentID, pipelineInputMap, false, createPipelineRunWithPipelineRef)
	matcher.MatchPipelineRuns(createdPipelineRun, expectedPipelineRun)
	createdPipelineRunFromDB, createRunError := runClient.Get(&run_params.RunServiceGetRunParams{
		RunID: createdPipelineRun.RunID,
	})
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to get run with Id="+createdPipelineRun.RunID)

	// Making the fields that can be different but we don't care about equal to stabilize tests
	matcher.MatchPipelineRuns(createdPipelineRun, createdPipelineRunFromDB)
}

func createPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}, createPipelineRunWithPipelineRef bool) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", pipelineID, pipelineVersionID)
	createRunRequest := &run_params.RunServiceCreateRunParams{Body: createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams, createPipelineRunWithPipelineRef)}
	createdRun, createRunError := runClient.Create(createRunRequest)
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to create run for pipeline with id="+*pipelineID)
	createdRunIds = append(createdRunIds, createdRun.RunID)
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

func archivePipelineRun(pipelineRunID *string) {
	logger.Log("Archiving a pipeline run with id=%s and versionId=%s", pipelineRunID)
	archiveRunParams := &run_params.RunServiceArchiveRunParams{
		RunID: *pipelineRunID,
	}
	archiveRunError := runClient.Archive(archiveRunParams)
	Expect(archiveRunError).NotTo(HaveOccurred(), "Failed to archive run with id="+*pipelineRunID)
	logger.Log("Successfully archived run with runId=%s", pipelineRunID)
}

func unArchivePipelineRun(pipelineRunID *string) {
	logger.Log("Unarchiving a pipeline run with id=%s and versionId=%s", pipelineRunID)
	unArchiveRunParams := &run_params.RunServiceUnarchiveRunParams{
		RunID: *pipelineRunID,
	}
	unarchiveRunError := runClient.Unarchive(unArchiveRunParams)
	Expect(unarchiveRunError).NotTo(HaveOccurred(), "Failed to un-archive run with id="+*pipelineRunID)
	logger.Log("Successfully unarchived run with runId=%s", pipelineRunID)
}

func getPipelineRun(pipelineRunID *string) *run_model.V2beta1Run {
	logger.Log("Get a pipeline run with id=%s", *pipelineRunID)
	pipelineRun, runError := runClient.Get(&run_params.RunServiceGetRunParams{
		RunID: *pipelineRunID,
	})
	Expect(runError).NotTo(HaveOccurred(), "Failed to get run with id="+*pipelineRunID)
	return pipelineRun
}

func waitForRunToBeInState(pipelineRunID *string, expectedState run_model.V2beta1RuntimeState) {
	timeout := time.After(30 * time.Second)
	currentPipelineRunState := getPipelineRun(pipelineRunID).State
	for currentPipelineRunState != expectedState {
		logger.Log("Waiting for pipeline run with id=%s to be in %v state", pipelineRunID, expectedState)
		time.Sleep(1 * time.Second)
		select {
		case <-timeout:
			Fail("Timeout waiting for pipeline run with id runId=" + *pipelineRunID + " to be in expected state")
		default:
			logger.Log("Pipeline run with id=%s is in %v state", pipelineRunID, currentPipelineRunState)
			currentPipelineRunState = getPipelineRun(pipelineRunID).State
		}
	}
}

func createPipelineRunPayload(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}, createPipelineRunWithPipelineRef bool) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run body")
	runPayload := &run_model.V2beta1Run{
		DisplayName:    runName,
		Description:    runDescription,
		ExperimentID:   utils.ParsePointersToString(experimentID),
		ServiceAccount: utils.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: inputParams,
		},
	}
	if createPipelineRunWithPipelineRef {
		runPayload.PipelineVersionReference = &run_model.V2beta1PipelineVersionReference{
			PipelineID:        utils.ParsePointersToString(pipelineID),
			PipelineVersionID: utils.ParsePointersToString(pipelineVersionID),
		}
	} else {
		runPayload.PipelineVersionID = utils.ParsePointersToString(pipelineVersionID)
	}
	return runPayload
}

func createExpectedPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}, archived bool, createPipelineRunWithPipelineRef bool) *run_model.V2beta1Run {
	expectedRun := createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams, createPipelineRunWithPipelineRef)
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

func configurePipelineCacheSettings(pipelineSpec *map[string]interface{}, cacheEnabled bool) *map[string]interface{} {
	if pipelineTasks, pipelineTasksExists := (*pipelineSpec)["root"].(map[string]interface{})["dag"].(map[string]interface{})["tasks"]; pipelineTasksExists {
		for _, pipelineTask := range pipelineTasks.(map[string]interface{}) {
			task := pipelineTask.(map[string]interface{})
			cachingOptionMap := make(map[string]interface{})
			if cacheEnabled {
				cachingOptionMap["enableCache"] = true
			}
			task["cachingOptions"] = cachingOptionMap
		}
		return pipelineSpec
	} else {
		return nil
	}
}

func getPipelineRunTimeInputs(pipelineSpecFile string) map[string]interface{} {
	pipelineSpec := utils.ReadYamlFile(pipelineSpecFile).(map[string]interface{})
	pipelineInputMap := make(map[string]interface{})
	var pipelineRoot map[string]interface{}
	if _, platformSpecExists := pipelineSpec["platform_spec"]; platformSpecExists {
		pipelineRoot = pipelineSpec["pipeline_spec"].(map[string]interface{})["root"].(map[string]interface{})
	} else {
		pipelineRoot = pipelineSpec["root"].(map[string]interface{})
	}
	if pipelineInputDef, pipelineInputParamsExists := pipelineRoot["inputDefinitions"]; pipelineInputParamsExists {
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
