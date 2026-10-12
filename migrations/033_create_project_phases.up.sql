CREATE TABLE project_phases (
 id UUID PRIMARY KEY,
 organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
 project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 name TEXT NOT NULL,
 sort_order INTEGER NOT NULL,
 status TEXT NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'in_progress', 'done')),
 starts_on DATE,
 due_on DATE,
 version INTEGER NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL,
 CONSTRAINT project_phases_project_sort_unique UNIQUE (project_id, sort_order),
 CONSTRAINT project_phases_project_name_unique UNIQUE (project_id, name),
 CONSTRAINT project_phases_dates_check CHECK (starts_on IS NULL OR due_on IS NULL OR starts_on <= due_on)
);

CREATE INDEX idx_project_phases_organization_id ON project_phases (organization_id);
CREATE INDEX idx_project_phases_project_sort ON project_phases (project_id, sort_order);
CREATE INDEX idx_project_phases_project_status ON project_phases (project_id, status);
