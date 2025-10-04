package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopePath(t *testing.T) {
	// Load pipeline spec
	pipelineSpec, err := LoadPipelineSpecFromYAML("test_data/loop_collected.py.yaml")
	require.NoError(t, err)
	require.NotNil(t, pipelineSpec)

	scopePath := NewScopePath(pipelineSpec)
	require.NotNil(t, scopePath)

	require.Empty(t, scopePath.StringPath())

	err = scopePath.Push("not-root")
	require.Error(t, err)

	err = scopePath.Push("root")
	require.NoError(t, err)
	require.NotNil(t, scopePath)

	head := scopePath.GetRoot()
	last := scopePath.GetLast()
	require.Equal(t, head, last)
	require.NotNil(t, head)
	require.NotNil(t, head.GetComponentSpec())
	require.Nil(t, head.GetTaskSpec())
	require.Equal(t, 2, len(head.componentSpec.GetDag().GetTasks()))
	require.NotNil(t, head.componentSpec.GetDag().GetTasks()["analyze-artifact-list"])

	err = scopePath.Push("secondary-pipeline")
	last = scopePath.GetLast()
	require.NotEqual(t, head, last)
	require.NoError(t, err)
	require.NotNil(t, last.GetComponentSpec())
	require.NotNil(t, last.GetTaskSpec())

	err = scopePath.Push("does-not-exist")
	require.Error(t, err)

	err = scopePath.Push("for-loop-2")
	require.NoError(t, err)
	last = scopePath.GetLast()
	require.Equal(t, last.GetTaskSpec().GetTaskInfo().GetName(), "for-loop-2")
	require.Len(t, last.GetComponentSpec().GetDag().GetTasks(), 2)

	require.Equal(t, []string{"root", "secondary-pipeline", "for-loop-2"}, scopePath.StringPath())

	spe, ok := scopePath.Pop()
	require.True(t, ok)
	require.Equal(t, spe.GetTaskSpec().GetTaskInfo().GetName(), "for-loop-2")

	spe, ok = scopePath.Pop()
	require.True(t, ok)
	require.Equal(t, "secondary-pipeline", spe.GetTaskSpec().GetTaskInfo().GetName())

	require.Equal(t, []string{"root"}, scopePath.StringPath())

	// Back to the head
	spe, ok = scopePath.Pop()
	require.True(t, ok)
	require.NotNil(t, head)
	require.NotNil(t, head.GetComponentSpec())
	require.Nil(t, head.GetTaskSpec())

	spe, ok = scopePath.Pop()
	require.False(t, ok)
	require.Empty(t, spe)

	require.Empty(t, scopePath.StringPath())
}
