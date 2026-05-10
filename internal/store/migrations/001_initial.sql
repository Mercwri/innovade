-- Migration 001: initial schema
-- Run via store.Migrate() on startup

CREATE TABLE IF NOT EXISTS card_sets (
    code     TEXT PRIMARY KEY,  -- "GD01", "ST01", "T"
    name     TEXT NOT NULL,     -- "Newtype Rising [GD01]"
    set_type TEXT NOT NULL      -- "booster" | "starter" | "token"
);

CREATE TABLE IF NOT EXISTS cards (
    -- Identity
    id                  TEXT PRIMARY KEY,  -- UUID
    card_code           TEXT NOT NULL UNIQUE,  -- "GD01-001"
    set_code            TEXT NOT NULL REFERENCES card_sets(code),
    name                TEXT NOT NULL,
    category            TEXT NOT NULL,  -- Unit | Command | Pilot | Base | Unit-token
    rarity              TEXT NOT NULL,  -- C | U | R | LR
    created_at          TEXT NOT NULL,

    -- Stats (0 when not applicable to this category)
    cost                INTEGER NOT NULL DEFAULT 0,
    level               INTEGER NOT NULL DEFAULT 0,
    ap                  INTEGER NOT NULL DEFAULT 0,
    hp                  INTEGER NOT NULL DEFAULT 0,

    -- Text
    effect              TEXT NOT NULL DEFAULT '',
    burst               TEXT NOT NULL DEFAULT '',
    link_requirement    TEXT NOT NULL DEFAULT '',

    -- Images
    default_image_path  TEXT NOT NULL DEFAULT ''
);

-- Colors: one row per color per card (currently always 1 row)
CREATE TABLE IF NOT EXISTS card_colors (
    card_code  TEXT NOT NULL REFERENCES cards(card_code) ON DELETE CASCADE,
    color      TEXT NOT NULL,  -- Blue | Red | Green | White | Purple
    PRIMARY KEY (card_code, color)
);

-- Faction / archetype types: up to 3 per card
CREATE TABLE IF NOT EXISTS card_types (
    card_code  TEXT NOT NULL REFERENCES cards(card_code) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (card_code, type)
);

-- Deployment locations: Space and/or Earth
CREATE TABLE IF NOT EXISTS card_locations (
    card_code  TEXT NOT NULL REFERENCES cards(card_code) ON DELETE CASCADE,
    location   TEXT NOT NULL,  -- Space | Earth
    PRIMARY KEY (card_code, location)
);

-- Alternate art image paths (index 0 = default, already in cards.default_image_path)
CREATE TABLE IF NOT EXISTS card_images (
    card_code   TEXT NOT NULL REFERENCES cards(card_code) ON DELETE CASCADE,
    sort_order  INTEGER NOT NULL,  -- 0 = default art
    image_path  TEXT NOT NULL,
    PRIMARY KEY (card_code, sort_order)
);

-- Decks
CREATE TABLE IF NOT EXISTS decks (
    id          TEXT PRIMARY KEY,  -- UUID generated on creation
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- Deck entries: one row per unique card in a deck
CREATE TABLE IF NOT EXISTS deck_entries (
    deck_id    TEXT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    card_code  TEXT NOT NULL REFERENCES cards(card_code),
    quantity   INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 3),
    sort_order INTEGER NOT NULL DEFAULT 0,  -- player-defined order
    PRIMARY KEY (deck_id, card_code)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_cards_set_code   ON cards(set_code);
CREATE INDEX IF NOT EXISTS idx_cards_category   ON cards(category);
CREATE INDEX IF NOT EXISTS idx_cards_rarity     ON cards(rarity);
CREATE INDEX IF NOT EXISTS idx_cards_cost       ON cards(cost);
CREATE INDEX IF NOT EXISTS idx_cards_level      ON cards(level);
CREATE INDEX IF NOT EXISTS idx_cards_name       ON cards(name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_card_colors_color ON card_colors(color);
CREATE INDEX IF NOT EXISTS idx_card_types_type   ON card_types(type);
CREATE INDEX IF NOT EXISTS idx_deck_entries_deck ON deck_entries(deck_id);