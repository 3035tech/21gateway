CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX companies_name_trgm_idx
    ON companies USING gin (name gin_trgm_ops)
    WHERE archived_at IS NULL;

CREATE INDEX companies_reference_trgm_idx
    ON companies USING gin (reference gin_trgm_ops)
    WHERE archived_at IS NULL;
