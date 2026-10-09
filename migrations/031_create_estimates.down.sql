DROP INDEX IF EXISTS projects_estimate_run_id_ix;

ALTER TABLE projects
  DROP COLUMN IF EXISTS target_end_date,
  DROP COLUMN IF EXISTS estimated_hours,
  DROP COLUMN IF EXISTS estimate_run_id;

DROP TABLE IF EXISTS estimate_runs;
DROP TABLE IF EXISTS estimate_coefficients;
DROP TABLE IF EXISTS estimate_catalog_version;
