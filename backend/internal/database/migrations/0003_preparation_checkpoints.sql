ALTER TABLE english_app.study_jobs
 ADD COLUMN settings jsonb,
 ADD COLUMN generator_version integer NOT NULL DEFAULT 0,
 ADD COLUMN attempt integer NOT NULL DEFAULT 0;
CREATE TABLE english_app.study_job_checkpoints (
 job_id text NOT NULL REFERENCES english_app.study_jobs(id) ON DELETE CASCADE,
 step_key text NOT NULL,
 generator_version integer NOT NULL,
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(job_id,step_key)
);
