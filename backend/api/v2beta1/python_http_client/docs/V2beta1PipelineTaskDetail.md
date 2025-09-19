# V2beta1PipelineTaskDetail

Runtime information of a task execution.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 
**display_name** | **str** | User specified name of a task that is defined in [Pipeline.spec][]. | [optional] 
**task_id** | **str** | System-generated ID of a task. | [optional] 
**run_id** | **str** | ID of the parent run. | [optional] 
**pods** | [**list[PipelineTaskDetailTaskPod]**](PipelineTaskDetailTaskPod.md) |  | [optional] 
**cache_fingerprint** | **str** |  | [optional] 
**create_time** | **datetime** | Creation time of a task. | [optional] 
**start_time** | **datetime** | Starting time of a task. | [optional] 
**end_time** | **datetime** | Completion time of a task. | [optional] 
**status** | [**V2beta1RuntimeState**](V2beta1RuntimeState.md) |  | [optional] 
**status_metadata** | **dict(str, object)** |  | [optional] 
**state_history** | [**list[V2beta1RuntimeStatus]**](V2beta1RuntimeStatus.md) | A sequence of task statuses. This field keeps a record of state transitions. | [optional] 
**type** | [**PipelineTaskDetailTaskType**](PipelineTaskDetailTaskType.md) |  | [optional] 
**type_attributes** | [**PipelineTaskDetailTypeAttributes**](PipelineTaskDetailTypeAttributes.md) |  | [optional] 
**error** | [**GooglerpcStatus**](GooglerpcStatus.md) |  | [optional] 
**parent_task_id** | **str** | ID of the parent task if the task is within a component scope. Empty if the task is at the root level. | [optional] 
**child_tasks** | [**list[PipelineTaskDetailChildTask]**](PipelineTaskDetailChildTask.md) | Sequence of dependent tasks. | [optional] 
**inputs** | [**PipelineTaskDetailInputOutputs**](PipelineTaskDetailInputOutputs.md) |  | [optional] 
**outputs** | [**PipelineTaskDetailInputOutputs**](PipelineTaskDetailInputOutputs.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


