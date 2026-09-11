-- Description: Add immutable, nullable creator ownership to knowledge records.
-- Existing rows intentionally remain NULL because no reliable uploader history exists.
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS creator_id VARCHAR(36);
