
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
