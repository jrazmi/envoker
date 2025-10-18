-- =============================================================================
-- Schema Reflection: postgres.public
-- Reflected at: 2025-10-18 12:26:23
-- Tables: 2
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Table: schema_migrations
-- -----------------------------------------------------------------------------
CREATE TABLE public.schema_migrations (
    version varchar(255) NOT NULL,
    checksum varchar(64) NOT NULL,
    applied_at timestamp NOT NULL DEFAULT now(),
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (version)
);

-- -----------------------------------------------------------------------------
-- Table: tasks
-- -----------------------------------------------------------------------------
CREATE TABLE public.tasks (
    task_id varchar NOT NULL,
    processing_status varchar(50) NOT NULL DEFAULT pending,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    task_type varchar(100) NOT NULL,
    metadata jsonb,
    priority int4 DEFAULT 0,
    max_retries int4 DEFAULT 3,
    retry_count int4 DEFAULT 0,
    error_message text,
    processing_time_ms int4,
    last_run_at timestamp,
    PRIMARY KEY (task_id)
);

