package mcpserver

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/core/downtasks"
	"dreamcreator/backend/pkg/logger"
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

type Service struct {
	ctx      context.Context
	svr      *server.MCPServer
	downtask *downtasks.Service
}

func NewService(downtask *downtasks.Service) *Service {
	// Create MCP server
	s := server.NewMCPServer(
		consts.AppDisplayName(),
		consts.APP_VERSION,
	)
	return &Service{
		svr:      s,
		downtask: downtask,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.ctx = ctx

	// Add tool
	// download video
	s.svr.AddTool(s.videoDownloader(), s.downloadHandler)
	// download status
	s.svr.AddTool(s.videoDownloaderStatus(), s.downloadStatusHandler)
	// list all tasks
	s.svr.AddTool(s.listDownloadTasks(), s.listTasksHandler)
	// Start the stdio server
	if err := server.ServeStdio(s.svr); err != nil {
		return fmt.Errorf("Server error: %v\n", err)
	}

	sseServer := server.NewSSEServer(s.svr, server.WithBaseURL(fmt.Sprintf("http://localhost:%v", consts.MCP_SERVER_PORT)))
	if err := sseServer.Start(fmt.Sprintf(":%v", consts.MCP_SERVER_PORT)); err != nil {
		logger.Error("SSE server start failed", zap.Int("port", consts.MCP_SERVER_PORT), zap.Error(err))
	}

	return nil
}

func (s *Service) Stop() error {
	return nil
}
