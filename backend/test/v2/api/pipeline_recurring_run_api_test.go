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
		It("Create a Pipeline Run with cron that runs at a specific time and day", func() {
		})
		It("Create a Pipeline Run with cron that runs on alternate days", func() {
		})
		It("Create a Pipeline Run with cron that runs right now", func() {
		})
		It("Create a Pipeline Run with cache disabled", func() {
		})
	})

	Context("Disable reccurring pipeline run >", func() {
		It("Create a Recurring pipeline Run, disable a run and make sure its not deleted", func() {
		})
	})
	Context("Enable a disabled reccurring pipeline run >", func() {
		It("Create a Recurring pipeline Run, disable it and then enable it", func() {
		})
	})
	Context("List reccurring pipeline run >", func() {
		It("Create 2 Recurring pipeline Runs, and list it", func() {
		})
		It("List Recurring pipeline Runs when no recurring runs exist", func() {
		})
		It("List Recurring pipeline Runs and sort by display name in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by display name in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by id in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by id in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by pipeline version id in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by pipeline version id in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by creation date in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by creation date in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by updated date in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by updated date in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by cron time in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by cron time in descending order", func() {
		})
		It("List Recurring pipeline Runs by specifying page size", func() {
		})
		It("List Recurring pipeline Runs by specifying page size and iterate over atleast 2 pages", func() {
		})
		It("List Recurring pipeline Runs filtering by run id", func() {
		})
		It("List Recurring pipeline Runs filtering by display name containing", func() {
		})
		It("List Recurring pipeline Runs filtering by creation date", func() {
		})
		It("List Recurring pipeline Runs filtering by cron time", func() {
		})
		It("List pipeline Runs by experiment id", func() {
		})
		It("List pipeline Runs by namespace", func() {
		})
	})

	Context("Get a reccurring pipeline run >", func() {
		It("Create a Recurring pipeline Run, verify its details", func() {
		})
	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################
var _ = Describe("Verify Pipeline Run Negative Tests >", Label("Negative", "PipelineRun", FullRegression, S2), func() {

	Context("Create reccurring pipeline run >", func() {
		It("Create a Pipeline Run with invalid cron", func() {
		})
	})
	Context("Disable a recurring pipeline run >", func() {
		It("Disable a deleted recurring run", func() {
		})
		It("Disable a non existent recurring run", func() {
		})
		It("Disable already disabled recurring run", func() {
		})
	})
	Context("Enable a recurring pipeline run >", func() {
		It("Enable a deleted recurring run", func() {
		})
		It("Enable a non existent recurring run", func() {
		})
		It("Enable an already enabled recurring run", func() {
		})
		It("Enable an recurring run for a run that's associated with a delete experiment", func() {
		})
		It("Enable an recurring run for a run that's associated with an archived experiment", func() {
		})
	})
	Context("Delete a recurring pipeline run >", func() {
		It("Delete a deleted recurring run", func() {
		})
		It("Delete a non existent recurring run", func() {
		})
		It("Delete an already deleted recurring run", func() {
		})
	})
	Context("Associate a pipeline recurring run with invalid experiment >", func() {
		It("Associate a recurring run with an archived experiment", func() {
		})
		It("Associate a recurring run with non existent experiment", func() {
		})
		It("Associate a recurring run with deleted experiment", func() {
		})
	})
	Context("Get a reccurring pipeline run >", func() {
		It("Get Recurring pipeline Run for a deleted run", func() {
		})
		It("Get Recurring pipeline Run for a non existing run", func() {
		})
		It("Get Recurring pipeline Run for a non recurring run i.e. one-off run", func() {
		})
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################
