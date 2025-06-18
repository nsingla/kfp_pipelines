package api

import (
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/onsi/gomega"
)

// MatchPipelines - Deep compare 2 pipelines
func MatchPipelines(actual *model.V2beta1Pipeline, expected *model.V2beta1Pipeline) {
	Expect(actual.PipelineID).To(Not(BeEmpty()), "Pipeline ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline creation time is not as expected")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline name not matching")
	Expect(actual.Namespace).To(Equal(expected.Namespace), "Pipeline Namespace not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")

}

// MatchPipelineVersions - Deep compare 2 pipeline versions - even with deep comparison of pipeline specs
func MatchPipelineVersions(actual *model.V2beta1PipelineVersion, expected *model.V2beta1PipelineVersion) {
	Expect(actual.PipelineVersionID).To(Not(Equal(expected.PipelineVersionID)), "Pipeline Version ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline Version creation time is not as expected")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline Display Name not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")
	MatchMaps(actual.PipelineSpec, expected.PipelineSpec, "Pipeline Spec")
}

// MatchPipelineRunShallow - Shallow match 2 pipeline runs i.e. match only the fields that you do add to the payload when creating a run
func MatchPipelineRunShallow(actual *run_model.V2beta1Run, expected *run_model.V2beta1Run) {
	Expect(actual.RunID).To(Not(BeEmpty()), "Run ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Run time is not before the expected time")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Run Name is not matching")
	Expect(actual.ExperimentID).To(Not(BeEmpty()), "Experiment Id associated with the run is empty")
	Expect(actual.PipelineVersionID).To(Equal(expected.PipelineVersionID), "Pipeline Version Id is not matching")
	MatchMaps(actual.PipelineSpec, expected.PipelineSpec, "Pipeline Spec")
	Expect(actual.PipelineVersionReference.PipelineVersionID).To(Equal(expected.PipelineVersionReference.PipelineVersionID), "Referred Pipeline Version Idis not matching")
	Expect(actual.PipelineVersionReference.PipelineID).To(Equal(expected.PipelineVersionReference.PipelineID), "Referred Pipeline Id is not matching")
	Expect(actual.ServiceAccount).To(Equal(expected.ServiceAccount), "Service Account is not matching")
}
