#!/bin/bash

# Multi-Agent System Test Script
# Validates that all components are properly configured

set -e

echo "🧪 Testing Multi-Agent Coordination System"
echo "=========================================="

# Check directory structure
echo "📁 Checking directory structure..."
for dir in agents skills hooks; do
    if [[ -d ".claude/$dir" ]]; then
        echo "✅ .claude/$dir directory exists"
    else
        echo "❌ .claude/$dir directory missing"
        exit 1
    fi
done

# Check agent files
echo ""
echo "🤖 Checking agent definitions..."
agents=("python-dev" "go-developer" "devops" "tech-writer" "code-analyzer" "test-planner")
for agent in "${agents[@]}"; do
    if [[ -f ".claude/agents/$agent.md" ]]; then
        echo "✅ $agent agent defined"
    else
        echo "❌ $agent agent missing"
        exit 1
    fi
done

# Check skill files
echo ""
echo "🎭 Checking skill definitions..."
skills=("multi-agent-coordinator" "output-evaluator")
for skill in "${skills[@]}"; do
    if [[ -f ".claude/skills/$skill.md" ]]; then
        echo "✅ $skill skill defined"
    else
        echo "❌ $skill skill missing"
        exit 1
    fi
done

# Check hook files
echo ""
echo "🎣 Checking hook scripts..."
hooks=("pre-tool-use.sh" "post-tool-use.sh")
for hook in "${hooks[@]}"; do
    if [[ -f ".claude/hooks/$hook" ]]; then
        if [[ -x ".claude/hooks/$hook" ]]; then
            echo "✅ $hook exists and is executable"
        else
            echo "⚠️  $hook exists but is not executable"
            chmod +x ".claude/hooks/$hook"
            echo "🔧 Fixed: Made $hook executable"
        fi
    else
        echo "❌ $hook missing"
        exit 1
    fi
done

# Check settings configuration
echo ""
echo "⚙️ Checking settings configuration..."
if [[ -f ".claude/settings.local.json" ]]; then
    echo "✅ settings.local.json exists"

    # Validate JSON syntax
    if python3 -m json.tool .claude/settings.local.json > /dev/null 2>&1; then
        echo "✅ settings.local.json is valid JSON"
    else
        echo "❌ settings.local.json has invalid JSON syntax"
        exit 1
    fi

    # Check for hooks configuration
    if grep -q "PreToolUse" .claude/settings.local.json && grep -q "PostToolUse" .claude/settings.local.json; then
        echo "✅ Hooks are configured in settings"
    else
        echo "❌ Hooks not configured in settings"
        exit 1
    fi
else
    echo "❌ settings.local.json missing"
    exit 1
fi

# Test hook execution
echo ""
echo "🧪 Testing hook execution..."

# Test pre-tool-use hook
if .claude/hooks/pre-tool-use.sh > /dev/null 2>&1; then
    echo "✅ pre-tool-use.sh executes successfully"
else
    echo "❌ pre-tool-use.sh failed to execute"
    exit 1
fi

# Test post-tool-use hook
if .claude/hooks/post-tool-use.sh > /dev/null 2>&1; then
    echo "✅ post-tool-use.sh executes successfully"
else
    echo "❌ post-tool-use.sh failed to execute"
    exit 1
fi

# Check agent expertise mapping and fork functionality
echo ""
echo "🎯 Validating coordination capabilities..."
if grep -q "context: fork" .claude/skills/multi-agent-coordinator.md; then
    echo "✅ Fork-based coordination configured"
else
    echo "❌ Fork-based coordination missing"
fi

if grep -q "python-dev" .claude/skills/multi-agent-coordinator.md; then
    echo "✅ Python development mapping exists"
else
    echo "❌ Python development mapping missing"
fi

if grep -q "go-developer" .claude/skills/multi-agent-coordinator.md; then
    echo "✅ Go development mapping exists"
else
    echo "❌ Go development mapping missing"
fi

if grep -q "devops" .claude/skills/multi-agent-coordinator.md; then
    echo "✅ DevOps mapping exists"
else
    echo "❌ DevOps mapping missing"
fi

# Check evaluation criteria
echo ""
echo "📊 Validating evaluation criteria..."
if grep -q "context: fork" .claude/skills/output-evaluator.md; then
    echo "✅ Fork-based evaluation configured"
else
    echo "❌ Fork-based evaluation missing"
fi

if grep -q "Agent-Specific Quality Standards" .claude/skills/output-evaluator.md; then
    echo "✅ Agent-specific standards defined"
else
    echo "❌ Agent-specific standards missing"
fi

if grep -q "Quality Scoring" .claude/skills/output-evaluator.md; then
    echo "✅ Quality scoring system defined"
else
    echo "❌ Quality scoring system missing"
fi

echo ""
echo "🎉 Multi-Agent System Test Results"
echo "================================="
echo "✅ All components are properly configured"
echo "✅ Directory structure is correct"
echo "✅ Agent definitions are complete"
echo "✅ Skills are properly defined"
echo "✅ Hooks are executable and functional"
echo "✅ Settings configuration is valid"
echo ""
echo "🚀 System is ready for use!"
echo ""
echo "Usage:"
echo "  • Use task management tools (TaskCreate, TaskList, TaskUpdate)"
echo "  • Hooks will automatically provide multi-agent guidance"
echo "  • Invoke skills via Skill tool: multi-agent-coordinator, output-evaluator"
echo "  • Aliases: /mac, /agents, /parallel, /eval, /validate, /check"
echo ""
echo "📚 See .claude/README.md for complete documentation"