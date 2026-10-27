CREATE TABLE system_user_companies (
    user_id BIGINT NOT NULL REFERENCES system_users(id) ON DELETE CASCADE,
    company_id BIGINT NOT NULL REFERENCES companies(id),
    PRIMARY KEY (user_id, company_id)
);
CREATE INDEX idx_system_user_companies_company ON system_user_companies(company_id, user_id);
