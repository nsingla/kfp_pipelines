package util

import "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"

type ScopePath struct {
	path          []string
	taskSpec      *pipelinespec.PipelineTaskSpec
	componentSpec *pipelinespec.ComponentSpec
	pipelineSpec  *pipelinespec.PipelineSpec
}

func (s *ScopePath) GetTaskSpec() *pipelinespec.PipelineTaskSpec {
	return s.taskSpec
}

func (s *ScopePath) GetComponent() *pipelinespec.ComponentSpec {
	return s.componentSpec
}

func (s *ScopePath) AddTask(entry string) error {
	return nil
}

func (s *ScopePath) RemoveTask(entry string) error {
	return nil
}

func (s *ScopePath) GetScopePath() []string {
	return s.path
}
