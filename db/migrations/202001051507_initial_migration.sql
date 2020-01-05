-- +migrate Up
CREATE TABLE IF NOT EXISTS sound (
  id SERIAL PRIMARY KEY,
  guild_id TEXT NOT NULL,
  name TEXT NOT NULL,
  file_path TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS command (
  id SERIAL PRIMARY KEY,
  guild_id TEXT NOT NULL,
  sound_id INTEGER NOT NULL,
  command TEXT NOT NULL,
  weight INTEGER NOT NULL DEFAULT 1,
	FOREIGN KEY(sound_id) REFERENCES sound(id) ON DELETE CASCADE
);

-- +migrate Down
DROP TABLE sound;
DROP TABLE command;