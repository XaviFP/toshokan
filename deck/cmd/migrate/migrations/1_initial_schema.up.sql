BEGIN;

CREATE TABLE IF NOT EXISTS decks (
    id UUID DEFAULT gen_random_uuid () PRIMARY KEY,
    author_id UUID NOT NULL,
    title TEXT NOT NULL,
    "description" TEXT NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS cards (
    id UUID DEFAULT gen_random_uuid () PRIMARY KEY,
    deck_id UUID REFERENCES decks (id),
    title TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS answers (
    id UUID DEFAULT gen_random_uuid () PRIMARY KEY,
    card_id UUID REFERENCES cards (id),
    "text" TEXT NOT NULL,
    is_correct boolean NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

COMMIT;