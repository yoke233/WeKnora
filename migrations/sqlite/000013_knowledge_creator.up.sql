-- Mirrors versioned migration 000091_knowledge_creator.
-- Existing rows intentionally remain NULL because no reliable uploader history exists.
ALTER TABLE knowledges ADD COLUMN creator_id VARCHAR(36);
