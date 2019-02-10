# lines that start with # are ignored
CREATE TABLE Client (
    id SERIAL PRIMARY KEY,
    twitch_user_id TEXT NOT NULL,
    authentication_token TEXT NOT NULL,
    UNIQUE(twitch_user_id)
);
