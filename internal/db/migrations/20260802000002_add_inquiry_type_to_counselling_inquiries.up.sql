ALTER TABLE counselling_inquiries
    ADD COLUMN inquiry_type VARCHAR(20) NOT NULL DEFAULT 'counselling';

ALTER TABLE counselling_inquiries
    ADD CONSTRAINT counselling_inquiries_inquiry_type_check
        CHECK (inquiry_type IN ('counselling', 'brochure'));

CREATE INDEX idx_counselling_inquiries_inquiry_type
    ON counselling_inquiries (inquiry_type, created_at DESC);
