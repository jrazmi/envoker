// bridge/repositories/tasksrepobridge/marshal.go
package tasksrepobridge

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/jrazmi/envoker/core/repositories/tasksrepo"
	"github.com/jrazmi/envoker/sdk/validation"
)

// TO this pattern (like your other bridges):
func MarshalToBridge(task tasksrepo.Task) Task {
	var metadata map[string]interface{}
	if task.Metadata != nil {
		json.Unmarshal(*task.Metadata, &metadata)
	}

	// Handle pointer fields safely
	var priority, retryCount, maxRetries, processingTimeMs int
	var errorMessage string

	if task.Priority != nil {
		priority = *task.Priority
	}
	if task.RetryCount != nil {
		retryCount = *task.RetryCount
	}
	if task.MaxRetries != nil {
		maxRetries = *task.MaxRetries
	}
	if task.ProcessingTimeMs != nil {
		processingTimeMs = *task.ProcessingTimeMs
	}
	if task.ErrorMessage != nil {
		errorMessage = *task.ErrorMessage
	}

	return Task{
		TaskID:           task.TaskID,
		TaskType:         task.TaskType,
		ProcessingStatus: task.ProcessingStatus,
		Metadata:         metadata,
		Priority:         priority,
		RetryCount:       retryCount,
		MaxRetries:       maxRetries,
		CreatedAt:        validation.FormatTimePtrToString(task.CreatedAt),
		UpdatedAt:        validation.FormatTimePtrToString(task.UpdatedAt),
		ErrorMessage:     errorMessage,
		ProcessingTimeMs: processingTimeMs,
		LastRunAt:        validation.FormatTimePtrToString(task.LastRunAt),
	}
}

// MarshalListToBridge converts a list of core models to bridge models
func MarshalListToBridge(tasks []tasksrepo.Task) []Task {
	bridgeTasks := make([]Task, len(tasks))
	for i, task := range tasks {
		bridgeTasks[i] = MarshalToBridge(task)
	}
	return bridgeTasks
}

// MarshalCreateToRepository converts bridge create input to repository input
func MarshalCreateToRepository(input CreateTaskInput) tasksrepo.CreateTask {
	// Handle convenience fields for OCR tasks
	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	var metadataRaw *json.RawMessage
	if len(metadata) > 0 {
		bytes, _ := json.Marshal(metadata)
		raw := json.RawMessage(bytes)
		metadataRaw = &raw
	}

	return tasksrepo.CreateTask{
		TaskType:         input.TaskType,
		ProcessingStatus: "pending",
		Metadata:         metadataRaw,
		Priority:         &input.Priority,
	}
}

// MarshalUpdateToRepository converts bridge update input to repository input
func MarshalUpdateToRepository(input UpdateTaskInput) tasksrepo.UpdateTask {
	var metadataRaw *json.RawMessage
	if input.Metadata != nil && len(input.Metadata) > 0 {
		bytes, _ := json.Marshal(input.Metadata)
		raw := json.RawMessage(bytes)
		metadataRaw = &raw
	}

	return tasksrepo.UpdateTask{
		ProcessingStatus: input.ProcessingStatus,
		TaskType:         input.TaskType,
		Metadata:         metadataRaw,
		Priority:         input.Priority,
		MaxRetries:       input.MaxRetries,
		RetryCount:       input.RetryCount,
		ErrorMessage:     input.ErrorMessage,
		ProcessingTimeMs: input.ProcessingTimeMs,
	}
}

// MarshalResultToRepository converts result metadata to repository update
func MarshalResultToRepository(result map[string]interface{}) tasksrepo.UpdateTask {
	var metadataRaw *json.RawMessage
	if result != nil && len(result) > 0 {
		bytes, _ := json.Marshal(result)
		raw := json.RawMessage(bytes)
		metadataRaw = &raw
	}

	return tasksrepo.UpdateTask{
		Metadata: metadataRaw,
	}
}

func extractFilename(path string) string {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}
