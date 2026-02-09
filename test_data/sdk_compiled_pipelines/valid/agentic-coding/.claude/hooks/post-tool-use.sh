#!/bin/bash

# Post-Tool-Use Hook: Output Evaluation
# This hook runs after tool execution to evaluate outputs and ensure quality

set -e

# Check if this was a task completion or agent output
if [[ "$CLAUDE_TOOL_NAME" == "TaskUpdate" ]] && echo "$CLAUDE_TOOL_ARGS" | grep -q "completed"; then

    echo "✅ Task completion detected - triggering quality evaluation..."

    # Extract task information
    TASK_INFO="${CLAUDE_TOOL_ARGS}"

    echo "🔍 Output Evaluator: Analyzing completed task..."
    echo "📊 Evaluation criteria:"
    echo "   • System prompt compliance"
    echo "   • User request fulfillment"
    echo "   • Technical excellence"
    echo "   • Agent-specific standards"

    echo "💡 Use /output-evaluator skill for detailed validation"

elif [[ "$CLAUDE_TOOL_NAME" == "Task" ]]; then

    echo "🤖 Agent task execution completed - evaluation recommended"

    # Check what type of agent was used
    if echo "$CLAUDE_TOOL_ARGS" | grep -qE "(python|go|devops|test|doc|code)" -i; then
        echo "🎯 Specialized agent output detected"
        echo "🔬 Quality evaluation available via: /output-evaluator"

        # Provide specific evaluation points based on agent type
        if echo "$CLAUDE_TOOL_ARGS" | grep -q "python" -i; then
            echo "🐍 Python-specific evaluation points:"
            echo "   • Type hints present"
            echo "   • PEP 8 compliance"
            echo "   • Error handling"
            echo "   • Test coverage"
        elif echo "$CLAUDE_TOOL_ARGS" | grep -q "go" -i; then
            echo "🔧 Go-specific evaluation points:"
            echo "   • Idiomatic Go patterns"
            echo "   • Error handling"
            echo "   • Race condition safety"
            echo "   • Performance considerations"
        elif echo "$CLAUDE_TOOL_ARGS" | grep -q "devops" -i; then
            echo "🚀 DevOps-specific evaluation points:"
            echo "   • Security hardening"
            echo "   • Resource optimization"
            echo "   • Monitoring setup"
            echo "   • Documentation completeness"
        fi
    fi

elif [[ "$CLAUDE_TOOL_NAME" == "Edit" ]] || [[ "$CLAUDE_TOOL_NAME" == "Write" ]]; then

    # Check if this was code modification
    if echo "$CLAUDE_TOOL_ARGS" | grep -qE "\.(py|go|js|ts|yaml|dockerfile)" -i; then
        echo "📝 Code modification detected"
        echo "🔍 Consider output evaluation for:"
        echo "   • Code quality standards"
        echo "   • Security implications"
        echo "   • Best practices adherence"
        echo "   • Testing requirements"

        # Extract file extension for specific guidance
        if echo "$CLAUDE_TOOL_ARGS" | grep -q "\.py" -i; then
            echo "🐍 Python code - verify: type hints, PEP 8, error handling"
        elif echo "$CLAUDE_TOOL_ARGS" | grep -q "\.go" -i; then
            echo "🔧 Go code - verify: idiomatic patterns, error handling, tests"
        fi
    fi
fi

# Always provide evaluation framework reminder
echo ""
echo "📋 Quality Evaluation Framework:"
echo "   1. Functional correctness"
echo "   2. Code quality standards"
echo "   3. Security considerations"
echo "   4. Performance implications"
echo "   5. Documentation completeness"
echo ""
echo "🎭 Use /output-evaluator for comprehensive validation"

# Check for common quality issues in tool output
if [[ -n "$CLAUDE_TOOL_OUTPUT" ]]; then
    # Basic automated checks
    OUTPUT_LENGTH=$(echo "$CLAUDE_TOOL_OUTPUT" | wc -c)

    if [[ $OUTPUT_LENGTH -lt 50 ]]; then
        echo "⚠️  Warning: Tool output seems brief - verify completeness"
    fi

    if echo "$CLAUDE_TOOL_OUTPUT" | grep -qE "(TODO|FIXME|XXX)"; then
        echo "⚠️  Warning: TODO/FIXME markers found - incomplete implementation"
    fi

    if echo "$CLAUDE_TOOL_OUTPUT" | grep -qE "(password|secret|key)" -i; then
        echo "🚨 Security Alert: Potential sensitive data in output"
    fi
fi

exit 0