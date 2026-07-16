CREATE TABLE IF NOT EXISTS public.avatars
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name       VARCHAR(255) NOT NULL,
    user_id    VARCHAR(255) NOT NULL,
    width      INT NOT NULL,
    height     INT NOT NULL,
    file_size  BIGINT NOT NULL,
    mime_type  VARCHAR(25) NOT NULL,
    inactive   BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    CONSTRAINT avatars_valid_dimension_chk
    CHECK (width > 0 AND height > 0),
    CONSTRAINT avatars_valid_file_size_chk
    CHECK (file_size > 0),
    CONSTRAINT avatars_valid_mime_type_chk
    CHECK (
              mime_type IN (
              'image/jpeg',
              'image/png',
              'image/webp'
                           )
    )
    );

CREATE TABLE IF NOT EXISTS public.images
(
    avatar_id UUID NOT NULL
    REFERENCES public.avatars (id)
    ON DELETE CASCADE,

    format     VARCHAR(25) NOT NULL,
    size       VARCHAR(25) NOT NULL,
    file_size  BIGINT NOT NULL,
    s3_key     VARCHAR(500) NOT NULL,
    e_tag      VARCHAR(500) NOT NULL,
    mime_type  VARCHAR(25) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT images_valid_file_size_chk
    CHECK (file_size > 0),

    CONSTRAINT images_valid_mime_type_chk
    CHECK (
              mime_type IN (
              'image/jpeg',
              'image/png',
              'image/webp'
                           )
    ),
    CONSTRAINT images_valid_format_chk
    CHECK (format IN ('jpeg', 'png', 'webp')),

    CONSTRAINT images_valid_size_chk
    CHECK (size IN ('original', '100x100', '300x300')),
    PRIMARY KEY (avatar_id, format, size)
    );

DO
$$
BEGIN
CREATE TYPE public.task_status AS ENUM (
        'pending',
        'processing',
        'completed',
        'failed'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS public.tasks
(
    id         VARCHAR(50) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type       VARCHAR(25) NOT NULL,
    content    JSONB NOT NULL,
    status     public.task_status NOT NULL DEFAULT 'pending',
    record_id  UUID NOT NULL,
    attempt    INT NOT NULL DEFAULT 0,
    error      VARCHAR(500) DEFAULT NULL
    );

CREATE TABLE IF NOT EXISTS public.inbox
(
    id         VARCHAR(50) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    consumer   VARCHAR(50),
    content    JSONB
    );

CREATE INDEX IF NOT EXISTS avatars_user_ix
    ON public.avatars (user_id);

CREATE INDEX IF NOT EXISTS tasks_created_at_ix
    ON public.tasks (created_at);

CREATE UNIQUE INDEX IF NOT EXISTS avatars_one_active_per_user_ux
    ON public.avatars(user_id)
    WHERE inactive = FALSE
    AND deleted_at IS NULL;