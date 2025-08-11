package cmd

import (
	"fmt"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/spf13/cobra"
	"mcp-pipeline-server/tools"
	"os"
)

var pipelineCfg = struct {
	ListenAddress string
	PipelinesURL  string
}{
	ListenAddress: "0.0.0.0:8085",
	PipelinesURL:  "http://127.0.0.1:8888",
}

var MCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Kubeflow Pipelines MCP server",
	Long:  `Launch the MCP server for Kubeflow Pipelines`,
	RunE:  runMCPServer,
}

func init() {
	MCPCmd.Flags().StringVarP(&pipelineCfg.ListenAddress, "listen", "l", pipelineCfg.ListenAddress, "Address to listen on")
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	pipelinesUrl := os.Getenv("KFP_PIPELINE_URL")
	if pipelinesUrl != "" {
		pipelineCfg.PipelinesURL = pipelinesUrl
	}

	done := make(chan struct{})

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	err := server.RegisterTool("List Tools", "List all available tools", tools.ListToolsHandler)
	if err != nil {
		return fmt.Errorf("error registering tool %s: %v", "List Tools", err)
	}
	for _, tool := range tools.PipelineTools {
		err = server.RegisterTool(tool.Name, tool.Description, tool.Handler)
		if err != nil {
			return fmt.Errorf("error registering tool %s: %v", tool.Name, err)
		}
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	<-done

	return nil
}
