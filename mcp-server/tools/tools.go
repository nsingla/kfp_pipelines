package tools

var PipelineTools = []struct {
	Name        string
	Description string
	Handler     any
}{
	{
		Name:        "list_pipeline_versions",
		Description: "List all pipeline versions either by pipeline id or all",
		Handler:     ListPipelineVersionsHandler,
	},
	{
		Name:        "upload_pipeline",
		Description: "Upload a pipeline IR yaml",
		Handler:     UploadPipelineHandler,
	},
}
