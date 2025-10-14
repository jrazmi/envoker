
-- Users table for evokeshop application
CREATE TABLE public.users (
    user_id CHARACTER VARYING NOT NULL,
    email CHARACTER VARYING UNIQUE NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status CHARACTER VARYING DEFAULT 'active'::CHARACTER VARYING,
    CONSTRAINT users_pkey PRIMARY KEY (user_id),
    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE TABLE public.users_kvs (
    user_id CHARACTER VARYING NOT NULL,
    key CHARACTER VARYING NOT NULL,
    value CHARACTER VARYING,
    CONSTRAINT user_kvs_pkey PRIMARY KEY (user_id, key),
    CONSTRAINT user_kvs_user_id_fk FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);



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


