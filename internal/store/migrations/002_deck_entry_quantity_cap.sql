-- Raise per-card quantity cap from 3 → 4 to match MaxCopiesPerCard.
-- SQLite cannot ALTER a CHECK constraint, so we recreate the table.

CREATE TABLE deck_entries_new (
    deck_id    TEXT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    card_code  TEXT NOT NULL REFERENCES cards(card_code),
    quantity   INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 4),
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (deck_id, card_code)
);

INSERT INTO deck_entries_new SELECT * FROM deck_entries;

DROP TABLE deck_entries;

ALTER TABLE deck_entries_new RENAME TO deck_entries;
