package mcpserver

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/types"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Service) videoDownloader() mcp.Tool {
	return mcp.NewTool("video_downloader",
		mcp.WithDescription("Download videos from given URL"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Video source URL"),
		),
	)
}

func (s *Service) downloadHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取LLM提供的参数
	url, _ := request.Params.Arguments["url"].(string)

	// params check
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	found, existingStatus, err := s.downtask.GetTaskStatusByURL(url) // <<--- 新增：调用检查方法
	if err != nil {
		// 处理查询状态时发生的错误
		return nil, fmt.Errorf("failed to check existing task status for url %s: %v", url, err)
	}

	// 如果找到了一个活动状态的任务 (非 Completed, Failed, Cancelled)
	if found && !isTerminalState(existingStatus.Stage) {
		resultText := fmt.Sprintf("A download task for URL %s is already active. Task ID: %s. Use 'video_downloader_status' to check its status.", url, existingStatus.ID)
		return mcp.NewToolResultText(resultText), nil
	}

	req := &types.DtQuickDownloadRequest{
		URL:         url,
		Video:       "best",
		BestCaption: false,
		Type:        consts.TASK_TYPE_MCP,
	}

	// download
	content, err := s.downtask.QuickDownload(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}

	resultText := fmt.Sprintf("Download started for URL: %s. Task ID: %s. You can check the status using the 'video_downloader_status' tool with this ID.", url, content.ID)

	return mcp.NewToolResultText(resultText), nil
}

func (s *Service) videoDownloaderStatus() mcp.Tool {
	return mcp.NewTool("video_downloader_status",
		mcp.WithDescription("Check the status of a video download task"),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("The ID of the download task to check"),
		),
	)
}

func (s *Service) downloadStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) { // <<--- 新增函数
	// 获取LLM提供的参数
	taskID, _ := request.Params.Arguments["task_id"].(string)

	// params check
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	// 查询任务状态
	status, err := s.downtask.GetTaskStatus(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status for ID %s: %v", taskID, err)
	}

	// 将状态信息转换为字符串返回给 LLM
	if status == nil {
		return mcp.NewToolResultText(fmt.Sprintf("No status found for task %s", taskID)), nil
	}

	var statusString string
	if status.Error != "" {
		statusString = fmt.Sprintf("Task %s failed. Error: %s", taskID, status.Error)
	}

	switch status.Stage {
	case types.DtStageInitializing:
		statusString = fmt.Sprintf("Task %s is initializing", taskID)
	case types.DtStageDownloading:
		statusString = fmt.Sprintf("Task %s is downloading. Percentage: %.2f%%. Speed: %v, Estimated Time:%v", taskID, status.Percentage, status.Speed, status.EstimatedTime)
	case types.DtStageTranslating, types.DtStageEmbedding:
		statusString = fmt.Sprintf("Task %s is %s", taskID, status.Stage)
	case types.DtStageCompleted:
		statusString = fmt.Sprintf("Task %s is completed. Resolution: %s. File Size(bytes): %d, Duration(seconds): %v", taskID, status.Resolution, status.FileSize, status.Duration)
	case types.DtStageCancelled:
		statusString = fmt.Sprintf("Task %s is Cancelled. Info: %v", taskID, status.StageInfo)
	case types.DtStageFailed:
		statusString = fmt.Sprintf("Task %s failed. Error: %s", taskID, status.Error)
	default:
		statusString = fmt.Sprintf("Unknown stage for task %s", taskID)
	}

	return mcp.NewToolResultText(statusString), nil
}

func (s *Service) listDownloadTasks() mcp.Tool {
	return mcp.NewTool("list_download_tasks",
		mcp.WithDescription("List all current video download tasks and their status."),
	)
}

func (s *Service) listTasksHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tasks := s.downtask.ListTasks()
	if len(tasks) == 0 {
		return mcp.NewToolResultText("No download tasks found."), nil
	}

	// 格式化任务列表为字符串
	var resultString string = "Current Download Tasks:\n"
	for i, task := range tasks {
		// 为了简洁，只显示部分关键信息
		taskInfo := fmt.Sprintf("%d. Task ID: %s, Stage: %s", i+1, task.ID, task.Stage)
		// fill in base info
		taskInfo += fmt.Sprintf(", URL: %s, Title: %s, Source: %s, Resolution: %s, File Size(bytes): %v", task.URL, task.Title, task.Extractor, task.Resolution, task.FileSize)
		if task.Stage == types.DtStageDownloading {
			taskInfo += fmt.Sprintf(", Percentage: %.2f%%", task.Percentage)
		} else if task.Error != "" {
			taskInfo += fmt.Sprintf(", Error: %s", task.Error)
		}
		resultString += taskInfo + "\n"
	}

	return mcp.NewToolResultText(resultString), nil
}

func isTerminalState(stage types.DtTaskStage) bool {
	switch stage {
	case types.DtStageCompleted, types.DtStageFailed, types.DtStageCancelled:
		return true
	default:
		return false
	}
}
