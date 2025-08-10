#!/bin/bash

echo "=== Kubeflow Pipelines AI Agent Integration Test ==="
echo

# Test 1: Component functions
echo "1. Testing core AI Agent functions..."
go run main.go test_functions.go test

if [ $? -eq 0 ]; then
    echo "✅ Core functions test passed!"
else
    echo "❌ Core functions test failed!"
    exit 1
fi

echo

# Test 2: Check generated pipeline
echo "2. Checking generated pipeline structure..."

PIPELINE_FILE=$(ls generated_pipelines/*.yaml 2>/dev/null | head -1)
if [ -f "$PIPELINE_FILE" ]; then
    echo "✅ Pipeline file generated: $PIPELINE_FILE"
    
    # Check if pipeline has required sections
    if grep -q "pipelineInfo:" "$PIPELINE_FILE" && \
       grep -q "components:" "$PIPELINE_FILE" && \
       grep -q "deploymentSpec:" "$PIPELINE_FILE" && \
       grep -q "root:" "$PIPELINE_FILE"; then
        echo "✅ Pipeline structure is valid"
    else
        echo "❌ Pipeline structure is invalid"
        exit 1
    fi
    
    # Check file size
    FILE_SIZE=$(stat -f%z "$PIPELINE_FILE" 2>/dev/null || stat -c%s "$PIPELINE_FILE" 2>/dev/null)
    if [ "$FILE_SIZE" -gt 1000 ]; then
        echo "✅ Pipeline file size is adequate ($FILE_SIZE bytes)"
    else
        echo "❌ Pipeline file is too small ($FILE_SIZE bytes)"
        exit 1
    fi
else
    echo "❌ No pipeline file generated"
    exit 1
fi

echo

# Test 3: MCP Server compilation
echo "3. Testing MCP Server compilation..."
cd ../mcp-server
if go build main.go; then
    echo "✅ MCP Server compiles successfully"
    rm -f main
else
    echo "❌ MCP Server compilation failed"
    exit 1
fi
cd ../ai-agent

echo

# Test 4: AI Agent compilation
echo "4. Testing AI Agent compilation..."
if go build main.go test_functions.go; then
    echo "✅ AI Agent compiles successfully"
    rm -f main
else
    echo "❌ AI Agent compilation failed"
    exit 1
fi

echo

# Test 5: Check pipeline file count
echo "5. Checking input pipeline files..."
PIPELINE_COUNT=$(find ../data/pipeline_files/valid -name "*.yaml" | wc -l)
if [ "$PIPELINE_COUNT" -gt 50 ]; then
    echo "✅ Found $PIPELINE_COUNT input pipeline files"
else
    echo "❌ Insufficient input pipeline files ($PIPELINE_COUNT found, expected > 50)"
    exit 1
fi

echo

echo "=== All Tests Passed! ==="
echo
echo "Summary:"
echo "  ✅ Core AI Agent functions work correctly"
echo "  ✅ Pipeline generation and compilation successful"
echo "  ✅ Generated pipeline structure is valid"
echo "  ✅ MCP Server integration ready"
echo "  ✅ Input pipeline files accessible"
echo
echo "To start the AI Agent server:"
echo "  cd ai-agent"
echo "  go run main.go"
echo "  # Visit http://localhost:8080"
echo
echo "To test the Upload Pipeline API:"
echo "  curl -X POST http://localhost:8080/api/v1/upload-pipeline"