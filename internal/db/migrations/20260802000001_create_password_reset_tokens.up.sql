CREATE TABLE password_reset_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash varchar(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX idx_password_reset_tokens_token ON password_reset_tokens USING btree (token_hash);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens USING btree (user_id);
