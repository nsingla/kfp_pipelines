# V2beta1ArtifactTask

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** | Output only. The unique server generated id of the ArtifactTask. | [optional] [readonly] 
**artifact_id** | **str** |  | [optional] 
**run_id** | **str** |  | [optional] 
**task_id** | **str** |  | [optional] 
**type** | [**V2beta1ArtifactTaskType**](V2beta1ArtifactTaskType.md) |  | [optional] 
**producer_task_name** | **str** | The task that produced this artifact For example in the case of a pipeline channel that is an output artifact you might have as input something like the following in the IR:   taskOutputArtifact:     outputArtifactKey: output_dataset     producerTask: create-dataset These fields are used to track this lineage.  For outputs, the producer task is the component name of the task that produced the artifact. | [optional] 
**producer_key** | **str** | For outputs, the key is the name of the parameter in the component spec (found in OutputDefinitions) used to output the artifact. | [optional] 
**artifact_key** | **str** | The parameter name for the input/output artifact This maybe the same as the Artifact name if the artifact name is not specified. It is used to resolve artifact pipeline channels. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


