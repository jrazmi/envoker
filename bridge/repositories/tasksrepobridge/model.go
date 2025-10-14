// bridge/repositories/tasksrepobridge/model.go
package tasksrepobridge

import (
	"encoding/json"
)

// Task represents the bridge model for task
type Task struct {
	TaskID           string                 `json:"taskId"`
	TaskType         string                 `json:"taskType"`
	ProcessingStatus string                 `json:"processingStatus"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Priority         int                    `json:"priority"`
	RetryCount       int                    `json:"retryCount"`
	MaxRetries       int                    `json:"maxRetries"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	ProcessingTimeMs int                    `json:"processingTimeMs,omitempty"`
	LastRunAt        string                 `json:"lastRunAt,omitempty"`
}

// Encode implements the encoder interface
func (t Task) Encode() ([]byte, string, error) {
	data, err := json.Marshal(t)
	return data, "application/json", err
}

// CreateTaskInput represents the input for creating a new task
type CreateTaskInput struct {
	TaskType string                 `json:"taskType"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Priority int                    `json:"priority,omitempty"`
}

// Decode implements the decoder interface
func (c *CreateTaskInput) Decode(data []byte) error {
	return json.Unmarshal(data, c)
}

// UpdateTaskInput represents the input for updating a task
type UpdateTaskInput struct {
	ProcessingStatus *string                `json:"processingStatus,omitempty"`
	TaskType         *string                `json:"taskType,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Priority         *int                   `json:"priority,omitempty"`
	MaxRetries       *int                   `json:"maxRetries,omitempty"`
	RetryCount       *int                   `json:"retryCount,omitempty"`
	ErrorMessage     *string                `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int                   `json:"processingTimeMs,omitempty"`
}

// Decode implements the decoder interface
func (u *UpdateTaskInput) Decode(data []byte) error {
	return json.Unmarshal(data, u)
}

// CompleteTaskInput represents the input for completing a task
type CompleteTaskInput struct {
	TaskID           string                 `json:"taskId"`
	ProcessingTimeMs int                    `json:"processingTimeMs"`
	Result           map[string]interface{} `json:"result,omitempty"`
}

// Decode implements the decoder interface
func (c *CompleteTaskInput) Decode(data []byte) error {
	return json.Unmarshal(data, c)
}

// FailTaskInput represents the input for failing a task
type FailTaskInput struct {
	TaskID       string `json:"taskId"`
	ErrorMessage string `json:"errorMessage"`
}

// Decode implements the decoder interface
func (f *FailTaskInput) Decode(data []byte) error {
	return json.Unmarshal(data, f)
}

// TasksList represents a list of tasks
type TasksList struct {
	Tasks []Task `json:"tasks,omitempty"`
}

// Encode implements the encoder interface
func (tl TasksList) Encode() ([]byte, string, error) {
	data, err := json.Marshal(tl)
	return data, "application/json", err
}
