# kfp_server_api.ArtifactServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_artifact**](ArtifactServiceApi.md#create_artifact) | **POST** /apis/v2beta1/artifacts | Creates a new artifact.
[**get_artifact**](ArtifactServiceApi.md#get_artifact) | **GET** /apis/v2beta1/artifacts/{artifact_id} | Finds a specific Artifact by ID.
[**get_metric**](ArtifactServiceApi.md#get_metric) | **GET** /apis/v2beta1/metrics/{task_id}/{name} | Gets a metric by task ID and name.
[**list_artifact_tasks**](ArtifactServiceApi.md#list_artifact_tasks) | **GET** /apis/v2beta1/artifact_tasks | Lists artifact-task relationships.
[**list_artifacts**](ArtifactServiceApi.md#list_artifacts) | **GET** /apis/v2beta1/artifacts | Finds all artifacts within the specified namespace.
[**list_metrics**](ArtifactServiceApi.md#list_metrics) | **GET** /apis/v2beta1/metrics | Lists all metrics.
[**log_metric**](ArtifactServiceApi.md#log_metric) | **POST** /apis/v2beta1/metrics | Logs a metric for a specific task.
[**update_artifact**](ArtifactServiceApi.md#update_artifact) | **PUT** /apis/v2beta1/artifacts/{artifact.artifact_id} | Updates an existing artifact.


# **create_artifact**
> V2beta1Artifact create_artifact(body)

Creates a new artifact.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    body = kfp_server_api.V2beta1CreateArtifactRequest() # V2beta1CreateArtifactRequest | 

    try:
        # Creates a new artifact.
        api_response = api_instance.create_artifact(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->create_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1CreateArtifactRequest**](V2beta1CreateArtifactRequest.md)|  | 

### Return type

[**V2beta1Artifact**](V2beta1Artifact.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_artifact**
> V2beta1Artifact get_artifact(artifact_id)

Finds a specific Artifact by ID.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    artifact_id = 'artifact_id_example' # str | Required. The ID of the artifact to be retrieved.

    try:
        # Finds a specific Artifact by ID.
        api_response = api_instance.get_artifact(artifact_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **artifact_id** | **str**| Required. The ID of the artifact to be retrieved. | 

### Return type

[**V2beta1Artifact**](V2beta1Artifact.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_metric**
> V2beta1Metric get_metric(task_id, name)

Gets a metric by task ID and name.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    task_id = 'task_id_example' # str | Required. Task UUID that owns this metric
name = 'name_example' # str | Required. Name of the metric

    try:
        # Gets a metric by task ID and name.
        api_response = api_instance.get_metric(task_id, name)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_metric: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task_id** | **str**| Required. Task UUID that owns this metric | 
 **name** | **str**| Required. Name of the metric | 

### Return type

[**V2beta1Metric**](V2beta1Metric.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_artifact_tasks**
> V2beta1ListArtifactTasksResponse list_artifact_tasks(task_ids=task_ids, run_ids=run_ids, artifact_ids=artifact_ids, type=type, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Lists artifact-task relationships.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    task_ids = ['task_ids_example'] # list[str] | Optional, filter artifact task by a set of task_ids We can also likely just rely on filter for this and omit this field (optional)
run_ids = ['run_ids_example'] # list[str] | Optional input, filter artifact task by a set of run_ids (optional)
artifact_ids = ['artifact_ids_example'] # list[str] | Optional, filter artifact task by a set of artifact_ids We can also likely just rely on filter for this and omit this field (optional)
type = 'INPUT' # str | Optional. Only list artifact tasks that have artifacts of this type. (optional) (default to 'INPUT')
page_token = 'page_token_example' # str |  (optional)
page_size = 56 # int |  (optional)
sort_by = 'sort_by_example' # str |  (optional)
filter = 'filter_example' # str |  (optional)

    try:
        # Lists artifact-task relationships.
        api_response = api_instance.list_artifact_tasks(task_ids=task_ids, run_ids=run_ids, artifact_ids=artifact_ids, type=type, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->list_artifact_tasks: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task_ids** | [**list[str]**](str.md)| Optional, filter artifact task by a set of task_ids We can also likely just rely on filter for this and omit this field | [optional] 
 **run_ids** | [**list[str]**](str.md)| Optional input, filter artifact task by a set of run_ids | [optional] 
 **artifact_ids** | [**list[str]**](str.md)| Optional, filter artifact task by a set of artifact_ids We can also likely just rely on filter for this and omit this field | [optional] 
 **type** | **str**| Optional. Only list artifact tasks that have artifacts of this type. | [optional] [default to &#39;INPUT&#39;]
 **page_token** | **str**|  | [optional] 
 **page_size** | **int**|  | [optional] 
 **sort_by** | **str**|  | [optional] 
 **filter** | **str**|  | [optional] 

### Return type

[**V2beta1ListArtifactTasksResponse**](V2beta1ListArtifactTasksResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_artifacts**
> V2beta1ListArtifactResponse list_artifacts(namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Finds all artifacts within the specified namespace.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    namespace = 'namespace_example' # str | Optional input. Namespace for the artifacts. (optional)
page_token = 'page_token_example' # str | A page token to request the results page. (optional)
page_size = 56 # int | The number of artifacts to be listed per page. If there are more artifacts than this number, the response message will contain a valid value in the nextPageToken field. (optional)
sort_by = 'sort_by_example' # str | Sorting order in form of \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/artifacts/blob/master/backend/api/filter.proto)). (optional)

    try:
        # Finds all artifacts within the specified namespace.
        api_response = api_instance.list_artifacts(namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->list_artifacts: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Optional input. Namespace for the artifacts. | [optional] 
 **page_token** | **str**| A page token to request the results page. | [optional] 
 **page_size** | **int**| The number of artifacts to be listed per page. If there are more artifacts than this number, the response message will contain a valid value in the nextPageToken field. | [optional] 
 **sort_by** | **str**| Sorting order in form of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot;. Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/artifacts/blob/master/backend/api/filter.proto)). | [optional] 

### Return type

[**V2beta1ListArtifactResponse**](V2beta1ListArtifactResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_metrics**
> V2beta1ListMetricsResponse list_metrics(task_ids=task_ids, run_ids=run_ids, namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Lists all metrics.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    task_ids = ['task_ids_example'] # list[str] | Optional input, filter metrics by a set of task_ids (optional)
run_ids = ['run_ids_example'] # list[str] | Optional input, filter metrics by a set of run_ids (optional)
namespace = 'namespace_example' # str | Optional input. Namespace for the metrics. (optional)
page_token = 'page_token_example' # str | A page token to request the results page. (optional)
page_size = 56 # int | The number of metrics to be listed per page. (optional)
sort_by = 'sort_by_example' # str | Sorting order in form of \"field_name\", \"field_name asc\" or \"field_name desc\". (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized filter protocol buffer. (optional)

    try:
        # Lists all metrics.
        api_response = api_instance.list_metrics(task_ids=task_ids, run_ids=run_ids, namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->list_metrics: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task_ids** | [**list[str]**](str.md)| Optional input, filter metrics by a set of task_ids | [optional] 
 **run_ids** | [**list[str]**](str.md)| Optional input, filter metrics by a set of run_ids | [optional] 
 **namespace** | **str**| Optional input. Namespace for the metrics. | [optional] 
 **page_token** | **str**| A page token to request the results page. | [optional] 
 **page_size** | **int**| The number of metrics to be listed per page. | [optional] 
 **sort_by** | **str**| Sorting order in form of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot;. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized filter protocol buffer. | [optional] 

### Return type

[**V2beta1ListMetricsResponse**](V2beta1ListMetricsResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **log_metric**
> V2beta1Metric log_metric(body)

Logs a metric for a specific task.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    body = kfp_server_api.V2beta1LogMetricRequest() # V2beta1LogMetricRequest | 

    try:
        # Logs a metric for a specific task.
        api_response = api_instance.log_metric(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->log_metric: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1LogMetricRequest**](V2beta1LogMetricRequest.md)|  | 

### Return type

[**V2beta1Metric**](V2beta1Metric.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_artifact**
> V2beta1Artifact update_artifact(artifact_artifact_id, artifact)

Updates an existing artifact.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    artifact_artifact_id = 'artifact_artifact_id_example' # str | Output only. The unique server generated id of the artifact. Note: Updated id name to be consistent with other api naming patterns (with prefix)
artifact = kfp_server_api.RequiredTheArtifactToUpdateTheArtifactIdFieldIsRequired() # RequiredTheArtifactToUpdateTheArtifactIdFieldIsRequired | 

    try:
        # Updates an existing artifact.
        api_response = api_instance.update_artifact(artifact_artifact_id, artifact)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->update_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **artifact_artifact_id** | **str**| Output only. The unique server generated id of the artifact. Note: Updated id name to be consistent with other api naming patterns (with prefix) | 
 **artifact** | [**RequiredTheArtifactToUpdateTheArtifactIdFieldIsRequired**](RequiredTheArtifactToUpdateTheArtifactIdFieldIsRequired.md)|  | 

### Return type

[**V2beta1Artifact**](V2beta1Artifact.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

