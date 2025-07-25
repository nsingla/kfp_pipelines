package api

import (
	"fmt"
	"strings"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	pipelinev2alpha1 "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArgoWorkflowConverter provides utilities to convert KFP Pipeline Specs combined with
// compiled Argo Workflows into new Argo Workflow objects for testing and comparison purposes.
// The converter combines structural information from Pipeline Specs with execution
// details from compiled Argo Workflows to create comprehensive workflow representations.
type ArgoWorkflowConverter struct {
	namespace string
}

// NewArgoWorkflowConverter creates a new converter instance with the specified namespace.
// If namespace is empty, defaults to "kubeflow".
func NewArgoWorkflowConverter(namespace string) *ArgoWorkflowConverter {
	if namespace == "" {
		namespace = "kubeflow"
	}
	return &ArgoWorkflowConverter{
		namespace: namespace,
	}
}

// ConvertToArgoWorkflow combines information from a KFP Pipeline Spec and compiled Argo Workflow
// to create a new Argo Workflow object. This method merges:
// - Component definitions and metadata from Pipeline Spec
// - Execution flow, templates, and runtime details from compiled Argo Workflow
// - Enhanced metadata and labels for identification and comparison
// - Status nodes populated from pipeline components and workflow structure
func (c *ArgoWorkflowConverter) ConvertToArgoWorkflow(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	compiledWorkflow *argov1alpha1.Workflow,
	workflowName string,
) (*argov1alpha1.Workflow, error) {
	// Validate inputs
	if pipelineSpec == nil {
		return nil, fmt.Errorf("pipeline spec cannot be nil")
	}
	if compiledWorkflow == nil {
		return nil, fmt.Errorf("compiled workflow cannot be nil")
	}
	if workflowName == "" {
		return nil, fmt.Errorf("workflow name cannot be empty")
	}

	// Create new Argo Workflow by combining both sources
	combinedWorkflow := &argov1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.sanitizeName(workflowName),
			Namespace:   c.namespace,
			Labels:      c.createCombinedLabels(pipelineSpec, compiledWorkflow, workflowName),
			Annotations: c.createCombinedAnnotations(pipelineSpec, compiledWorkflow, workflowName),
		},
		Spec: argov1alpha1.WorkflowSpec{
			Templates: []argov1alpha1.Template{},
		},
		Status: argov1alpha1.WorkflowStatus{
			Phase: argov1alpha1.WorkflowRunning,
			Nodes: make(map[string]argov1alpha1.NodeStatus),
		},
	}

	// Copy and enhance workflow specification from compiled workflow
	c.copyWorkflowSpec(compiledWorkflow, combinedWorkflow)

	// Enhance templates with Pipeline Spec information
	err := c.enhanceTemplatesWithPipelineSpec(pipelineSpec, combinedWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance templates with pipeline spec: %w", err)
	}

	// Add additional templates for Pipeline Spec components not in compiled workflow
	err = c.addMissingComponentTemplates(pipelineSpec, combinedWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to add missing component templates: %w", err)
	}

	// Populate workflow status nodes from Pipeline Spec and compiled workflow
	err = c.populateWorkflowStatusNodes(pipelineSpec, combinedWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to populate workflow status nodes: %w", err)
	}

	return combinedWorkflow, nil
}

// createCombinedAnnotations creates comprehensive annotations combining information
// from both the Pipeline Spec and compiled Argo Workflow.
func (c *ArgoWorkflowConverter) createCombinedAnnotations(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	compiledWorkflow *argov1alpha1.Workflow,
	workflowName string,
) map[string]string {
	annotations := make(map[string]string)

	// Copy existing annotations from compiled workflow
	if compiledWorkflow.Annotations != nil {
		for key, value := range compiledWorkflow.Annotations {
			annotations[key] = value
		}
	}

	// Add KFP-specific annotations
	annotations["pipelines.kubeflow.org/run_name"] = fmt.Sprintf("Run of %s", workflowName)
	annotations["workflows.argoproj.io/pod-name-format"] = "v2"

	// Add pipeline information if available
	if pipelineSpec.PipelineInfo != nil {
		if pipelineSpec.PipelineInfo.Name != "" {
			annotations["pipelines.kubeflow.org/pipeline_name"] = pipelineSpec.PipelineInfo.Name
		}
		if pipelineSpec.PipelineInfo.Description != "" {
			annotations["pipelines.kubeflow.org/pipeline_description"] = pipelineSpec.PipelineInfo.Description
		}
	}

	// Add converter metadata
	annotations["converter.kubeflow.org/created-by"] = "argo-workflow-converter"
	annotations["converter.kubeflow.org/source"] = "pipeline-spec-and-compiled-workflow"

	return annotations
}

// createCombinedLabels creates a comprehensive set of labels combining information
// from both the Pipeline Spec and compiled Argo Workflow for identification and tracking.
func (c *ArgoWorkflowConverter) createCombinedLabels(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	compiledWorkflow *argov1alpha1.Workflow,
	workflowName string,
) map[string]string {
	labels := make(map[string]string)

	// Copy existing labels from compiled workflow
	if compiledWorkflow.Labels != nil {
		for key, value := range compiledWorkflow.Labels {
			labels[key] = value
		}
	}

	// Add Argo workflow specific labels
	labels["workflows.argoproj.io/completed"] = "false"
	labels["workflows.argoproj.io/phase"] = "Running"

	// Add KFP pipeline specific labels
	if compiledWorkflow.Labels != nil && compiledWorkflow.Labels["pipeline/runid"] != "" {
		labels["pipeline/runid"] = compiledWorkflow.Labels["pipeline/runid"]
	} else {
		// Generate a placeholder run ID if not present
		labels["pipeline/runid"] = "converted-workflow-run"
	}

	// Add converter-specific labels
	labels["pipeline-name"] = c.sanitizeLabelValue(workflowName)
	labels["converted-from"] = "kfp-pipeline-spec"
	labels["converter-type"] = "combined-spec-workflow"
	labels["component-count"] = fmt.Sprintf("%d", len(pipelineSpec.Components))

	// Add pipeline information labels if available
	if pipelineSpec.PipelineInfo != nil && pipelineSpec.PipelineInfo.Name != "" {
		labels["original-pipeline-name"] = c.sanitizeLabelValue(pipelineSpec.PipelineInfo.Name)
	}

	return labels
}

// copyWorkflowSpec copies the workflow specification from compiled workflow to the new workflow,
// including entrypoint, templates, volumes, and other execution configurations.
func (c *ArgoWorkflowConverter) copyWorkflowSpec(
	source *argov1alpha1.Workflow,
	target *argov1alpha1.Workflow,
) {
	// Copy core workflow specification
	target.Spec.Entrypoint = source.Spec.Entrypoint
	target.Spec.Arguments = source.Spec.Arguments
	target.Spec.ServiceAccountName = source.Spec.ServiceAccountName
	target.Spec.ActiveDeadlineSeconds = source.Spec.ActiveDeadlineSeconds
	target.Spec.Priority = source.Spec.Priority
	target.Spec.NodeSelector = source.Spec.NodeSelector
	target.Spec.Affinity = source.Spec.Affinity
	target.Spec.Tolerations = source.Spec.Tolerations
	target.Spec.PodMetadata = source.Spec.PodMetadata

	// Copy volumes and volume claims
	if len(source.Spec.Volumes) > 0 {
		target.Spec.Volumes = make([]k8v1.Volume, len(source.Spec.Volumes))
		copy(target.Spec.Volumes, source.Spec.Volumes)
	}

	if len(source.Spec.VolumeClaimTemplates) > 0 {
		target.Spec.VolumeClaimTemplates = make([]k8v1.PersistentVolumeClaim, len(source.Spec.VolumeClaimTemplates))
		copy(target.Spec.VolumeClaimTemplates, source.Spec.VolumeClaimTemplates)
	}

	// Copy templates (will be enhanced later)
	if len(source.Spec.Templates) > 0 {
		target.Spec.Templates = make([]argov1alpha1.Template, len(source.Spec.Templates))
		copy(target.Spec.Templates, source.Spec.Templates)
	}
}

// enhanceTemplatesWithPipelineSpec enhances existing templates in the compiled workflow
// with additional information from the Pipeline Spec, such as component metadata,
// input/output definitions, and execution details.
func (c *ArgoWorkflowConverter) enhanceTemplatesWithPipelineSpec(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	workflow *argov1alpha1.Workflow,
) error {
	// Create a mapping of component names to their specifications
	componentMap := make(map[string]*pipelinev2alpha1.ComponentSpec)
	for componentName, component := range pipelineSpec.Components {
		componentMap[componentName] = component
	}

	// Enhance each template with Pipeline Spec information
	for i := range workflow.Spec.Templates {
		template := &workflow.Spec.Templates[i]

		// Find matching component for this template
		matchingComponent := c.findMatchingComponent(template.Name, componentMap)
		if matchingComponent != nil {
			c.enhanceTemplateWithComponent(template, matchingComponent)
		}

		// Add pipeline-specific annotations
		c.addPipelineAnnotations(template, pipelineSpec)
	}

	return nil
}

// findMatchingComponent attempts to find a Pipeline Spec component that corresponds
// to the given template name using various matching strategies.
func (c *ArgoWorkflowConverter) findMatchingComponent(
	templateName string,
	componentMap map[string]*pipelinev2alpha1.ComponentSpec,
) *pipelinev2alpha1.ComponentSpec {
	// Direct name match
	if component, exists := componentMap[templateName]; exists {
		return component
	}

	// Try sanitized name match
	for componentName, component := range componentMap {
		if c.sanitizeName(componentName) == templateName {
			return component
		}
	}

	// Try partial name match for components with prefixes
	for componentName, component := range componentMap {
		sanitizedComponentName := c.sanitizeName(componentName)
		if strings.Contains(templateName, sanitizedComponentName) ||
			strings.Contains(sanitizedComponentName, templateName) {
			return component
		}
	}

	return nil
}

// enhanceTemplateWithComponent adds Pipeline Spec component information to an Argo template,
// including input/output definitions, component metadata, and execution specifications.
func (c *ArgoWorkflowConverter) enhanceTemplateWithComponent(
	template *argov1alpha1.Template,
	component *pipelinev2alpha1.ComponentSpec,
) {
	// Initialize metadata if not present
	if template.Metadata.Labels == nil {
		template.Metadata.Labels = make(map[string]string)
	}
	if template.Metadata.Annotations == nil {
		template.Metadata.Annotations = make(map[string]string)
	}

	// Add component information to template metadata
	template.Metadata.Labels["component-type"] = "kfp-component"

	// Enhance input definitions with Pipeline Spec information
	c.enhanceTemplateInputs(template, component.InputDefinitions)

	// Enhance output definitions with Pipeline Spec information
	c.enhanceTemplateOutputs(template, component.OutputDefinitions)

	// Container specifications are preserved from compiled workflow
	// to maintain execution compatibility
}

// enhanceTemplateInputs merges input definitions from Pipeline Spec with existing template inputs,
// adding type information and metadata while preserving existing Argo-specific configurations.
func (c *ArgoWorkflowConverter) enhanceTemplateInputs(
	template *argov1alpha1.Template,
	inputDefs *pipelinev2alpha1.ComponentInputsSpec,
) {
	if inputDefs == nil {
		return
	}

	// Create maps for quick lookup of existing inputs
	existingParams := make(map[string]*argov1alpha1.Parameter)
	for i := range template.Inputs.Parameters {
		param := &template.Inputs.Parameters[i]
		existingParams[param.Name] = param
	}

	existingArtifacts := make(map[string]*argov1alpha1.Artifact)
	for i := range template.Inputs.Artifacts {
		artifact := &template.Inputs.Artifacts[i]
		existingArtifacts[artifact.Name] = artifact
	}

	// Enhance existing parameters and add new ones from Pipeline Spec
	for paramName, paramSpec := range inputDefs.Parameters {
		if existingParam, exists := existingParams[paramName]; exists {
			// Enhance existing parameter with Pipeline Spec information
			c.enhanceParameter(existingParam, paramSpec)
		} else {
			// Add new parameter from Pipeline Spec
			newParam := c.createParameterFromSpec(paramName, paramSpec)
			template.Inputs.Parameters = append(template.Inputs.Parameters, newParam)
		}
	}

	// Enhance existing artifacts and add new ones from Pipeline Spec
	for artifactName, artifactSpec := range inputDefs.Artifacts {
		if existingArtifact, exists := existingArtifacts[artifactName]; exists {
			// Enhance existing artifact with Pipeline Spec information
			c.enhanceArtifact(existingArtifact, artifactSpec)
		} else {
			// Add new artifact from Pipeline Spec
			newArtifact := c.createArtifactFromSpec(artifactName, artifactSpec)
			template.Inputs.Artifacts = append(template.Inputs.Artifacts, newArtifact)
		}
	}
}

// enhanceTemplateOutputs merges output definitions from Pipeline Spec with existing template outputs.
func (c *ArgoWorkflowConverter) enhanceTemplateOutputs(
	template *argov1alpha1.Template,
	outputDefs *pipelinev2alpha1.ComponentOutputsSpec,
) {
	if outputDefs == nil {
		return
	}

	// Create maps for quick lookup of existing outputs
	existingParams := make(map[string]*argov1alpha1.Parameter)
	for i := range template.Outputs.Parameters {
		param := &template.Outputs.Parameters[i]
		existingParams[param.Name] = param
	}

	existingArtifacts := make(map[string]*argov1alpha1.Artifact)
	for i := range template.Outputs.Artifacts {
		artifact := &template.Outputs.Artifacts[i]
		existingArtifacts[artifact.Name] = artifact
	}

	// Enhance existing parameters and add new ones from Pipeline Spec
	for paramName, paramSpec := range outputDefs.Parameters {
		if existingParam, exists := existingParams[paramName]; exists {
			// Enhance existing parameter
			c.enhanceOutputParameter(existingParam, paramSpec)
		} else {
			// Add new parameter from Pipeline Spec
			newParam := c.createOutputParameterFromSpec(paramName, paramSpec)
			template.Outputs.Parameters = append(template.Outputs.Parameters, newParam)
		}
	}

	// Enhance existing artifacts and add new ones from Pipeline Spec
	for artifactName, artifactSpec := range outputDefs.Artifacts {
		if existingArtifact, exists := existingArtifacts[artifactName]; exists {
			// Enhance existing artifact
			c.enhanceOutputArtifact(existingArtifact, artifactSpec)
		} else {
			// Add new artifact from Pipeline Spec
			newArtifact := c.createOutputArtifactFromSpec(artifactName, artifactSpec)
			template.Outputs.Artifacts = append(template.Outputs.Artifacts, newArtifact)
		}
	}
}

// enhanceParameter adds Pipeline Spec parameter information to an existing Argo parameter.
func (c *ArgoWorkflowConverter) enhanceParameter(
	argoParam *argov1alpha1.Parameter,
	pipelineParam *pipelinev2alpha1.ComponentInputsSpec_ParameterSpec,
) {
	if pipelineParam.ParameterType != 0 {
		typeDesc := fmt.Sprintf("Type: %s", pipelineParam.ParameterType.String())

		// Append to existing description or create new one
		if argoParam.Description != nil {
			currentDesc := string(*argoParam.Description)
			desc := fmt.Sprintf("%s, %s", currentDesc, typeDesc)
			argoParam.Description = argov1alpha1.AnyStringPtr(desc)
		} else {
			argoParam.Description = argov1alpha1.AnyStringPtr(typeDesc)
		}
	}

	// Note default value availability without resolving the value
	if pipelineParam.DefaultValue != nil {
		currentDesc := ""
		if argoParam.Description != nil {
			currentDesc = string(*argoParam.Description)
		}
		desc := fmt.Sprintf("%s, Default available in spec", currentDesc)
		argoParam.Description = argov1alpha1.AnyStringPtr(desc)
	}
}

// createParameterFromSpec creates a new Argo parameter from a Pipeline Spec parameter.
func (c *ArgoWorkflowConverter) createParameterFromSpec(
	name string,
	pipelineParam *pipelinev2alpha1.ComponentInputsSpec_ParameterSpec,
) argov1alpha1.Parameter {
	param := argov1alpha1.Parameter{
		Name: name,
	}

	if pipelineParam.ParameterType != 0 {
		typeDesc := fmt.Sprintf("Type: %s", pipelineParam.ParameterType.String())
		param.Description = argov1alpha1.AnyStringPtr(typeDesc)
	}

	return param
}

// enhanceArtifact adds Pipeline Spec artifact information to an existing Argo artifact.
func (c *ArgoWorkflowConverter) enhanceArtifact(
	argoArtifact *argov1alpha1.Artifact,
	pipelineArtifact *pipelinev2alpha1.ComponentInputsSpec_ArtifactSpec,
) {
	// Store artifact type information in the From field for tracking
	if pipelineArtifact.ArtifactType != nil {
		typeInfo := fmt.Sprintf("Type: %s v%s",
			pipelineArtifact.ArtifactType.GetSchemaTitle(),
			pipelineArtifact.ArtifactType.GetSchemaVersion())

		if argoArtifact.From != "" {
			argoArtifact.From = fmt.Sprintf("%s, %s", argoArtifact.From, typeInfo)
		} else {
			argoArtifact.From = typeInfo
		}
	}
}

// createArtifactFromSpec creates a new Argo artifact from a Pipeline Spec artifact.
func (c *ArgoWorkflowConverter) createArtifactFromSpec(
	name string,
	pipelineArtifact *pipelinev2alpha1.ComponentInputsSpec_ArtifactSpec,
) argov1alpha1.Artifact {
	artifact := argov1alpha1.Artifact{
		Name: name,
	}

	// Store type information in From field
	if pipelineArtifact.ArtifactType != nil {
		artifact.From = fmt.Sprintf("Type: %s v%s",
			pipelineArtifact.ArtifactType.GetSchemaTitle(),
			pipelineArtifact.ArtifactType.GetSchemaVersion())
	}

	return artifact
}

// addMissingComponentTemplates creates new templates for Pipeline Spec components
// that don't have corresponding templates in the compiled workflow.
func (c *ArgoWorkflowConverter) addMissingComponentTemplates(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	workflow *argov1alpha1.Workflow,
) error {
	// Create a set of existing template names for quick lookup
	existingTemplates := make(map[string]bool)
	for _, template := range workflow.Spec.Templates {
		existingTemplates[template.Name] = true

		// Also check for partial matches to avoid creating duplicates
		for componentName := range pipelineSpec.Components {
			sanitizedComponentName := c.sanitizeName(componentName)
			if strings.Contains(template.Name, sanitizedComponentName) ||
				strings.Contains(sanitizedComponentName, template.Name) {
				existingTemplates[sanitizedComponentName] = true
			}
		}
	}

	// Check each component and create missing templates
	for componentName, component := range pipelineSpec.Components {
		sanitizedName := c.sanitizeName(componentName)

		// Skip if template already exists
		if existingTemplates[sanitizedName] {
			continue
		}

		// Create template for missing component
		newTemplate := c.createTemplateFromComponent(componentName, component)
		workflow.Spec.Templates = append(workflow.Spec.Templates, newTemplate)
	}

	return nil
}

// createTemplateFromComponent creates a new Argo template from a Pipeline Spec component.
func (c *ArgoWorkflowConverter) createTemplateFromComponent(
	componentName string,
	component *pipelinev2alpha1.ComponentSpec,
) argov1alpha1.Template {
	template := argov1alpha1.Template{
		Name: c.sanitizeName(componentName),
		Metadata: argov1alpha1.Metadata{
			Labels: map[string]string{
				"component-name": componentName,
				"component-type": "kfp-component",
				"created-by":     "converter",
			},
		},
	}

	// Add input and output definitions from component spec
	template.Inputs = c.convertInputDefinitions(component.InputDefinitions)
	template.Outputs = c.convertOutputDefinitions(component.OutputDefinitions)

	return template
}

// convertInputDefinitions converts Pipeline Spec input definitions to Argo input format.
func (c *ArgoWorkflowConverter) convertInputDefinitions(
	inputDefs *pipelinev2alpha1.ComponentInputsSpec,
) argov1alpha1.Inputs {
	inputs := argov1alpha1.Inputs{}

	if inputDefs != nil {
		// Convert parameter definitions
		for paramName, paramSpec := range inputDefs.Parameters {
			param := c.createParameterFromSpec(paramName, paramSpec)
			inputs.Parameters = append(inputs.Parameters, param)
		}

		// Convert artifact definitions
		for artifactName, artifactSpec := range inputDefs.Artifacts {
			artifact := c.createArtifactFromSpec(artifactName, artifactSpec)
			inputs.Artifacts = append(inputs.Artifacts, artifact)
		}
	}

	return inputs
}

// convertOutputDefinitions converts Pipeline Spec output definitions to Argo output format.
func (c *ArgoWorkflowConverter) convertOutputDefinitions(
	outputDefs *pipelinev2alpha1.ComponentOutputsSpec,
) argov1alpha1.Outputs {
	outputs := argov1alpha1.Outputs{}

	if outputDefs != nil {
		// Convert parameter definitions
		for paramName, paramSpec := range outputDefs.Parameters {
			param := argov1alpha1.Parameter{
				Name: paramName,
			}
			if paramSpec.ParameterType != 0 {
				typeDesc := fmt.Sprintf("Type: %s", paramSpec.ParameterType.String())
				param.Description = argov1alpha1.AnyStringPtr(typeDesc)
			}
			outputs.Parameters = append(outputs.Parameters, param)
		}

		// Convert artifact definitions
		for artifactName, artifactSpec := range outputDefs.Artifacts {
			artifact := argov1alpha1.Artifact{
				Name: artifactName,
			}
			if artifactSpec.ArtifactType != nil {
				artifact.From = fmt.Sprintf("Type: %s v%s",
					artifactSpec.ArtifactType.GetSchemaTitle(),
					artifactSpec.ArtifactType.GetSchemaVersion())
			}
			outputs.Artifacts = append(outputs.Artifacts, artifact)
		}
	}

	return outputs
}

// addPipelineAnnotations adds pipeline-specific annotations to templates for tracking and debugging.
func (c *ArgoWorkflowConverter) addPipelineAnnotations(
	template *argov1alpha1.Template,
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
) {
	if template.Metadata.Annotations == nil {
		template.Metadata.Annotations = make(map[string]string)
	}

	// Add pipeline information
	if pipelineSpec.PipelineInfo != nil {
		if pipelineSpec.PipelineInfo.Name != "" {
			template.Metadata.Annotations["pipeline.kubeflow.org/pipeline-name"] = pipelineSpec.PipelineInfo.Name
		}
		if pipelineSpec.PipelineInfo.Description != "" {
			template.Metadata.Annotations["pipeline.kubeflow.org/pipeline-description"] = pipelineSpec.PipelineInfo.Description
		}
	}

	// Add converter metadata
	template.Metadata.Annotations["converter.kubeflow.org/enhanced-by"] = "argo-workflow-converter"
	template.Metadata.Annotations["converter.kubeflow.org/component-count"] = fmt.Sprintf("%d", len(pipelineSpec.Components))
}

// sanitizeName ensures the name is valid for Argo Workflow templates.
// Argo template names must be DNS-1123 compliant (lowercase, alphanumeric, hyphens).
func (c *ArgoWorkflowConverter) sanitizeName(name string) string {
	var result strings.Builder

	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z':
			result.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			result.WriteRune(char + 32) // Convert to lowercase
		case char >= '0' && char <= '9':
			result.WriteRune(char)
		case char == '-' || char == '_':
			result.WriteRune('-')
		case char == '.' || char == ' ':
			result.WriteRune('-')
		default:
			// Skip invalid characters
		}
	}

	sanitized := result.String()

	// Ensure it doesn't start or end with hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// sanitizeLabelValue ensures the value is valid for Kubernetes labels.
func (c *ArgoWorkflowConverter) sanitizeLabelValue(value string) string {
	if len(value) > 63 {
		value = value[:63]
	}
	return c.sanitizeName(value)
}

// enhanceOutputParameter adds Pipeline Spec parameter information to an existing Argo parameter.
func (c *ArgoWorkflowConverter) enhanceOutputParameter(
	argoParam *argov1alpha1.Parameter,
	pipelineParam *pipelinev2alpha1.ComponentOutputsSpec_ParameterSpec,
) {
	if pipelineParam.ParameterType != 0 {
		typeDesc := fmt.Sprintf("Type: %s", pipelineParam.ParameterType.String())
		argoParam.Description = argov1alpha1.AnyStringPtr(typeDesc)
	}
}

// createOutputParameterFromSpec creates a new Argo parameter from a Pipeline Spec parameter.
func (c *ArgoWorkflowConverter) createOutputParameterFromSpec(
	name string,
	pipelineParam *pipelinev2alpha1.ComponentOutputsSpec_ParameterSpec,
) argov1alpha1.Parameter {
	param := argov1alpha1.Parameter{Name: name}
	if pipelineParam.ParameterType != 0 {
		typeDesc := fmt.Sprintf("Type: %s", pipelineParam.ParameterType.String())
		param.Description = argov1alpha1.AnyStringPtr(typeDesc)
	}
	return param
}

// enhanceOutputArtifact adds Pipeline Spec output artifact information to an existing Argo artifact.
func (c *ArgoWorkflowConverter) enhanceOutputArtifact(
	argoArtifact *argov1alpha1.Artifact,
	pipelineArtifact *pipelinev2alpha1.ComponentOutputsSpec_ArtifactSpec,
) {
	// Store artifact type information in the From field for tracking
	if pipelineArtifact.ArtifactType != nil {
		typeInfo := fmt.Sprintf("Type: %s v%s",
			pipelineArtifact.ArtifactType.GetSchemaTitle(),
			pipelineArtifact.ArtifactType.GetSchemaVersion())

		if argoArtifact.From != "" {
			argoArtifact.From = fmt.Sprintf("%s, %s", argoArtifact.From, typeInfo)
		} else {
			argoArtifact.From = typeInfo
		}
	}
}

// createOutputArtifactFromSpec creates a new Argo artifact from a Pipeline Spec output artifact.
func (c *ArgoWorkflowConverter) createOutputArtifactFromSpec(
	name string,
	pipelineArtifact *pipelinev2alpha1.ComponentOutputsSpec_ArtifactSpec,
) argov1alpha1.Artifact {
	artifact := argov1alpha1.Artifact{
		Name: name,
	}

	// Store type information in From field
	if pipelineArtifact.ArtifactType != nil {
		artifact.From = fmt.Sprintf("Type: %s v%s",
			pipelineArtifact.ArtifactType.GetSchemaTitle(),
			pipelineArtifact.ArtifactType.GetSchemaVersion())
	}

	return artifact
}

// populateWorkflowStatusNodes creates status nodes for each template and component
// combining information from Pipeline Spec and compiled workflow structure.
func (c *ArgoWorkflowConverter) populateWorkflowStatusNodes(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	workflow *argov1alpha1.Workflow,
) error {
	// Initialize nodes map if not present
	if workflow.Status.Nodes == nil {
		workflow.Status.Nodes = make(map[string]argov1alpha1.NodeStatus)
	}

	// Create nodes for each template in the workflow
	for _, template := range workflow.Spec.Templates {
		nodeID := c.generateNodeID(template.Name)
		node := c.createNodeFromTemplate(template, pipelineSpec)
		workflow.Status.Nodes[nodeID] = node
	}

	// Create additional nodes for DAG tasks that reference other templates
	c.createDAGTaskNodes(workflow)

	// Establish node relationships based on DAG dependencies
	c.establishNodeRelationships(workflow)

	return nil
}

// createDAGTaskNodes creates individual nodes for each task within DAG templates.
func (c *ArgoWorkflowConverter) createDAGTaskNodes(workflow *argov1alpha1.Workflow) {
	for _, template := range workflow.Spec.Templates {
		if template.DAG != nil {
			c.processDAGTasks(template, workflow)
		}
	}
}

// processDAGTasks creates nodes for individual tasks within a DAG template.
func (c *ArgoWorkflowConverter) processDAGTasks(dagTemplate argov1alpha1.Template, workflow *argov1alpha1.Workflow) {
	for _, task := range dagTemplate.DAG.Tasks {
		// Create a unique node ID for the task
		taskNodeID := c.generateTaskNodeID(dagTemplate.Name, task.Name)

		// Skip if node already exists
		if _, exists := workflow.Status.Nodes[taskNodeID]; exists {
			continue
		}

		// Create node for the task
		taskNode := argov1alpha1.NodeStatus{
			ID:           taskNodeID,
			Name:         task.Name,
			DisplayName:  task.Name,
			Type:         c.determineTaskNodeType(task),
			TemplateName: task.Template,
			Phase:        argov1alpha1.NodePending,
		}

		// Add task-specific information
		if len(task.Arguments.Parameters) > 0 {
			taskNode.Message = fmt.Sprintf("Task with %d parameters", len(task.Arguments.Parameters))
		}

		// Handle dependencies
		if task.Dependencies != nil && len(task.Dependencies) > 0 {
			taskNode.Message += fmt.Sprintf(", depends on: %s", strings.Join(task.Dependencies, ", "))
		}

		workflow.Status.Nodes[taskNodeID] = taskNode
	}
}

// determineTaskNodeType determines the node type for a DAG task.
func (c *ArgoWorkflowConverter) determineTaskNodeType(task argov1alpha1.DAGTask) argov1alpha1.NodeType {
	// Check for specific patterns in template names
	if strings.Contains(task.Template, "driver") {
		return argov1alpha1.NodeTypePod
	}
	if strings.Contains(task.Template, "executor") {
		return argov1alpha1.NodeTypePod
	}
	if strings.Contains(task.Template, "system-") {
		return argov1alpha1.NodeTypePod
	}

	// Default to Pod type for most tasks
	return argov1alpha1.NodeTypePod
}

// generateTaskNodeID creates a unique node ID for a task within a DAG.
func (c *ArgoWorkflowConverter) generateTaskNodeID(dagName, taskName string) string {
	return fmt.Sprintf("%s.%s", c.sanitizeName(dagName), c.sanitizeName(taskName))
}

// createNodeFromTemplate creates a workflow node status from a template and Pipeline Spec information.
func (c *ArgoWorkflowConverter) createNodeFromTemplate(
	template argov1alpha1.Template,
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
) argov1alpha1.NodeStatus {
	nodeStatus := argov1alpha1.NodeStatus{
		ID:           c.generateNodeID(template.Name),
		Name:         template.Name,
		DisplayName:  template.Name,
		Type:         c.determineNodeType(template),
		TemplateName: template.Name,
		Phase:        argov1alpha1.NodePending, // Default to pending for conversion
	}

	// Add component-specific information from Pipeline Spec
	matchingComponent := c.findMatchingComponentForTemplate(template.Name, pipelineSpec)
	if matchingComponent != nil {
		c.enhanceNodeWithComponentInfo(&nodeStatus, matchingComponent, template.Name)
	}

	// Set node-specific metadata
	nodeStatus.Outputs = c.createNodeOutputs(template)
	nodeStatus.Inputs = c.createNodeInputs(template)

	return nodeStatus
}

// determineNodeType determines the appropriate node type based on template structure.
func (c *ArgoWorkflowConverter) determineNodeType(template argov1alpha1.Template) argov1alpha1.NodeType {
	// Check if it's a DAG template
	if template.DAG != nil {
		return argov1alpha1.NodeTypeDAG
	}

	// Check if it's a container template
	if template.Container != nil {
		return argov1alpha1.NodeTypePod
	}

	// Check if it has steps
	if template.Steps != nil && len(template.Steps) > 0 {
		return argov1alpha1.NodeTypeSteps
	}

	// Check for specific system templates
	if strings.Contains(template.Name, "driver") {
		return argov1alpha1.NodeTypePod
	}

	if strings.Contains(template.Name, "executor") {
		return argov1alpha1.NodeTypePod
	}

	// Default to Task type for KFP components
	return argov1alpha1.NodeTypePod
}

// findMatchingComponentForTemplate finds a Pipeline Spec component matching the template.
func (c *ArgoWorkflowConverter) findMatchingComponentForTemplate(
	templateName string,
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
) *pipelinev2alpha1.ComponentSpec {
	// Create component map for lookup
	componentMap := make(map[string]*pipelinev2alpha1.ComponentSpec)
	for componentName, component := range pipelineSpec.Components {
		componentMap[componentName] = component
	}

	return c.findMatchingComponent(templateName, componentMap)
}

// enhanceNodeWithComponentInfo adds Pipeline Spec component information to a node status.
func (c *ArgoWorkflowConverter) enhanceNodeWithComponentInfo(
	nodeStatus *argov1alpha1.NodeStatus,
	component *pipelinev2alpha1.ComponentSpec,
	originalComponentName string,
) {
	// Set display name to original component name
	nodeStatus.DisplayName = originalComponentName

	// Add component metadata as node message
	if component.InputDefinitions != nil {
		paramCount := len(component.InputDefinitions.Parameters)
		artifactCount := len(component.InputDefinitions.Artifacts)
		nodeStatus.Message = fmt.Sprintf("KFP Component: %d parameters, %d artifacts", paramCount, artifactCount)
	}
}

// createNodeOutputs creates node outputs based on template output definitions.
func (c *ArgoWorkflowConverter) createNodeOutputs(template argov1alpha1.Template) *argov1alpha1.Outputs {
	if len(template.Outputs.Parameters) == 0 && len(template.Outputs.Artifacts) == 0 {
		return nil
	}

	outputs := &argov1alpha1.Outputs{
		Parameters: make([]argov1alpha1.Parameter, len(template.Outputs.Parameters)),
		Artifacts:  make([]argov1alpha1.Artifact, len(template.Outputs.Artifacts)),
	}

	// Copy parameters
	copy(outputs.Parameters, template.Outputs.Parameters)

	// Copy artifacts
	copy(outputs.Artifacts, template.Outputs.Artifacts)

	return outputs
}

// createNodeInputs creates node inputs based on template input definitions.
func (c *ArgoWorkflowConverter) createNodeInputs(template argov1alpha1.Template) *argov1alpha1.Inputs {
	if len(template.Inputs.Parameters) == 0 && len(template.Inputs.Artifacts) == 0 {
		return nil
	}

	inputs := &argov1alpha1.Inputs{
		Parameters: make([]argov1alpha1.Parameter, len(template.Inputs.Parameters)),
		Artifacts:  make([]argov1alpha1.Artifact, len(template.Inputs.Artifacts)),
	}

	// Copy parameters
	copy(inputs.Parameters, template.Inputs.Parameters)

	// Copy artifacts
	copy(inputs.Artifacts, template.Inputs.Artifacts)

	return inputs
}

// addMissingComponentNodes creates nodes for Pipeline Spec components not represented in workflow templates.
func (c *ArgoWorkflowConverter) addMissingComponentNodes(
	pipelineSpec *pipelinev2alpha1.PipelineSpec,
	workflow *argov1alpha1.Workflow,
) {
	// Get existing template names
	existingTemplates := make(map[string]bool)
	for _, template := range workflow.Spec.Templates {
		existingTemplates[template.Name] = true
	}

	// Check each component
	for componentName, component := range pipelineSpec.Components {
		sanitizedName := c.sanitizeName(componentName)

		// Skip if already has a template/node
		if existingTemplates[sanitizedName] {
			continue
		}

		// Check if already has a partial match
		hasMatch := false
		for templateName := range existingTemplates {
			if strings.Contains(templateName, sanitizedName) || strings.Contains(sanitizedName, templateName) {
				hasMatch = true
				break
			}
		}

		if !hasMatch {
			// Create node for missing component
			nodeID := c.generateNodeID(sanitizedName)
			nodeStatus := c.createNodeFromComponent(componentName, component)
			workflow.Status.Nodes[nodeID] = nodeStatus
		}
	}
}

// createNodeFromComponent creates a node status directly from a Pipeline Spec component.
func (c *ArgoWorkflowConverter) createNodeFromComponent(
	componentName string,
	component *pipelinev2alpha1.ComponentSpec,
) argov1alpha1.NodeStatus {
	nodeStatus := argov1alpha1.NodeStatus{
		ID:           c.generateNodeID(c.sanitizeName(componentName)),
		Name:         c.sanitizeName(componentName),
		DisplayName:  componentName,
		Type:         argov1alpha1.NodeTypePod, // Default to Pod for KFP components
		TemplateName: c.sanitizeName(componentName),
		Phase:        argov1alpha1.NodePending,
		Message:      "KFP Component (spec-only)",
	}

	// Add component-specific information
	if component.InputDefinitions != nil {
		paramCount := len(component.InputDefinitions.Parameters)
		artifactCount := len(component.InputDefinitions.Artifacts)
		nodeStatus.Message = fmt.Sprintf("KFP Component (spec-only): %d parameters, %d artifacts", paramCount, artifactCount)
	}

	return nodeStatus
}

// establishNodeRelationships sets up parent-child relationships between nodes based on DAG structure.
func (c *ArgoWorkflowConverter) establishNodeRelationships(workflow *argov1alpha1.Workflow) {
	// Find DAG templates and establish relationships
	for _, template := range workflow.Spec.Templates {
		if template.DAG != nil {
			c.processDAGRelationships(template, workflow)
		}
	}
}

// processDAGRelationships processes DAG tasks to establish node dependencies.
func (c *ArgoWorkflowConverter) processDAGRelationships(dagTemplate argov1alpha1.Template, workflow *argov1alpha1.Workflow) {
	dagNodeID := c.generateNodeID(dagTemplate.Name)

	// Process each task in the DAG
	for _, task := range dagTemplate.DAG.Tasks {
		taskNodeID := c.generateTaskNodeID(dagTemplate.Name, task.Name)

		// Find the task node
		if taskNode, exists := workflow.Status.Nodes[taskNodeID]; exists {
			// Add DAG as parent if DAG node exists
			if dagNode, dagExists := workflow.Status.Nodes[dagNodeID]; dagExists {
				// Add task as child of DAG
				if dagNode.Children == nil {
					dagNode.Children = []string{}
				}
				dagNode.Children = append(dagNode.Children, taskNodeID)
				workflow.Status.Nodes[dagNodeID] = dagNode
			}

			// Handle dependencies between tasks
			if task.Dependencies != nil {
				for _, dep := range task.Dependencies {
					depNodeID := c.generateTaskNodeID(dagTemplate.Name, dep)
					if depNode, depExists := workflow.Status.Nodes[depNodeID]; depExists {
						// Add current task as child of dependency
						if depNode.Children == nil {
							depNode.Children = []string{}
						}
						depNode.Children = append(depNode.Children, taskNodeID)
						workflow.Status.Nodes[depNodeID] = depNode
					}
				}
			}

			workflow.Status.Nodes[taskNodeID] = taskNode
		}
	}
}

// generateNodeID creates a node ID for the workflow status.
func (c *ArgoWorkflowConverter) generateNodeID(templateName string) string {
	return fmt.Sprintf("%s", strings.ReplaceAll(templateName, "_", "-"))
}
