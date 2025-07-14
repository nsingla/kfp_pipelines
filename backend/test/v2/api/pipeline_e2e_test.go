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

var _ = Describe("Verify Pipeline Upload >", Label("Positive", "PipelineUpload", S1), func() {

	/* Critical Positive Scenarios of uploading a pipeline file */
	Context("Upload a valid critical pipeline file and verify pipeline metadata after upload >", Label(Smoke, S1), func() {
		var pipelineDir = "valid/types"
		criticalPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range criticalPipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				logger.Log("Uploading pipeline file %s", pipelineFile)
				uploadedPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFile, &pipelineGeneratedName)
				logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
				uploadedPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
				pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
				logger.Log("Create run for pipeline with id: '%s' and name: '%s'", uploadedPipeline.PipelineID, uploadedPipeline.DisplayName)
				uploadedPipelineRun := createPipelineRun(&uploadedPipeline.PipelineID, &uploadedPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
				logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s", uploadedPipelineRun.RunID, uploadedPipeline.PipelineID)

			})
		}
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################
