-- =============================================================================
-- Add Users Table
-- This migration adds user management functionality to the system
-- =============================================================================

-- -----------------------------------------------------------------------------
-- USERS
-- -----------------------------------------------------------------------------
CREATE TABLE users (
    user_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    -- User identification
    email varchar(255) NOT NULL UNIQUE,
    username varchar(100) UNIQUE,

    -- Authentication
    password_hash varchar(255),
    password_salt varchar(255),

    -- User profile
    first_name varchar(100),
    last_name varchar(100),
    display_name varchar(255),

    -- User status
    email_verified boolean DEFAULT false,
    email_verified_at timestamp,

    -- Role and permissions
    role varchar(50) DEFAULT 'user',
    permissions jsonb DEFAULT '[]',

    -- Security
    last_login_at timestamp,
    last_login_ip inet,
    failed_login_attempts int4 DEFAULT 0,
    locked_until timestamp,

    -- Password reset
    password_reset_token varchar(255),
    password_reset_expires_at timestamp,

    -- Metadata
    metadata jsonb DEFAULT '{}',

    -- Timestamps
    status varchar(50) NOT NULL DEFAULT 'active',
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username) WHERE username IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_created ON users(created_at DESC);

COMMENT ON TABLE users IS 'User accounts for accessing and managing the system.';

-- -----------------------------------------------------------------------------
-- USER SESSIONS
-- -----------------------------------------------------------------------------
CREATE TABLE user_sessions (
    session_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,

    -- Session data
    session_token varchar(255) NOT NULL UNIQUE,
    refresh_token varchar(255),

    -- Session metadata
    ip_address inet,
    user_agent text,
    device_info jsonb,

    -- Session expiry
    expires_at timestamp NOT NULL,
    refresh_expires_at timestamp,

    -- Status
    status varchar(50) NOT NULL DEFAULT 'active',

    -- Timestamps
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    last_active_at timestamp NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_sessions_user ON user_sessions(user_id, status);
CREATE INDEX idx_sessions_token ON user_sessions(session_token) WHERE status = 'active';
CREATE INDEX idx_sessions_expiry ON user_sessions(expires_at) WHERE status = 'active';
CREATE INDEX idx_sessions_created ON user_sessions(created_at DESC);

COMMENT ON TABLE user_sessions IS 'Active user sessions for authentication and authorization.';
