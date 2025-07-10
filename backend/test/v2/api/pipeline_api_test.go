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
	. "github.com/kubeflow/pipelines/backend/test/v2/api/constants"
	. "github.com/onsi/ginkgo/v2"
)

// ####################################################################################################################################################################
// ################################################################### CLASS VARIABLES ################################################################################
// ####################################################################################################################################################################

// ####################################################################################################################################################################
// ################################################################### SETUP AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Run >", Label("Positive", "PipelineRun", FullRegression, S1), func() {

	Context("Create reccurring pipeline run >", func() {
		It("Create a Pipeline Run with cron that runs every 5min", func() {
		})
	})

	Context("Create reccurring pipeline run associated with a specific experiment>", func() {
		It("Create a Pipeline Run with cron that runs every 5min", func() {
		})
	})

	Context("Terminate a pipeline run >", func() {
		It("Terminate a run in RUNNING state", func() {
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
	})
	Context("Archive a pipeline run >", func() {
		It("Archive a deleted run", func() {
		})
	})
	Context("Terminate a pipeline run >", func() {
		It("Terminate a deleted run", func() {
		})
	})
	Context("Delete a pipeline run >", func() {
		It("Delete a deleted run", func() {
		})
	})
	Context("Associate a pipeline run with invalid experiment >", func() {
		It("Associate a run with an archived experiment", func() {
		})
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################
