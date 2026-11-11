ALTER TABLE english_app.lessons DROP CONSTRAINT IF EXISTS lessons_user_id_day_key;
ALTER TABLE english_app.lessons ADD COLUMN IF NOT EXISTS request_id text;
ALTER TABLE english_app.lessons ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
CREATE UNIQUE INDEX IF NOT EXISTS lesson_request_once ON english_app.lessons(user_id,request_id);
CREATE INDEX IF NOT EXISTS lessons_latest ON english_app.lessons(user_id,day,created_at DESC);
CREATE TABLE IF NOT EXISTS english_app.study_jobs(
 id text PRIMARY KEY,user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 request_id text NOT NULL,plan jsonb NOT NULL,status text NOT NULL DEFAULT 'queued',
 stage text NOT NULL DEFAULT 'Menyiapkan sesi',progress integer NOT NULL DEFAULT 0,
 error text NOT NULL DEFAULT '',lesson_id text REFERENCES english_app.lessons(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),UNIQUE(user_id,request_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS one_study_job_per_user ON english_app.study_jobs(user_id) WHERE status IN ('queued','running');
CREATE TABLE IF NOT EXISTS english_app.reading_checks(
 id text PRIMARY KEY,user_id text NOT NULL REFERENCES english_app.users(id) ON DELETE CASCADE,
 lesson_id text NOT NULL REFERENCES english_app.lessons(id) ON DELETE CASCADE,
 source_id text NOT NULL,payload jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
