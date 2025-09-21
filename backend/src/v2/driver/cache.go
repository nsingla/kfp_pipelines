// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
)

func collectOutputArtifactMetadataFromCache(ctx context.Context, executorInput *pipelinespec.ExecutorInput, cachedMLMDExecutionID int64, mlmd *metadata.Client) ([]*metadata.OutputArtifact, error) {
	outputArtifacts, err := mlmd.GetOutputArtifactsByExecutionId(ctx, cachedMLMDExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MLMDOutputArtifactsByName by executionId %v: %w", cachedMLMDExecutionID, err)
	}

	// Register artifacts with MLMD.
	registeredMLMDArtifacts := make([]*metadata.OutputArtifact, 0, len(executorInput.GetOutputs().GetArtifacts()))
	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		artifact := artifactList.Artifacts[0]
		outputArtifact, ok := outputArtifacts[name]
		if !ok {
			return nil, fmt.Errorf("unable to find artifact with name %v in mlmd output artifacts", name)
		}
		outputArtifact.Schema = artifact.GetType().GetInstanceSchema()
		registeredMLMDArtifacts = append(registeredMLMDArtifacts, outputArtifact)
	}
	return registeredMLMDArtifacts, nil
}

// getFingerPrint generates a fingerprint for caching. The PVC names are included in the fingerprint since it's assumed
// PVCs have side effects (e.g. files written for tasks later on in the run) on the execution. If the PVC names are
// different, the execution shouldn't be reused for the cache.
func getFingerPrint(opts Options, executorInput *pipelinespec.ExecutorInput, pvcNames []string) (string, error) {
	outputParametersTypeMap := make(map[string]string)
	for outputParamName, outputParamSpec := range opts.Component.GetOutputDefinitions().GetParameters() {
		outputParametersTypeMap[outputParamName] = outputParamSpec.GetParameterType().String()
	}
	userCmdArgs := make([]string, 0, len(opts.Container.Command)+len(opts.Container.Args))
	userCmdArgs = append(userCmdArgs, opts.Container.Command...)
	userCmdArgs = append(userCmdArgs, opts.Container.Args...)

	// Deduplicate PVC names and sort them to ensure consistent fingerprint generation.
	pvcNamesMap := map[string]struct{}{}
	for _, pvcName := range pvcNames {
		pvcNamesMap[pvcName] = struct{}{}
	}

	sortedPVCNames := make([]string, 0, len(pvcNamesMap))
	for pvcName := range pvcNamesMap {
		sortedPVCNames = append(sortedPVCNames, pvcName)
	}
	sort.Strings(sortedPVCNames)

	cacheKey, err := cacheutils.GenerateCacheKey(
		executorInput.GetInputs(),
		executorInput.GetOutputs(),
		outputParametersTypeMap,
		userCmdArgs,
		opts.Container.Image,
		sortedPVCNames,
	)
	if err != nil {
		return "", fmt.Errorf("failure while generating CacheKey: %w", err)
	}
	fingerPrint, err := cacheutils.GenerateFingerPrint(cacheKey)
	return fingerPrint, err
}

func getFingerPrintsAndID(
	ctx context.Context,
	execution *Execution,
	driverAPI DriverAPI,
	opts *Options,
	pvcNames []string) (fingerprint string, task *apiv2beta1.PipelineTaskDetail, err error) {

	if opts.CacheDisabled || !execution.WillTrigger() || opts.Task.GetCachingOptions().GetEnableCache() == false {
		return "", nil, nil
	}

	glog.Infof("Task {%s} enables cache", opts.Task.GetTaskInfo().GetName())
	fingerPrint, err := getFingerPrint(*opts, execution.ExecutorInput, pvcNames)
	if err != nil {
		return "", nil, fmt.Errorf("failure while getting fingerPrint: %w", err)
	}

	filter := apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{
				Operation: apiv2beta1.Predicate_EQUALS,
				Key:       "fingerPrint",
				Value:     &apiv2beta1.Predicate_StringValue{StringValue: fingerPrint},
			},
			{
				Operation: apiv2beta1.Predicate_EQUALS,
				Key:       "status",
				Value:     &apiv2beta1.Predicate_IntValue{IntValue: int32(apiv2beta1.PipelineTaskDetail_SUCCEEDED)},
			},
		},
	}
	tasks, err := driverAPI.ListTasks(ctx, &apiv2beta1.ListTasksRequest{
		ParentFilter: &apiv2beta1.ListTasksRequest_RunId{RunId: opts.RunID},
		Filter:       filter.String(),
	})
	if err != nil {
		return "", nil, fmt.Errorf("failure while listing tasks: %w", err)
	}

	if len(tasks.Tasks) == 0 {
		glog.Infof("No cached tasks found for task {%s}", opts.Task.GetTaskInfo().GetName())
		return fingerPrint, nil, nil
	} else if len(tasks.Tasks) > 1 {
		return "", nil, fmt.Errorf("multiple cached tasks found for task {%s}", opts.Task.GetTaskInfo().GetName())
	}

	glog.Infof("Got a cache hit for task {%s}", opts.Task.GetTaskInfo().GetName())
	return fingerPrint, tasks.Tasks[0], nil
}
