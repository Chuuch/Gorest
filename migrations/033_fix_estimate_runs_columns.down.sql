DO $$
BEGIN
  IF EXISTS (
  SELECT 1
  FROM information_schema.columns
  WHERE table_name = 'estimate_runs'
  AND column_name = 'risk_level'
) AND NOT EXISTS (
  SELECT 1
  FROM information_schema.columns
  WHERE table_name = 'estimate_runs'
  AND column_name = 'risk_label'
) THEN
  ALTER TABLE estimate_runs RENAME COLUMN risk_level TO risk_label;
  END IF;
END $$;

ALTER TABLE estimate_runs DROP COLUMN IF EXISTS estimated_hours;
