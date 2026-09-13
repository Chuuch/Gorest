ALTER TABLE memberships DROP CONTRAINT IF EXISTS memberships_role_check;

UPDATE memberships
SET role = 'owner'
WHERE role = 'admin';

ALTER TABLE memberships
ADD CONSTRAINT memberships_role_check
CHECK (role IN ('owner', 'admin', 'member'));
