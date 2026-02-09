#!/bin/bash

# Pre-Tool-Use Hook: Multi-Agent Coordination
# This hook runs before tool execution to coordinate multi-agent workflows

set -e

# Check if this is a task-related tool call that could benefit from multi-agent coordination
if [[ "$CLAUDE_TOOL_NAME" == "TaskCreate" ]] || [[ "$CLAUDE_TOOL_NAME" == "TaskUpdate" ]] || [[ "$CLAUDE_TOOL_NAME" == "TaskList" ]]; then

    echo "🤖 Multi-Agent Coordinator: Analyzing task requirements..."

    # Extract task information from environment variables set by Claude Code
    TASK_DESCRIPTION="${CLAUDE_TOOL_ARGS}"

    # Check if we have multiple pending tasks that could be parallelized
    if [[ -n "$TASK_DESCRIPTION" ]] && [[ "$CLAUDE_TOOL_NAME" == "TaskCreate" ]]; then
        echo "📋 New task detected, checking for multi-agent opportunities..."

        # Trigger multi-agent analysis for complex tasks
        if echo "$TASK_DESCRIPTION" | grep -qE "(python|go|docker|test|documentation|review)"; then
            echo "🎯 Task appears suitable for specialized agent handling"
            echo "💡 Consider using multi-agent-coordinator skill for optimal parallel execution"
        fi
    fi

    # Check for task list operations that might trigger coordination
    if [[ "$CLAUDE_TOOL_NAME" == "TaskList" ]]; then
        echo "📊 Task list being accessed - checking for coordination opportunities..."
        echo "🔄 Multi-agent coordination available via: /multi-agent-coordinator"
    fi
fi

# For other tool calls, provide context about available specialized agents
case "$CLAUDE_TOOL_NAME" in
    "Edit"|"Write"|"Read")
        if echo "$CLAUDE_TOOL_ARGS" | grep -qE "\.(py|go|yaml|md|dockerfile)" -i; then
            echo "🔧 File operation detected - specialized agents available:"
            echo "   • Python files: python-dev agent"
            echo "   • Go files: go-developer agent"
            echo "   • Docker/K8s: devops agent"
            echo "   • Documentation: tech-writer agent"
        fi
        ;;
    "Bash")
        if echo "$CLAUDE_TOOL_ARGS" | grep -qE "(docker|kubectl|go|python|test)" -i; then
            echo "⚡ Command execution detected - consider specialized agents for:"
            echo "   • DevOps commands: devops agent"
            echo "   • Python operations: python-dev agent"
            echo "   • Go operations: go-developer agent"
            echo "   • Testing: test-planner agent"
        fi
        ;;
esac

# Always provide reminder about available coordination
echo "🎭 Multi-agent skills available:"
echo "   • /multi-agent-coordinator - Parallel task execution"
echo "   • /output-evaluator - Quality validation"

exit 0