package driver

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
)

type ScopePath struct {
	list         *LinkedList[ScopePathEntry]
	pipelineSpec *pipelinespec.PipelineSpec
}
type ScopePathEntry struct {
	taskName      string
	taskSpec      *pipelinespec.PipelineTaskSpec
	componentSpec *pipelinespec.ComponentSpec
}

func (e *ScopePathEntry) GetTaskSpec() *pipelinespec.PipelineTaskSpec {
	return e.taskSpec
}

func (e *ScopePathEntry) GetComponentSpec() *pipelinespec.ComponentSpec {
	return e.componentSpec
}

func NewScopePath(
	pipelineSpec *pipelinespec.PipelineSpec,
) ScopePath {
	return ScopePath{
		pipelineSpec: pipelineSpec,
	}
}

func (s *ScopePath) Push(taskName string) error {
	if s.list == nil {
		s.list = &LinkedList[ScopePathEntry]{}
	}
	if taskName == "root" {
		sp := ScopePathEntry{
			taskName:      taskName,
			componentSpec: s.pipelineSpec.Root,
		}
		s.list.append(sp)
		return nil
	}
	if s.list.head == nil {
		return fmt.Errorf("scope path is empty, first task should be root")
	}
	if s.list.head.Value.componentSpec.GetDag() == nil {
		return fmt.Errorf("this component is not a DAG component")
	}
	lastTask := s.GetLast()
	if lastTask == nil {
		return fmt.Errorf("last task is nil")
	}
	if _, ok := lastTask.componentSpec.GetDag().Tasks[taskName]; !ok {
		return fmt.Errorf("task %s is not found", taskName)
	}
	taskSpec := lastTask.componentSpec.GetDag().Tasks[taskName]
	if _, ok := s.pipelineSpec.Components[taskSpec.GetComponentRef().GetName()]; !ok {
		return fmt.Errorf("component %s is not found", taskSpec.GetComponentRef().GetName())
	}
	componentSpec := s.pipelineSpec.Components[taskSpec.GetComponentRef().GetName()]
	sp := ScopePathEntry{
		taskName:      taskName,
		taskSpec:      taskSpec,
		componentSpec: componentSpec,
	}
	s.list.append(sp)
	return nil
}

func (s *ScopePath) Pop() (ScopePathEntry, bool) {
	return s.list.pop()
}

func (s *ScopePath) GetRoot() *ScopePathEntry {
	return &s.list.head.Value
}

func (s *ScopePath) GetLast() *ScopePathEntry {
	spe, ok := s.list.last()
	if !ok {
		return nil
	}
	return &spe
}

func (s *ScopePath) StringPath() []string {
	var path []string
	if s.list == nil {
		return path
	}
	for n := s.list.head; n != nil; n = n.Next {
		path = append(path, n.Value.taskName)
	}
	return path
}

// Node represents one element in the list.
type Node[T any] struct {
	Value T
	Next  *Node[T]
}

// LinkedList is a simple singly linked list.
type LinkedList[T any] struct {
	head *Node[T]
}

// append adds a new node to the end of the list.
func (l *LinkedList[T]) append(v T) {
	newNode := &Node[T]{Value: v}
	if l.head == nil {
		l.head = newNode
		return
	}
	curr := l.head
	for curr.Next != nil {
		curr = curr.Next
	}
	curr.Next = newNode
}

// pop removes and returns the last element.
// Returns (zeroValue, false) if list is empty.
func (l *LinkedList[T]) pop() (T, bool) {
	var zero T
	if l.head == nil {
		return zero, false
	}
	// Single element case
	if l.head.Next == nil {
		val := l.head.Value
		l.head = nil
		return val, true
	}
	// Traverse to second-last node
	curr := l.head
	for curr.Next.Next != nil {
		curr = curr.Next
	}
	val := curr.Next.Value
	curr.Next = nil
	return val, true
}

// last returns the value of the last node without removing it.
// Returns (zeroValue, false) if list is empty.
func (l *LinkedList[T]) last() (T, bool) {
	var zero T
	if l.head == nil {
		return zero, false
	}
	curr := l.head
	for curr.Next != nil {
		curr = curr.Next
	}
	return curr.Value, true
}
