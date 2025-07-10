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

var _ = Describe("List Pipelines API Tests >", Label("Positive", "Pipeline", "PipelineList", FullRegression, S1), func() {

	Context("Basic List Operations >", func() {
		It("When no pipelines exist", func() {
		})
		It("After creating a single pipeline", func() {
		})
		It("After creating multiple pipelines", func() {
		})
		It("By namespace", func() {
		})
	})
	Context("Pagination >", func() {
		It("List pipelines with page size limit", func() {
		})
		It("List pipelines with pagination - iterate through all pages (atleast 2)", func() {
		})
	})
	Context("Sorting >", func() {
		It("Sort by name in ascending order", func() {
		})
		It("Sort by name in descending order", func() {
		})
		It("Sort by display name containing substring in ascending order", func() {
		})
		It("Sort by display name containing substring in descending order", func() {
		})
		It("Sort by creation date in ascending order", func() {
		})
		It("Sort by creation date in descending order", func() {
		})
	})
	Context("Filtering >", func() {
		It("Filter by pipeline id", func() {
		})
		It("Filter by name", func() {
		})
		It("Filter by created at", func() {
		})
		It("Filter by namespace", func() {
		})
		It("Filter by description", func() {
		})
	})
	Context("Combined Parameters >", func() {
		It("Filter and sort by name in ascending order", func() {
		})
		It("Filter and sort by created date in descending order", func() {
		})
	})
	Context("Combined Parameters >", func() {
		It("Filter and sort by name in ascending order", func() {
		})
		It("Filter and sort by created date in descending order", func() {
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
