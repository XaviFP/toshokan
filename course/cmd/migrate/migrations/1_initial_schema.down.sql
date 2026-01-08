BEGIN;

DROP INDEX IF EXISTS idx_lesson_decks_lesson_id;
DROP INDEX IF EXISTS idx_lessons_course_id_order;
DROP INDEX IF EXISTS idx_user_course_progress_user_id;

DROP TABLE IF EXISTS user_course_progress;
DROP TABLE IF EXISTS lesson_decks;
DROP TABLE IF EXISTS lessons;
DROP TABLE IF EXISTS courses;

COMMIT;
