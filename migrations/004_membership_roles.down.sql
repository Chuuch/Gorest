ALTER TABLE memberships DROP CONSTRAINT IF EXISTS memberships_role_check;

UPDATE memberships
SET role = 'admin'
WHERE role IN ('owner', 'admin');

ALTER TABLE memberships
ADD CONSTRAINT memberhips_role_check
CHECK (role IN ('admin', 'member'));
