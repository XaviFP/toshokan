CREATE TABLE IF NOT EXISTS user_card_level (
    user_id UUID NOT NULL,
    card_id UUID REFERENCES cards (id) ON DELETE CASCADE NOT NULL,
    lvl integer NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    edited_at TIMESTAMP WITH TIME ZONE,
    UNIQUE (card_id, user_id)
);

CREATE TABLE IF NOT EXISTS user_deck (
    user_id UUID NOT NULL,
    deck_id UUID REFERENCES decks (id) ON DELETE CASCADE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_practiced TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (deck_id, user_id)
);

CREATE TABLE IF NOT EXISTS card_practice (
    user_id UUID NOT NULL,
    answer_id UUID REFERENCES answers (id) ON DELETE CASCADE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);
