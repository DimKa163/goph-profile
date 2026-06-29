CREATE TABLE IF NOT EXISTS public.users(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    name VARCHAR(50) NOT NULL
    );

CREATE TABLE IF NOT EXISTS public.avatars(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    name VARCHAR(50) NOT NULL,
    user_id UUID NOT NULL REFERENCES public.users(id)  ON DELETE CASCADE,
    s3_key VARCHAR(500) NOT NULL DEFAULT '',
    upload_status VARCHAR(50) DEFAULT 'uploading',
    processing_status VARCHAR(50) DEFAULT 'pending',
    mime_type VARCHAR(25) NOT NULL,
    file_size INT NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    CONSTRAINT valid_dimension CHECK(width > 0 AND height > 0),
    CONSTRAINT valid_size CHECK(file_size > 0),
    CONSTRAINT valid_mime_type CHECK(mime_type IN (
                                     'image/jpeg',
                                     'image/png',
                                     'image/webp',
                                     'image/gif'
                                                  ))
    );

CREATE TABLE IF NOT EXISTS public.thumbnails(
    avatar_id UUID NOT NULL REFERENCES public.avatars(id)  ON DELETE CASCADE,
    size VARCHAR(25) NOT NULL,
    url VARCHAR(100) NOT NULL,

    PRIMARY KEY (avatar_id, size)
);

CREATE INDEX avatars_user_ix ON public.avatars(user_id);