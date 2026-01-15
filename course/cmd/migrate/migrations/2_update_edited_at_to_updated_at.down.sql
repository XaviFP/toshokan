ALTER TABLE lessons
    RENAME COLUMN updated_at TO edited_at;

ALTER TABLE courses
    RENAME COLUMN updated_at TO edited_at;