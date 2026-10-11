ALTER TABLE estimate_runs
  ADD COLUMN IF NOT EXISTS estimated_hours NUMERIC;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'estimate_runs'
      AND column_name = 'risk_label'
) AND NOT EXISTS (
  SELECT 1
  FROM information_schema.columns
  WHERE table_name = 'estimate_runs'
    AND column_name = 'risk_label'
) THEN
  ALTER TABLE estimate_runs RENAME COLUMN risk_label TO risk_level;
  END IF;
END $$;
