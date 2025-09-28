package resolver

//
//func resolveInputArtifactV3(
//	ctx context.Context,
//	opts driver.Options,
//	name string,
//	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
//	inputArtifacts map[string]*pipelinespec.ArtifactList,
//	task *pipelinespec.PipelineTaskSpec,
//) (*pipelinespec.ArtifactList, error) {
//	artifactError := func(err error) error {
//		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
//	}
//	switch t := artifactSpec.Kind.(type) {
//	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
//		// TODO(HumairAK)
//		return nil, fmt.Errorf("not implemented yet")
//	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
//		cfg := resolveUpstreamOutputsConfig{
//			ctx:          ctx,
//			artifactSpec: artifactSpec,
//			opts:         opts,
//			err:          artifactError,
//		}
//		artifacts, err := resolveUpstreamArtifacts(cfg)
//		if err != nil {
//			return nil, err
//		}
//		return artifacts, nil
//	default:
//		return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
//	}
//}
