// bridge/repositories/tasksrepobridge/fop.go
package tasksrepobridge

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jrazmi/envoker/core/repositories/tasksrepo"
	"github.com/jrazmi/envoker/core/scaffolding/fop"
	"github.com/jrazmi/envoker/infrastructure/web"
)

// PARAMS
type QueryParams struct {
	Limit  string
	Cursor string
	Order  string
	// Filter fields
	SearchTerm          string
	TaskID              string
	ProcessingStatus    string
	TaskType            string
	CreatedAtBefore     string
	CreatedAtAfter      string
	UpdatedAtBefore     string
	UpdatedAtAfter      string
	MinPriority         string
	MaxPriority         string
	MinRetryCount       string
	MaxRetryCount       string
	HasErrorMessage     string
	MinProcessingTimeMs string
	MaxProcessingTimeMs string
}

func parseQueryParams(r *http.Request) QueryParams {
	q := r.URL.Query()
	return QueryParams{
		Limit:               q.Get("limit"),
		Cursor:              q.Get("cursor"),
		Order:               q.Get("order"),
		SearchTerm:          q.Get("searchTerm"),
		TaskID:              q.Get("taskId"),
		ProcessingStatus:    q.Get("processingStatus"),
		TaskType:            q.Get("taskType"),
		CreatedAtBefore:     q.Get("createdAtBefore"),
		CreatedAtAfter:      q.Get("createdAtAfter"),
		UpdatedAtBefore:     q.Get("updatedAtBefore"),
		UpdatedAtAfter:      q.Get("updatedAtAfter"),
		MinPriority:         q.Get("minPriority"),
		MaxPriority:         q.Get("maxPriority"),
		MinRetryCount:       q.Get("minRetryCount"),
		MaxRetryCount:       q.Get("maxRetryCount"),
		HasErrorMessage:     q.Get("hasErrorMessage"),
		MinProcessingTimeMs: q.Get("minProcessingTimeMs"),
		MaxProcessingTimeMs: q.Get("maxProcessingTimeMs"),
	}
}

// FILTER
func parseFilter(qp QueryParams) (tasksrepo.QueryFilter, error) {
	filter := tasksrepo.QueryFilter{}

	// String filters
	if qp.SearchTerm != "" {
		filter.SearchTerm = &qp.SearchTerm
	}
	if qp.TaskID != "" {
		filter.TaskID = &qp.TaskID
	}
	if qp.ProcessingStatus != "" {
		filter.ProcessingStatus = &qp.ProcessingStatus
	}
	if qp.TaskType != "" {
		filter.TaskType = &qp.TaskType
	}

	// Time filters
	if qp.CreatedAtBefore != "" {
		if t, err := time.Parse(time.RFC3339, qp.CreatedAtBefore); err == nil {
			filter.CreatedAtBefore = &t
		} else {
			return filter, fmt.Errorf("invalid createdAtBefore format: %s", qp.CreatedAtBefore)
		}
	}
	if qp.CreatedAtAfter != "" {
		if t, err := time.Parse(time.RFC3339, qp.CreatedAtAfter); err == nil {
			filter.CreatedAtAfter = &t
		} else {
			return filter, fmt.Errorf("invalid createdAtAfter format: %s", qp.CreatedAtAfter)
		}
	}
	if qp.UpdatedAtBefore != "" {
		if t, err := time.Parse(time.RFC3339, qp.UpdatedAtBefore); err == nil {
			filter.UpdatedAtBefore = &t
		} else {
			return filter, fmt.Errorf("invalid updatedAtBefore format: %s", qp.UpdatedAtBefore)
		}
	}
	if qp.UpdatedAtAfter != "" {
		if t, err := time.Parse(time.RFC3339, qp.UpdatedAtAfter); err == nil {
			filter.UpdatedAtAfter = &t
		} else {
			return filter, fmt.Errorf("invalid updatedAtAfter format: %s", qp.UpdatedAtAfter)
		}
	}

	// Integer filters
	if qp.MinPriority != "" {
		if val, err := strconv.Atoi(qp.MinPriority); err == nil {
			filter.MinPriority = &val
		} else {
			return filter, fmt.Errorf("invalid minPriority: %s", qp.MinPriority)
		}
	}
	if qp.MaxPriority != "" {
		if val, err := strconv.Atoi(qp.MaxPriority); err == nil {
			filter.MaxPriority = &val
		} else {
			return filter, fmt.Errorf("invalid maxPriority: %s", qp.MaxPriority)
		}
	}
	if qp.MinRetryCount != "" {
		if val, err := strconv.Atoi(qp.MinRetryCount); err == nil {
			filter.MinRetryCount = &val
		} else {
			return filter, fmt.Errorf("invalid minRetryCount: %s", qp.MinRetryCount)
		}
	}
	if qp.MaxRetryCount != "" {
		if val, err := strconv.Atoi(qp.MaxRetryCount); err == nil {
			filter.MaxRetryCount = &val
		} else {
			return filter, fmt.Errorf("invalid maxRetryCount: %s", qp.MaxRetryCount)
		}
	}
	if qp.MinProcessingTimeMs != "" {
		if val, err := strconv.Atoi(qp.MinProcessingTimeMs); err == nil {
			filter.MinProcessingTimeMs = &val
		} else {
			return filter, fmt.Errorf("invalid minProcessingTimeMs: %s", qp.MinProcessingTimeMs)
		}
	}
	if qp.MaxProcessingTimeMs != "" {
		if val, err := strconv.Atoi(qp.MaxProcessingTimeMs); err == nil {
			filter.MaxProcessingTimeMs = &val
		} else {
			return filter, fmt.Errorf("invalid maxProcessingTimeMs: %s", qp.MaxProcessingTimeMs)
		}
	}

	// Boolean filters
	if qp.HasErrorMessage != "" {
		if val, err := strconv.ParseBool(qp.HasErrorMessage); err == nil {
			filter.HasErrorMessage = &val
		} else {
			return filter, fmt.Errorf("invalid hasErrorMessage: %s", qp.HasErrorMessage)
		}
	}

	return filter, nil
}

// PATH
type queryPath struct {
	TaskID   string
	TaskType string
}

func parsePath(r *http.Request) (queryPath, error) {
	pathParams := queryPath{
		TaskID:   web.Param(r, "task_id"),
		TaskType: web.Param(r, "task_type"),
	}

	return pathParams, nil
}

// ORDER
var orderByFields = map[string]string{
	"task_id":            tasksrepo.OrderByPK,
	"created_at":         tasksrepo.OrderByCreatedAt,
	"updated_at":         tasksrepo.OrderByUpdatedAt,
	"processing_status":  tasksrepo.OrderByProcessingStatus,
	"task_type":          tasksrepo.OrderByTaskType,
	"priority":           tasksrepo.OrderByPriority,
	"retry_count":        tasksrepo.OrderByRetryCount,
	"processing_time_ms": tasksrepo.OrderByProcessingTimeMs,
}

func parseOrderBy(order string) fop.By {
	if order == "" {
		return tasksrepo.DefaultOrderBy
	}

	// Use the existing FOP order parsing which handles {field},{direction} format
	orderBy, err := fop.ParseOrder(orderByFields, order, tasksrepo.DefaultOrderBy)
	if err != nil {
		return tasksrepo.DefaultOrderBy
	}

	return orderBy
}
