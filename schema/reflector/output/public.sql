-- =============================================================================
-- Schema Reflection: postgres.public
-- Reflected at: 2025-10-19 20:42:36
-- Tables: 4
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

-- -----------------------------------------------------------------------------
-- Table: user_sessions
-- Active user sessions for authentication and authorization.
-- -----------------------------------------------------------------------------
CREATE TABLE public.user_sessions (
    session_id uuid NOT NULL DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    session_token varchar(255) NOT NULL,
    refresh_token varchar(255),
    ip_address inet,
    user_agent text,
    device_info jsonb,
    expires_at timestamp NOT NULL,
    refresh_expires_at timestamp,
    status varchar(50) NOT NULL DEFAULT active,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    last_active_at timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id),
    FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);
CREATE INDEX idx_sessions_created ON public.user_sessions USING btree (created_at);
CREATE INDEX idx_sessions_expiry ON public.user_sessions USING btree (expires_at);
CREATE INDEX idx_sessions_token ON public.user_sessions USING btree (session_token);
CREATE INDEX idx_sessions_user ON public.user_sessions USING btree (user_id, status);
CREATE UNIQUE INDEX user_sessions_session_token_key ON public.user_sessions USING btree (session_token);

COMMENT ON TABLE public.user_sessions IS 'Active user sessions for authentication and authorization.';

-- -----------------------------------------------------------------------------
-- Table: users
-- User accounts for accessing and managing the system.
-- -----------------------------------------------------------------------------
CREATE TABLE public.users (
    user_id uuid NOT NULL DEFAULT gen_random_uuid(),
    email varchar(255) NOT NULL,
    username varchar(100),
    password_hash varchar(255),
    password_salt varchar(255),
    first_name varchar(100),
    last_name varchar(100),
    display_name varchar(255),
    email_verified bool DEFAULT false,
    email_verified_at timestamp,
    role varchar(50) DEFAULT user,
    permissions jsonb DEFAULT [],
    last_login_at timestamp,
    last_login_ip inet,
    failed_login_attempts int4 DEFAULT 0,
    locked_until timestamp,
    password_reset_token varchar(255),
    password_reset_expires_at timestamp,
    metadata jsonb DEFAULT {},
    status varchar(50) NOT NULL DEFAULT active,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id)
);
CREATE INDEX idx_users_created ON public.users USING btree (created_at);
CREATE INDEX idx_users_email ON public.users USING btree (email);
CREATE INDEX idx_users_role ON public.users USING btree (role);
CREATE INDEX idx_users_status ON public.users USING btree (status);
CREATE INDEX idx_users_username ON public.users USING btree (username);
CREATE UNIQUE INDEX users_email_key ON public.users USING btree (email);
CREATE UNIQUE INDEX users_username_key ON public.users USING btree (username);

COMMENT ON TABLE public.users IS 'User accounts for accessing and managing the system.';

