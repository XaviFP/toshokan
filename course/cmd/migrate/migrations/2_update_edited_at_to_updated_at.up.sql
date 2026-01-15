ALTER TABLE lessons
    RENAME COLUMN edited_at TO updated_at;

ALTER TABLE courses
    RENAME COLUMN edited_at TO updated_at;