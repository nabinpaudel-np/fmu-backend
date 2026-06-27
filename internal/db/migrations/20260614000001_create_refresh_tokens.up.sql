CREATE TABLE refresh_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    token varchar(500) NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_agent varchar(500),
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked boolean DEFAULT false NOT NULL
);

CREATE INDEX idx_refresh_tokens_token ON refresh_tokens USING btree (token);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens USING btree (user_id);