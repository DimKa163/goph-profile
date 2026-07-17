ALTER TABLE public.tasks
ADD COLUMN traceparent TEXT NULL,
ADD COLUMN tracestate TEXT NULL;