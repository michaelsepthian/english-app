CREATE TABLE IF NOT EXISTS english_app.users (
 id text PRIMARY KEY, email text UNIQUE NOT NULL, password text NOT NULL,
 settings jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS english_app.sessions (
 token text PRIMARY KEY, user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 expires timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS english_app.words (
 id text PRIMARY KEY,user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 word text NOT NULL,payload jsonb NOT NULL,UNIQUE(user_id,word)
);
CREATE TABLE IF NOT EXISTS english_app.lessons (
 id text PRIMARY KEY,user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 day date NOT NULL,payload jsonb NOT NULL,draft jsonb NOT NULL DEFAULT '{}',
 seconds integer NOT NULL DEFAULT 0,last_tick timestamptz,UNIQUE(user_id,day)
);
CREATE TABLE IF NOT EXISTS english_app.attempts (
 lesson_id text PRIMARY KEY REFERENCES english_app.lessons(id) ON DELETE CASCADE,
 user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 day date NOT NULL,payload jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS english_app.messages (
 seq bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,id text UNIQUE NOT NULL,
 user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 role text NOT NULL CHECK(role IN ('user','assistant')),content text NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS messages_user ON english_app.messages(user_id,seq);
CREATE TABLE IF NOT EXISTS english_app.limits(key text PRIMARY KEY,count integer NOT NULL,expires timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS english_app.locks(key text PRIMARY KEY,expires timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS english_app.audio(hash text PRIMARY KEY,data bytea NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS english_app.chatgpt_meta(key text PRIMARY KEY,value text NOT NULL);
CREATE TABLE IF NOT EXISTS english_app.chatgpt_profiles(
 id text PRIMARY KEY,user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 client_id text NOT NULL,subject text NOT NULL DEFAULT '',email text NOT NULL DEFAULT '',
 model text NOT NULL DEFAULT '',active boolean NOT NULL DEFAULT false,credentials bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(user_id,client_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS one_active_chatgpt_per_user ON english_app.chatgpt_profiles(user_id) WHERE active=true;
CREATE TABLE IF NOT EXISTS english_app.chatgpt_pending(
 state text PRIMARY KEY,user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 payload bytea NOT NULL,expires timestamptz NOT NULL
);
