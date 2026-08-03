DROP INDEX IF EXISTS idx_counselling_inquiries_inquiry_type;

ALTER TABLE counselling_inquiries
    DROP CONSTRAINT IF EXISTS counselling_inquiries_inquiry_type_check;

ALTER TABLE counselling_inquiries
    DROP COLUMN IF EXISTS inquiry_type;
