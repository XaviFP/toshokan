ALTER TABLE answers
    RENAME COLUMN updated_at TO edited_at;

ALTER TABLE cards
    RENAME COLUMN updated_at TO edited_at;

ALTER TABLE decks
    RENAME COLUMN updated_at TO edited_at;

ALTER TABLE user_card_level
    RENAME COLUMN updated_at TO edited_at;