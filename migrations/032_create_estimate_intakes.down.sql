ALTER TABLE estimate_runs
  DROP CONSTRAINT IF EXISTS estimate_runs_intake_id_fkey;

DROP TABLE IF EXISTS estimate_intakes;
