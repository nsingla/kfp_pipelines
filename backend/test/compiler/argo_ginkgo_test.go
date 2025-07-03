// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	matcher "github.com/kubeflow/pipelines/backend/test/v2/api/matcher"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	"github.com/onsi/ginkgo/v2/types"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// var update = flag.Bool("update", false, "update golden files")
var pipelineFilesRootDir = utils.GetPipelineFilesDir()
var pipelineDirectory = "valid"
var argoYAMLDir = "compiledworkflows"
var testReportDirectory = "reports"
var junitReportFilename = "junit-compile.xml"
var jsonReportFilename = "report-compile.json"
var updateGoldenFiles = flag.Bool("updateCompiledFiles", false, "update golden/expected compiled workflow files")
var createMissingGoldenFiles = flag.Bool("createGoldenFiles", false, "create missing golden/expected compiled workflow files")

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing test logs")
		AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)
	}
})

func TestCompilation(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.FailFast = false
	reporterConfig.GithubOutput = true
	reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "API Tests Suite", suiteConfig, reporterConfig)
}

var _ = BeforeEach(func() {
	logger.Log("Initializing proxy...")
	proxy.InitializeConfigWithEmptyForTests()
})

var _ = Describe("Verify Spec Compilation to Workflow >", Label("Positive", "WorkflowCompiler"), func() {
	pipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

	testParams := []struct {
		compilerOptions argocompiler.Options
		envVars         map[string]string
	}{
		{
			compilerOptions: argocompiler.Options{CacheDisabled: true},
		},
		{
			compilerOptions: argocompiler.Options{CacheDisabled: false},
		},
		{
			compilerOptions: argocompiler.Options{CacheDisabled: false},
			envVars:         map[string]string{"PIPELINE_RUN_AS_USER": "1001", "PIPELINE_LOG_LEVEL": "3"},
		},
	}
	for _, param := range testParams {
		Context(fmt.Sprintf("Verify compiled workflow for a pipeline with compiler options '%v' and env vars %v >", param.compilerOptions, param.envVars), func() {
			for _, pipelineSpecFileName := range pipelineFiles {
				compiledWorkflowFilePath := filepath.Join(argoYAMLDir, pipelineSpecFileName)
				// Set provided env variables
				for envVarName, envVarValue := range param.envVars {
					err := os.Setenv(envVarName, envVarValue)
					Expect(err).To(BeNil(), "Could not set env var %s", envVarName)
				}

				// Defer UnSetting the set env variables at the end of the test
				defer func() {
					for envVarName := range param.envVars {
						err := os.Unsetenv(envVarName)
						Expect(err).To(BeNil(), "Could not unset env var %s", envVarName)
					}
				}()
				It(fmt.Sprintf("When I compile %s pipeline spec, then the compiled yaml should be =%s", pipelineSpecFileName, compiledWorkflowFilePath), func() {
					pipelineSpecs, platformSpec := loadSpecs(pipelineSpecFileName)
					compiledWorflow := getCompiledArgoWorkflow(pipelineSpecs, platformSpec, &argocompiler.Options{})
					if *updateGoldenFiles {
						logger.Log(fmt.Sprintf("Updating golden file %s", compiledWorkflowFilePath))
						_, err := os.Stat(compiledWorkflowFilePath)
						if err != nil {
							logger.Log("File %s does not exist, but if you want to create the missing workflow file, please set 'createGoldenFiles' flag to true", compiledWorkflowFilePath)
						} else {
							createCompiledWorkflowFile(compiledWorflow, compiledWorkflowFilePath)
						}
					} else if *createMissingGoldenFiles {
						_, err := os.Stat(compiledWorkflowFilePath)
						if err == nil {
							logger.Log("Creating Compiled Workflow File '%s'", compiledWorkflowFilePath)
							createCompiledWorkflowFile(compiledWorflow, compiledWorkflowFilePath)
						} else {
							logger.Log("Compiled Workflow File '%s' already exists", compiledWorkflowFilePath)
						}
					} else {
						expectedWorkflow := unmarshallWorkflowYAML(compiledWorkflowFilePath)
						compareWorkflows(compiledWorflow, expectedWorkflow)
					}

				})
			}
		})
	}
})

func loadSpecs(pipelineFileName string) (*pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec) {
	pipelineSpecsFromFile := utils.PipelineSpecFromFile(pipelineFilesRootDir, pipelineDirectory, pipelineFileName)
	var singlePlatformSpec *pipelinespec.SinglePlatformSpec = nil
	pipelineSpecsYaml := make(map[string]interface{})
	if _, platformSpecExists := pipelineSpecsFromFile["platform_spec"]; platformSpecExists {
		pipelineSpecsYaml["pipelineSpec"] = pipelineSpecsFromFile["pipeline_spec"]
		platformSpecBytes, platformMarshallingError := json.Marshal(pipelineSpecsFromFile["platform_spec"])
		Expect(platformMarshallingError).NotTo(HaveOccurred(), "Failed to marshall platform spec map")
		platformSpecs := &pipelinespec.PlatformSpec{}
		err := protojson.Unmarshal(platformSpecBytes, platformSpecs)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to unmarshall platform spec %s", string(platformSpecBytes)))
		for _, spec := range platformSpecs.Platforms {
			singlePlatformSpec = spec
		}
	} else {
		pipelineSpecsYaml["pipelineSpec"] = pipelineSpecsFromFile
	}
	pipelineSpecBytes, marshallingError := json.Marshal(pipelineSpecsYaml)
	Expect(marshallingError).NotTo(HaveOccurred(), "Failed to marshall pipeline spec map")
	pipelineSpecs := &pipelinespec.PipelineJob{}
	err := protojson.Unmarshal(pipelineSpecBytes, pipelineSpecs)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to unmarshal pipeline spec\n %s", string(pipelineSpecBytes)))
	return pipelineSpecs, singlePlatformSpec
}

func getCompiledArgoWorkflow(pipelineSpecs *pipelinespec.PipelineJob, platformSpec *pipelinespec.SinglePlatformSpec, compilerOptions *argocompiler.Options) *v1alpha1.Workflow {
	logger.Log("Compiling Argo Workflow for provided pipeline job and platform spec")
	compiledWorflow, err := argocompiler.Compile(pipelineSpecs, platformSpec, compilerOptions)
	Expect(err).NotTo(HaveOccurred(), "Failed to compile Argo workflow")
	return compiledWorflow
}

func unmarshallWorkflowYAML(filePath string) *v1alpha1.Workflow {
	logger.Log("Unmarshalling Expected Workflow YAML")
	workflowFromFile := utils.ReadYamlFile(filePath)
	workflowFromFileBytes, marshallingError := json.Marshal(workflowFromFile)
	Expect(marshallingError).NotTo(HaveOccurred(), "Failed to marshall workflow map")
	workflow := v1alpha1.Workflow{}
	err := yaml.Unmarshal(workflowFromFileBytes, &workflow)
	Expect(err).NotTo(HaveOccurred())
	logger.Log("Unmarshalled Expected Workflow YAML")
	return &workflow
}

func compareWorkflows(actual *v1alpha1.Workflow, expected *v1alpha1.Workflow) {
	logger.Log("Compared Compiled Workflow with expected input YAML file")
	Expect(actual.Namespace).To(Equal(expected.Namespace), "Namespace is not same")
	Expect(actual.Finalizers).To(Equal(expected.Finalizers), "Finalizers are not same")
	Expect(actual.Name).To(Equal(expected.Name), "Name is not same")
	// Expect(actual.Kind).To(Equal(expected.Kind), "Kind is not same")
	// Expect(actual.GenerateName).To(Equal(expected.GenerateName), "Generate Name is not same")
	matcher.MatchMaps(actual.Labels, expected.Labels, "Labels")
	matcher.MatchMaps(actual.Annotations, expected.Annotations, "Annotations")
	// Match Specs
	for paramIndex, param := range expected.Spec.Arguments.Parameters {
		Expect(actual.Spec.Arguments.Parameters[paramIndex].Name).To(Equal(param.Name), "Parameter Name is not same")
		Expect(actual.Spec.Arguments.Parameters[paramIndex].Description).To(Equal(param.Description), "Parameter Description is not same")
		Expect(actual.Spec.Arguments.Parameters[paramIndex].Default).To(Equal(param.Default), "Parameter Default is not same")
		Expect(areStringsSameWithoutOrder(actual.Spec.Arguments.Parameters[paramIndex].Value.String(), param.Value.String())).To(BeTrue(), "Parameter Value is not same")
		Expect(actual.Spec.Arguments.Parameters[paramIndex].Enum).To(Equal(param.Enum), "Parameter Enum is not same")
		Expect(actual.Spec.Arguments.Parameters[paramIndex].ValueFrom).To(Equal(param.ValueFrom), "Parameter ValueFrom is not same")
	}
	if expected.Spec.Affinity != nil {
		Expect(actual.Spec.Affinity.NodeAffinity).To(Equal(expected.Spec.Affinity.NodeAffinity), "Node Affinity not same")
		Expect(actual.Spec.Affinity.PodAffinity).To(Equal(expected.Spec.Affinity.PodAffinity), "Pod Affinity not same")
		Expect(actual.Spec.Affinity.PodAntiAffinity).To(Equal(expected.Spec.Affinity.PodAntiAffinity), "Pod Anti Affinity not same")
	} else {
		Expect(actual.Spec.Affinity).To(BeNil(), "Affinity is not nil")
	}
	Expect(actual.Spec.ActiveDeadlineSeconds).To(Equal(expected.Spec.ActiveDeadlineSeconds), "ActiveDeadlineSeconds is not same")
	Expect(actual.Spec.ArchiveLogs).To(Equal(expected.Spec.ArchiveLogs), "ArchiveLogs is not same")
	Expect(actual.Spec.ArtifactGC).To(Equal(expected.Spec.ArtifactGC), "ArtifactGC is not same")
	Expect(actual.Spec.ArtifactRepositoryRef).To(Equal(expected.Spec.ArtifactRepositoryRef), "ArtifactRepositoryRef is not same")
	Expect(actual.Spec.AutomountServiceAccountToken).To(Equal(expected.Spec.AutomountServiceAccountToken), "AutomountServiceAccountToken is not same")
	Expect(actual.Spec.DNSConfig).To(Equal(expected.Spec.DNSConfig), "DNSConfig is not same")
	Expect(actual.Spec.DNSPolicy).To(Equal(expected.Spec.DNSPolicy), "DNSPolicy is not same")
	Expect(actual.Spec.Entrypoint).To(Equal(expected.Spec.Entrypoint), "Entrypoint is not same")
	Expect(actual.Spec.Executor).To(Equal(expected.Spec.Executor), "Executor is not same")
	Expect(actual.Spec.Hooks).To(Equal(expected.Spec.Hooks), "Hooks is not same")
	Expect(actual.Spec.HostAliases).To(Equal(expected.Spec.HostAliases), "HostAliases is not same")
	Expect(actual.Spec.HostNetwork).To(Equal(expected.Spec.HostNetwork), "HostNetwork is not same")
	Expect(actual.Spec.ImagePullSecrets).To(Equal(expected.Spec.ImagePullSecrets), "ImagePullSecrets are not same")
	Expect(actual.Spec.Metrics).To(Equal(expected.Spec.Metrics), "Metrics are not same")
	Expect(actual.Spec.NodeSelector).To(Equal(expected.Spec.NodeSelector), "NodeSelector is not same")
	Expect(actual.Spec.OnExit).To(Equal(expected.Spec.OnExit), "OnExit is not same")
	Expect(actual.Spec.Parallelism).To(Equal(expected.Spec.Parallelism), "Parallelism is not same")
	Expect(actual.Spec.PodDisruptionBudget).To(Equal(expected.Spec.PodDisruptionBudget), "PodDisruptionBudget is not same")
	Expect(actual.Spec.PodGC).To(Equal(expected.Spec.PodGC), "PodGC is not same")
	Expect(actual.Spec.PodMetadata).To(Equal(expected.Spec.PodMetadata), "PodMetadata is not same")
	Expect(actual.Spec.PodPriorityClassName).To(Equal(expected.Spec.PodPriorityClassName), "PodPriorityClassName is not same")
	Expect(actual.Spec.PodSpecPatch).To(Equal(expected.Spec.PodSpecPatch), "PodSpecPatch is not same")
	Expect(actual.Spec.Priority).To(Equal(expected.Spec.Priority), "Priority is not same")
	Expect(actual.Spec.RetryStrategy).To(Equal(expected.Spec.RetryStrategy), "RetryStrategy is not same")
	Expect(actual.Spec.SchedulerName).To(Equal(expected.Spec.SchedulerName), "SchedulerName is not same")
	Expect(actual.Spec.SecurityContext).To(Equal(expected.Spec.SecurityContext), "SecurityContext is not same")
	Expect(actual.Spec.Shutdown).To(Equal(expected.Spec.Shutdown), "Shutdown is not same")
	Expect(actual.Spec.ServiceAccountName).To(Equal(expected.Spec.ServiceAccountName), "ServiceAccountName is not same")
	Expect(actual.Spec.Suspend).To(Equal(expected.Spec.Suspend), "Suspend is not same")
	Expect(actual.Spec.Synchronization).To(Equal(expected.Spec.Synchronization), "Synchronization is not same")
	Expect(actual.Spec.Tolerations).To(Equal(expected.Spec.Tolerations), "Tolerations is not same")
	Expect(actual.Spec.TTLStrategy).To(Equal(expected.Spec.TTLStrategy), "TTLStrategy is not same")
	for index, template := range expected.Spec.Templates {
		Expect(actual.Spec.Templates[index].Inputs).To(Equal(template.Inputs), "Template Inputs is not same")
		Expect(actual.Spec.Templates[index].Outputs).To(Equal(template.Outputs), "Template Outputs is not same")
		Expect(actual.Spec.Templates[index].Tolerations).To(Equal(template.Tolerations), "Tolerations is not same")
		Expect(actual.Spec.Templates[index].PodSpecPatch).To(Equal(template.PodSpecPatch), "PodSpecPatch is not same")
		Expect(actual.Spec.Templates[index].Synchronization).To(Equal(template.Synchronization), "Synchronization is not same")
		Expect(actual.Spec.Templates[index].Volumes).To(Equal(template.Volumes), "Volumes is not same")
		Expect(actual.Spec.Templates[index].Suspend).To(Equal(template.Suspend), "Suspend is not same")
		Expect(actual.Spec.Templates[index].SecurityContext).To(Equal(template.SecurityContext), "SecurityContext is not same")
		Expect(actual.Spec.Templates[index].SchedulerName).To(Equal(template.SchedulerName), "SchedulerName is not same")
		Expect(actual.Spec.Templates[index].RetryStrategy).To(Equal(template.RetryStrategy), "RetryStrategy is not same")
		Expect(actual.Spec.Templates[index].Priority).To(Equal(template.Priority), "Priority is not same")
		Expect(actual.Spec.Templates[index].Parallelism).To(Equal(template.Parallelism), "Parallelism is not same")
		Expect(actual.Spec.Templates[index].NodeSelector).To(Equal(template.NodeSelector), "NodeSelector is not same")
		Expect(actual.Spec.Templates[index].Metrics).To(Equal(template.Metrics), "Metrics is not same")
		Expect(actual.Spec.Templates[index].HostAliases).To(Equal(template.HostAliases), "HostAliases is not same")
		Expect(actual.Spec.Templates[index].Executor).To(Equal(template.Executor), "Executor is not same")
		Expect(actual.Spec.Templates[index].ServiceAccountName).To(Equal(template.ServiceAccountName), "ServiceAccountName is not same")
		Expect(actual.Spec.Templates[index].AutomountServiceAccountToken).To(Equal(template.AutomountServiceAccountToken), "AutomountServiceAccountToken is not same")
		Expect(actual.Spec.Templates[index].ActiveDeadlineSeconds).To(Equal(template.ActiveDeadlineSeconds), "ActiveDeadlineSeconds is not same")
		Expect(actual.Spec.Templates[index].Affinity).To(Equal(template.Affinity), "Affinity is not same")
		Expect(actual.Spec.Templates[index].Name).To(Equal(template.Name), "Name is not same")
		// Compare Container
		matchContainer(actual.Spec.Templates[index].Container, template.Container)
		for containerIndex, userContainer := range template.InitContainers {
			matchUserContainer(&actual.Spec.Templates[index].InitContainers[containerIndex], &userContainer)
		}

		Expect(actual.Spec.Templates[index].FailFast).To(Equal(template.FailFast), "FailFast is not same")
		Expect(actual.Spec.Templates[index].ArchiveLocation).To(Equal(template.ArchiveLocation), "ArchiveLocation is not same")
		Expect(actual.Spec.Templates[index].ContainerSet).To(Equal(template.ContainerSet), "ContainerSet is not same")
		Expect(actual.Spec.Templates[index].Daemon).To(Equal(template.Daemon), "Daemon is not same")
		Expect(actual.Spec.Templates[index].Data).To(Equal(template.Data), "Data is not same")
		compareDAG(actual.Spec.Templates[index].DAG, template.DAG)
		Expect(actual.Spec.Templates[index].HTTP).To(Equal(template.HTTP), "HTTP is not same")
		Expect(actual.Spec.Templates[index].Memoize).To(Equal(template.Memoize), "Memoize is not same")
		Expect(actual.Spec.Templates[index].Metadata).To(Equal(template.Metadata), "Metadata is not same")
		Expect(actual.Spec.Templates[index].Plugin).To(Equal(template.Plugin), "Plugin is not same")
		Expect(actual.Spec.Templates[index].PriorityClassName).To(Equal(template.PriorityClassName), "PriorityClassName is not same")
		Expect(actual.Spec.Templates[index].Resource).To(Equal(template.Resource), "Resource is not same")
		Expect(actual.Spec.Templates[index].Script).To(Equal(template.Script), "Script is not same")
		Expect(actual.Spec.Templates[index].Sidecars).To(Equal(template.Sidecars), "Sidecars is not same")
		Expect(actual.Spec.Templates[index].Steps).To(Equal(template.Steps), "Steps is not same")
		Expect(actual.Spec.Templates[index].Timeout).To(Equal(template.Timeout), "Timeout is not same")
	}
	Expect(actual.Spec.TemplateDefaults).To(Equal(expected.Spec.TemplateDefaults), "TemplateDefaults are not same")
	Expect(actual.Spec.VolumeClaimGC).To(Equal(expected.Spec.VolumeClaimGC), "VolumeClaimGC is not same")
	Expect(actual.Spec.Volumes).To(Equal(expected.Spec.Volumes), "Volumes is not same")
	Expect(actual.Spec.VolumeClaimTemplates).To(Equal(expected.Spec.VolumeClaimTemplates), "VolumeClaimTemplates are not same")
	Expect(actual.Spec.WorkflowMetadata).To(Equal(expected.Spec.WorkflowMetadata), "WorkflowMetadata is not same")
	Expect(actual.Spec.WorkflowTemplateRef).To(Equal(expected.Spec.WorkflowTemplateRef), "WorkflowTemplateRef is not same")
}

func matchContainerResourceLimits(actual v1.ResourceList, expected v1.ResourceList) {
	Expect(actual.Pods()).To(Equal(expected.Pods()), "Container Resources Limits Pods is not same")
	actualCPU := actual.Cpu().AsApproximateFloat64()
	expectedCPU := expected.Cpu().AsApproximateFloat64()
	actualMemory := actual.Memory().AsApproximateFloat64()
	expectedMemory := expected.Memory().AsApproximateFloat64()
	Expect(actualCPU).To(Equal(expectedCPU), "Container Resources Limits Cpu is not same")
	Expect(actualMemory).To(Equal(expectedMemory), "Container Resources Limits Memory is not same")
	Expect(actual.Storage()).To(Equal(expected.Storage()), "Container Resources Limits Storage is not same")
	Expect(actual.StorageEphemeral()).To(Equal(expected.StorageEphemeral()), "Container Resources Limits StorageEphemeral is not same")
}

func matchContainer(actual *v1.Container, expected *v1.Container) {
	if expected != nil {
		Expect(actual.Name).To(Equal(expected.Name), "Container Name is not same")
		Expect(actual.Args).To(Equal(expected.Args), "Container Args is not same")
		Expect(actual.SecurityContext).To(Equal(expected.SecurityContext), "Container SecurityContext is not same")
		Expect(actual.Env).To(Equal(expected.Env), "Container Env is not same")
		Expect(actual.EnvFrom).To(Equal(expected.EnvFrom), "Container EnvFrom is not same")
		Expect(actual.Command).To(Equal(expected.Command), "Container Command is not same")
		Expect(actual.ImagePullPolicy).To(Equal(expected.ImagePullPolicy), "Container ImagePullPolicy is not same")
		Expect(actual.Image).To(Equal(expected.Image), "Container Image is not same")
		Expect(actual.Lifecycle).To(Equal(expected.Lifecycle), "Container Lifecycle is not same")
		Expect(actual.LivenessProbe).To(Equal(expected.LivenessProbe), "Container LivenessProbe is not same")
		Expect(actual.Ports).To(Equal(expected.Ports), "Container Ports is not same")
		Expect(actual.ReadinessProbe).To(Equal(expected.ReadinessProbe), "Container ReadinessProbe is not same")
		Expect(actual.ResizePolicy).To(Equal(expected.ResizePolicy), "Container ResizePolicy is not same")
		Expect(actual.StartupProbe).To(Equal(expected.StartupProbe), "Container StartupProbe is not same")
		Expect(actual.Stdin).To(Equal(expected.Stdin), "Container Stdin is not same")
		Expect(actual.StdinOnce).To(Equal(expected.StdinOnce), "Container StdinOnce is not same")
		Expect(actual.RestartPolicy).To(Equal(expected.RestartPolicy), "Container RestartPolicy is not same")
		Expect(actual.TerminationMessagePath).To(Equal(expected.TerminationMessagePath), "Container TerminationMessagePath is not same")
		Expect(actual.TTY).To(Equal(expected.TTY), "Container TTY is not same")
		Expect(actual.TerminationMessagePolicy).To(Equal(expected.TerminationMessagePolicy), "Container TerminationMessagePolicy is not same")
		Expect(actual.VolumeDevices).To(Equal(expected.VolumeDevices), "Container VolumeDevices is not same")
		Expect(actual.VolumeMounts).To(Equal(expected.VolumeMounts), "Container VolumeMounts is not same")
		Expect(actual.WorkingDir).To(Equal(expected.WorkingDir), "Container WorkingDir is not same")
		Expect(actual.Resources.Claims).To(Equal(expected.Resources.Claims), "Container Resources Claims is not same")
		matchContainerResourceLimits(actual.Resources.Limits, expected.Resources.Limits)
		matchContainerResourceLimits(actual.Resources.Requests, expected.Resources.Requests)
	} else {
		Expect(actual).To(BeNil(), "Container is expected to be nil")
	}
}

func matchUserContainer(actual *v1alpha1.UserContainer, expected *v1alpha1.UserContainer) {
	if expected != nil {
		Expect(actual.Name).To(Equal(expected.Name), "User Container Name is not same")
		Expect(actual.Args).To(Equal(expected.Args), "User Container Args is not same")
		Expect(actual.SecurityContext).To(Equal(expected.SecurityContext), "User Container SecurityContext is not same")
		Expect(actual.Env).To(Equal(expected.Env), "User Container Env is not same")
		Expect(actual.EnvFrom).To(Equal(expected.EnvFrom), "User Container EnvFrom is not same")
		Expect(actual.Command).To(Equal(expected.Command), "User Container Command is not same")
		Expect(actual.ImagePullPolicy).To(Equal(expected.ImagePullPolicy), "User Container ImagePullPolicy is not same")
		Expect(actual.Image).To(Equal(expected.Image), "User Container Image is not same")
		Expect(actual.Lifecycle).To(Equal(expected.Lifecycle), "User Container Lifecycle is not same")
		Expect(actual.LivenessProbe).To(Equal(expected.LivenessProbe), "User Container LivenessProbe is not same")
		Expect(actual.Ports).To(Equal(expected.Ports), "User Container Ports is not same")
		Expect(actual.ReadinessProbe).To(Equal(expected.ReadinessProbe), "User Container ReadinessProbe is not same")
		Expect(actual.ResizePolicy).To(Equal(expected.ResizePolicy), "User Container ResizePolicy is not same")
		Expect(actual.StartupProbe).To(Equal(expected.StartupProbe), "User Container StartupProbe is not same")
		Expect(actual.Stdin).To(Equal(expected.Stdin), "User Container Stdin is not same")
		Expect(actual.StdinOnce).To(Equal(expected.StdinOnce), "User Container StdinOnce is not same")
		Expect(actual.RestartPolicy).To(Equal(expected.RestartPolicy), "User Container RestartPolicy is not same")
		Expect(actual.TerminationMessagePath).To(Equal(expected.TerminationMessagePath), "User Container TerminationMessagePath is not same")
		Expect(actual.TTY).To(Equal(expected.TTY), "User Container TTY is not same")
		Expect(actual.TerminationMessagePolicy).To(Equal(expected.TerminationMessagePolicy), "User Container TerminationMessagePolicy is not same")
		Expect(actual.VolumeDevices).To(Equal(expected.VolumeDevices), "User Container VolumeDevices is not same")
		Expect(actual.VolumeMounts).To(Equal(expected.VolumeMounts), "User Container VolumeMounts is not same")
		Expect(actual.WorkingDir).To(Equal(expected.WorkingDir), "User Container WorkingDir is not same")
		Expect(actual.Resources.Claims).To(Equal(expected.Resources.Claims), "User Container Resources Claims is not same")
		matchContainerResourceLimits(actual.Resources.Limits, expected.Resources.Limits)
		matchContainerResourceLimits(actual.Resources.Requests, expected.Resources.Requests)
	} else {
		Expect(actual).To(BeNil(), "User Container is expected to be nil")
	}
}

func compareDAG(actual *v1alpha1.DAGTemplate, expected *v1alpha1.DAGTemplate) {
	if expected != nil {
		Expect(actual.FailFast).To(Equal(expected.FailFast), "DAGTemplate FailFast is not same")
		Expect(actual.Target).To(Equal(expected.Target), "DAGTemplate Target is not same")
		for index, task := range actual.Tasks {
			Expect(actual.Tasks[index].Name).To(Equal(task.Name), "DAGTemplate Task Name is not same")
			Expect(actual.Tasks[index].Hooks).To(Equal(task.Hooks), "DAGTemplate Task Hooks is not same")
			Expect(actual.Tasks[index].Template).To(Equal(task.Template), "DAGTemplate Task Template is not same")
			Expect(actual.Tasks[index].Arguments).To(Equal(task.Arguments), "DAGTemplate Task Arguments is not same")
			Expect(actual.Tasks[index].When).To(Equal(task.When), "DAGTemplate Task When is not same")
			Expect(actual.Tasks[index].ContinueOn).To(Equal(task.ContinueOn), "DAGTemplate Task ContinueOn is not same")
			Expect(actual.Tasks[index].Dependencies).To(Equal(task.Dependencies), "DAGTemplate Task Dependencies is not same")
			Expect(actual.Tasks[index].Depends).To(Equal(task.Depends), "DAGTemplate Task Depends is not same")
			Expect(actual.Tasks[index].Inline).To(Equal(task.Inline), "DAGTemplate Task Inline is not same")
			Expect(actual.Tasks[index].TemplateRef).To(Equal(task.TemplateRef), "DAGTemplate Task TemplateRef is not same")
			Expect(actual.Tasks[index].WithItems).To(Equal(task.WithItems), "DAGTemplate Task WithItems is not same")
			Expect(actual.Tasks[index].WithParam).To(Equal(task.WithParam), "DAGTemplate Task WithParam is not same")
			Expect(actual.Tasks[index].WithSequence).To(Equal(task.WithSequence), "DAGTemplate Task WithItems is not same")
			Expect(actual.Tasks[index].OnExit).To(Equal(task.OnExit), "DAGTemplate Task WithItems is not same")
		}
	} else {
		Expect(actual).To(BeNil(), "DAGTemplate is expected to be nil")
	}

}

// areStringsSameWithoutOrder - checks if two strings contain the same characters, regardless of order.
func areStringsSameWithoutOrder(s1, s2 string) bool {
	// Convert strings to rune slices
	r1 := []rune(s1)
	r2 := []rune(s2)

	// Sort the rune slices
	sort.Slice(r1, func(i, j int) bool { return r1[i] < r1[j] })
	sort.Slice(r2, func(i, j int) bool { return r2[i] < r2[j] })

	// Compare the sorted slices
	return reflect.DeepEqual(r1, r2)
}

func createCompiledWorkflowFile(compiledWorflow *v1alpha1.Workflow, compiledWorkflowFilePath string) *os.File {
	fileContents, err := yaml.Marshal(compiledWorflow)
	Expect(err).NotTo(HaveOccurred())
	return utils.CreateFile(compiledWorkflowFilePath, [][]byte{fileContents})
}
