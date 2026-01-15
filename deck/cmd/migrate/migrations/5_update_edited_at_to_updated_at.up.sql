ALTER TABLE answers
    RENAME COLUMN edited_at TO updated_at;

ALTER TABLE cards
    RENAME COLUMN edited_at TO updated_at;

ALTER TABLE decks
    RENAME COLUMN edited_at TO updated_at;

ALTER TABLE user_card_level
    RENAME COLUMN edited_at TO updated_at;