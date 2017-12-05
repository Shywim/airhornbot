package service

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
)

// Sound represents a sound clip
type Sound struct {
	ID      string `json:"id"`
	GuildID string
	Name    string `json:"name"`

	// Link to a gif url
	Gif string `json:"gif"`

	// Weight adjust how likely it is this song will play, higher = more likely
	Weight int `json:"weight"`

	// Command to type in channel
	Commands       []string `json:"commands"`
	CommandsString string

	FilePath string
}

// SaveSound saves a sound to the db
func SaveSound(gID string, s *Sound, commands []string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec(`INSERT INTO sound (guildID, name, gif, weight, filepath) VALUES ($1, $2, $3, $4, $5)`,
		gID,
		s.Name,
		s.Gif,
		s.Weight)
	if err != nil {
		tx.Rollback()
		return err
	}
	soundID := res.LastInsertId

	for command := range commands {
		res, err = tx.Exec("INSERT INTO command (soundId, guildId, command) VALUES ($1, $2, $3)",
			soundID,
			gID,
			command)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// SaveAudio write the sound to a file
func SaveAudio(a io.Reader, n string) error {
	// check user directory exists
	_, err := os.Stat(config.DataPath)
	if os.IsNotExist(err) {
		os.Mkdir(config.DataPath, os.ModePerm)
	} else if err != nil {
		return err
	}

	// create file
	out, err := os.Create(filepath.Join(config.DataPath, n))
	if err != nil {
		return err
	}

	// encore file
	io.Copy(out, a)

	return nil
}

// UpdateSound update a sound in DB
func UpdateSound(gID string, sID string, s *Sound, commands []string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE sound SET guildID = $1, name = $2, gif = $3, weight = $4 WHERE id = $5",
		gID,
		s.Name,
		s.Gif,
		s.Weight,
		sID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// delete every command associated to the sound
	_, err = tx.Exec("DELETE FROM command WHERE soundId = $1 AND guildId = $2", sID, gID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// add new commands
	for command := range commands {
		_, err = tx.Exec("INSERT INTO command (soundId, guildId, command) VALUES ($1, $2, $3)",
			sID,
			gID,
			command)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// DeleteSound delete a sound from the DB
func DeleteSound(gID string, sID string) error {
	// TODO: delete also the sound file?
	db, err := getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM sound WHERE id = $1 AND guildId = $2", sID, gID)

	return err
}

func GetSound(ID string) (*Sound, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow("SELECT id, name, weight, filepath FROM sound WHERE id = $1", ID)

	sound, err := buildSound(row)
	if err != nil {
		return nil, err
	}

	row = db.QueryRow("SELECT command FROM command WHERE soundId = $1", ID)

	var command string
	if err := row.Scan(&command); err != nil {
		return nil, err
	}
	sound.Commands = []string{command}
	sound.CommandsString = command

	return sound, nil
}

// GetSoundsByCommand return all sounds for a given command
func GetSoundsByCommand(command, guildID string) ([]*Sound, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT soundId FROM command WHERE guildId = $1 AND command = $2", guildID, command)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var sounds []*Sound
	for rows.Next() {
		var soundID int
		if err := rows.Scan(&soundID); err != nil {
			return nil, err
		}

		row := db.QueryRow("SELECT id, name, weight, filepath FROM sound WHERE id = $1", soundID)

		sound, err := buildSound(row)
		if err != nil {
			return nil, err
		}
		sounds = append(sounds, sound)
	}

	return sounds, nil
}

// GetSoundsByGuild return all sounds for a given Guild
func GetSoundsByGuild(guildID string) ([]*Sound, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT id, name, weight, filepath FROM sound WHERE guildId = $1", guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return buildSounds(db, rows)
}

// FilterByCommand filter a sound array by command
func FilterByCommand(c string, s []*Sound) (r []*Sound) {
	for _, sound := range s {
		for _, command := range sound.Commands {
			if c == command {
				r = append(r, sound)
				break
			}
		}
	}

	return r
}

func buildSound(row *sql.Row) (*Sound, error) {
	var sound Sound
	var gif sql.NullString // gif can be null
	if err := row.Scan(&sound.ID, &sound.Name, &sound.Weight, &sound.FilePath); err != nil {
		return nil, err
	}

	sound.Gif = gif.String
	return &sound, nil
}

func buildSounds(db *sql.DB, rows *sql.Rows) ([]*Sound, error) {
	var sounds []*Sound
	for rows.Next() {
		var sound Sound
		if err := rows.Scan(&sound.ID, &sound.Name, &sound.Weight, &sound.FilePath); err != nil {
			return nil, err
		}

		commandsRows, err := db.Query("SELECT command FROM command WHERE soundId = $1", sound.ID)
		if err != nil {
			return nil, err
		}

		var commands []string
		for commandsRows.Next() {
			var command string
			if err := commandsRows.Scan(&command); err != nil {
				return nil, err
			}

			commands = append(commands, command)
			if sound.CommandsString != "" {
				sound.CommandsString = sound.CommandsString + ","
			}
			sound.CommandsString = sound.CommandsString + command
		}
		sound.Commands = commands

		sounds = append(sounds, &sound)
	}

	return sounds, nil
}

// DefaultSounds are a set of default sounds available to every servers
var DefaultSounds = []*Sound{
	{
		Name:     "airhorn_default",
		Weight:   1000,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_default.dca",
	},
	{
		Name:     "airhorn_reverb",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_reverb.dca",
	},
	{
		Name:     "airhorn_spam",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_spam.dca",
	},
	{
		Name:     "airhorn_tripletap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_tripletap.dca",
	},
	{
		Name:     "airhorn_fourtap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_fourtap.dca",
	},
	{
		Name:     "airhorn_distant",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_distant.dca",
	},
	{
		Name:     "airhorn_echo",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_echo.dca",
	},
	{
		Name:     "airhorn_clownfull",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_clownfull.dca",
	},
	{
		Name:     "airhorn_clownshort",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_clownshort.dca",
	},
	{
		Name:     "airhorn_clownspam",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_clownspam.dca",
	},
	{
		Name:     "airhorn_highfartlong",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_highfartlong.dca",
	},
	{
		Name:     "airhorn_highfartshort",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_highfartshot.dca",
	},
	{
		Name:     "airhorn_midshort",
		Weight:   100,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_midshort.dca",
	},
	{
		Name:     "airhorn_truck",
		Weight:   10,
		Commands: []string{"airhorn"},
		FilePath: "../audio/airhorn_truck.dca",
	},
	{
		Name:     "another_one",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "../audio/another_one.dca",
	},
	{
		Name:     "another_one_classic",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "../audio/another_one_classic.dca",
	},
	{
		Name:     "another_one_echo",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "../audio/another_one_echo.dca",
	},
	{
		Name:     "jc_realfull",
		Weight:   1,
		Commands: []string{"cena"},
		FilePath: "../audio/jc_realfull.dca",
	},
	{
		Name:     "cow_herd",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "../audio/cow_herd.dca",
	},
	{
		Name:     "cow_moo",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "../audio/cow_moo.dca",
	},
	{
		Name:     "cow_x3",
		Weight:   1,
		Commands: []string{"stan"},
		FilePath: "../audio/cow_x3.dca",
	},
	{
		Name:     "birthday_horn",
		Weight:   50,
		Commands: []string{"bday"},
		FilePath: "../audio/birthday_horn.dca",
	},
	{
		Name:     "birthday_horn3",
		Weight:   30,
		Commands: []string{"bday"},
		FilePath: "../audio/birthday_horn3.dca",
	},
	{
		Name:     "birthday_sadhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "../audio/birthday_sadhorn.dca",
	},
	{
		Name:     "birthday_weakhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "../audio/birthday_weakhorn.dca",
	},
	{
		Name:     "wow_thatscool",
		Weight:   1,
		Commands: []string{"wtc"},
		FilePath: "../audio/wow_thatscool.dca",
	},
}
