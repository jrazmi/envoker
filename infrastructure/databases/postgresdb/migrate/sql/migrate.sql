-- Version: 1.01
-- Description: Initial schema 

CREATE TABLE tasks (
    task_id VARCHAR NOT NULL,
    processing_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    task_type VARCHAR(100) NOT NULL,
    metadata JSONB,
    priority INTEGER DEFAULT 0,           -- For prioritization
    max_retries INTEGER DEFAULT 3,        -- Retry limit
    retry_count INTEGER DEFAULT 0,        -- Current retry count
    error_message TEXT,                   -- Last error details
    processing_time_ms INTEGER,
    last_run_at TIMESTAMP,
    CONSTRAINT task_pk PRIMARY KEY (task_id)
);


