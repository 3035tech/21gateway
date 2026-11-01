ALTER TABLE company_provider_routes
ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active'
CHECK (status IN ('active', 'planned'));

ALTER TABLE company_provider_routes
ADD CONSTRAINT company_provider_route_activation_check
CHECK ((channel = 'sms' AND status = 'active')
    OR (channel IN ('rcs', 'whatsapp') AND status = 'planned'));
