package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Sound represents a sound clip
type Sound struct {
	ID      string `json:"id" db:"id"`
	GuildID string `json:"guildId" db:"guild_id"`
	Name    string `json:"name" db:"name"`

	FilePath string `json:"filepath" db:"file_path"`

	Commands []Command `json:"commands"`
}

// Command represents a
type Command struct {
	ID      string `json:"id" db:"id"`
	GuildID string `json:"guildId" db:"guild_id"`
	SoundID string `json:"soundId" db:"sound_id"`

	// Command to type in channel
	Command string `json:"command" db:"command"`
	// Weight adjust how likely it is this song that will play, higher = more likely
	Weight int `json:"weight" db:"weight"`
}

// Save saves a sound to the db
func (s *Sound) Save() error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	isNew := s.ID == ""

	if isNew {
		q := tx.Rebind(`INSERT INTO sound (guild_id, name, file_path) VALUES (?, ?, ?)`)
		s.ID, err = insertGetID(tx, q, s.GuildID, s.Name, s.FilePath)
	} else {
		q := tx.Rebind("UPDATE sound SET name = ? WHERE id = ? AND guild_id = ?")
		_, err = tx.Exec(q, s.Name, s.ID, s.GuildID)
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	if !isNew {
		// delete every command associated to the sound
		q := tx.Rebind("DELETE FROM command WHERE sound_id = ? AND guild_id = ?")
		_, err = tx.Exec(q, s.ID, s.GuildID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	for _, command := range s.Commands {
		q := tx.Rebind("INSERT INTO command (sound_id, guild_id, command, weight) VALUES (?, ?, ?, ?)")
		_, err = tx.Exec(q, s.ID, s.GuildID, strings.TrimSpace(command.Command), command.Weight)
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

// Delete delete a sound from the DB
func (s *Sound) Delete() error {
	// TODO: also delete the sound file?
	q := db.Rebind("DELETE FROM sound WHERE id = ? AND guild_id = ?")
	_, err := db.Exec(q, s.ID, s.GuildID)

	return err
}

// getCommands retrieve a sound's commands from database
func (s *Sound) getCommands() error {
	err := db.Select(&s.Commands, "SELECT * FROM command WHERE sound_id = $1", s.ID)
	if err != nil {
		return err
	}

	log.WithField("sound", s).Info("Au secours")

	return nil
}

// FindCommand find a corresponding command struct
func (s *Sound) FindCommand(command string) *Command {
	for i := range s.Commands {
		if s.Commands[i].Command == command {
			return &s.Commands[i]
		}
	}
	return nil
}

// GetSound retrieve a sound from database
func GetSound(ID string) (*Sound, error) {
	s := Sound{}
	q := db.Rebind("SELECT * FROM sound WHERE id = ?")
	if err := db.QueryRowx(q, ID).StructScan(&s); err != nil {
		return nil, err
	}

	if err := s.getCommands(); err != nil {
		return nil, err
	}

	return &s, nil
}

// GetSoundsByCommand return all sounds for a given command
func GetSoundsByCommand(command, guildID string) ([]*Sound, error) {
	q := db.Rebind("SELECT sound_id FROM command WHERE guild_id = ? AND command = ?")
	rows, err := db.Queryx(q, guildID, command)
	if err != nil {
		return nil, err
	}

	q = db.Rebind("SELECT * FROM sound WHERE id = ?")
	var sounds []*Sound
	for rows.Next() {
		var soundID int
		rows.Scan(&soundID)

		row := db.QueryRowx(q, soundID)

		sound := Sound{}
		row.StructScan(&sound)

		if err := sound.getCommands(); err != nil {
			return nil, err
		}

		sounds = append(sounds, &sound)
	}

	return sounds, nil
}

// GetSoundsByGuild return all sounds for a given Guild
func GetSoundsByGuild(guildID string) ([]*Sound, error) {
	q := db.Rebind("SELECT * FROM sound WHERE guild_id = ?")
	rows, err := db.Queryx(q, guildID)
	if err != nil {
		return nil, err
	}

	var sounds []*Sound
	for rows.Next() {
		sound := Sound{}
		rows.StructScan(&sound)
		sounds = append(sounds, &sound)

		if err = sound.getCommands(); err != nil {
			return nil, err
		}
	}

	return sounds, nil
}

// FilterByCommand filter a sound array by command
func FilterByCommand(c string, s []*Sound) (r []*Sound) {
	for _, sound := range s {
		for _, command := range sound.Commands {
			if c == command.Command {
				r = append(r, sound)
				break
			}
		}
	}

	return r
}

// DefaultSounds are a set of default sounds available to every servers
var DefaultSounds = []*Sound{
	{
		Name:     "airhorn_default",
		Commands: []Command{{Command: "airhorn", Weight: 1000}},
		FilePath: "./default_sounds/airhorn_default.dca",
	},
	{
		Name:     "airhorn_reverb",
		Commands: []Command{{Command: "airhorn", Weight: 800}},
		FilePath: "./default_sounds/airhorn_reverb.dca",
	},
	{
		Name:     "airhorn_spam",
		Commands: []Command{{Command: "airhorn", Weight: 800}},
		FilePath: "./default_sounds/airhorn_spam.dca",
	},
	{
		Name:     "airhorn_tripletap",
		Commands: []Command{{Command: "airhorn", Weight: 800}},
		FilePath: "./default_sounds/airhorn_tripletap.dca",
	},
	{
		Name:     "airhorn_fourtap",
		Commands: []Command{{Command: "airhorn", Weight: 800}},
		FilePath: "./default_sounds/airhorn_fourtap.dca",
	},
	{
		Name:     "airhorn_distant",
		Commands: []Command{{Command: "airhorn", Weight: 500}},
		FilePath: "./default_sounds/airhorn_distant.dca",
	},
	{
		Name:     "airhorn_echo",
		Commands: []Command{{Command: "airhorn", Weight: 500}},
		FilePath: "./default_sounds/airhorn_echo.dca",
	},
	{
		Name:     "airhorn_clownfull",
		Commands: []Command{{Command: "airhorn", Weight: 250}},
		FilePath: "./default_sounds/airhorn_clownfull.dca",
	},
	{
		Name:     "airhorn_clownshort",
		Commands: []Command{{Command: "airhorn", Weight: 250}},
		FilePath: "./default_sounds/airhorn_clownshort.dca",
	},
	{
		Name:     "airhorn_clownspam",
		Commands: []Command{{Command: "airhorn", Weight: 250}},
		FilePath: "./default_sounds/airhorn_clownspam.dca",
	},
	{
		Name:     "airhorn_highfartlong",
		Commands: []Command{{Command: "airhorn", Weight: 200}},
		FilePath: "./default_sounds/airhorn_highfartlong.dca",
	},
	{
		Name:     "airhorn_highfartshort",
		Commands: []Command{{Command: "airhorn", Weight: 200}},
		FilePath: "./default_sounds/airhorn_highfartshot.dca",
	},
	{
		Name:     "airhorn_midshort",
		Commands: []Command{{Command: "airhorn", Weight: 100}},
		FilePath: "./default_sounds/airhorn_midshort.dca",
	},
	{
		Name:     "airhorn_truck",
		Commands: []Command{{Command: "airhorn", Weight: 10}},
		FilePath: "./default_sounds/airhorn_truck.dca",
	},
	{
		Name:     "another_one",
		Commands: []Command{{Command: "anotha", Weight: 1}},
		FilePath: "./default_sounds/another_one.dca",
	},
	{
		Name:     "another_one_classic",
		Commands: []Command{{Command: "anotha", Weight: 1}},
		FilePath: "./default_sounds/another_one_classic.dca",
	},
	{
		Name:     "another_one_echo",
		Commands: []Command{{Command: "anotha", Weight: 1}},
		FilePath: "./default_sounds/another_one_echo.dca",
	},
	{
		Name:     "jc_realfull",
		Commands: []Command{{Command: "cena", Weight: 1}},
		FilePath: "./default_sounds/jc_realfull.dca",
	},
	{
		Name:     "cow_herd",
		Commands: []Command{{Command: "stan", Weight: 10}},
		FilePath: "./default_sounds/cow_herd.dca",
	},
	{
		Name:     "cow_moo",
		Commands: []Command{{Command: "stan", Weight: 10}},
		FilePath: "./default_sounds/cow_moo.dca",
	},
	{
		Name:     "cow_x3",
		Commands: []Command{{Command: "stan", Weight: 1}},
		FilePath: "./default_sounds/cow_x3.dca",
	},
	{
		Name:     "birthday_horn",
		Commands: []Command{{Command: "bday", Weight: 50}},
		FilePath: "./default_sounds/birthday_horn.dca",
	},
	{
		Name:     "birthday_horn3",
		Commands: []Command{{Command: "bday", Weight: 30}},
		FilePath: "./default_sounds/birthday_horn3.dca",
	},
	{
		Name:     "birthday_sadhorn",
		Commands: []Command{{Command: "bday", Weight: 25}},
		FilePath: "./default_sounds/birthday_sadhorn.dca",
	},
	{
		Name:     "birthday_weakhorn",
		Commands: []Command{{Command: "bday", Weight: 25}},
		FilePath: "./default_sounds/birthday_weakhorn.dca",
	},
	{
		Name:     "wow_thatscool",
		Commands: []Command{{Command: "wtc", Weight: 1}},
		FilePath: "./default_sounds/wow_thatscool.dca",
	},

	// { // chat chelou
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test3"},
	// 	FilePath: "./2d8da8da-ef9d-48a0-b7bb-a9c7ba6708dc",
	// },
	// { // COIN
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test8"},
	// 	FilePath: "./6cfca518-cd2f-4921-b7fa-78b416184689",
	// },
	// { // merry christmas
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test16"},
	// 	FilePath: "./58078b89-3468-4c54-a106-e6cf3ec39e6f",
	// },
	// { // GET BENT
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test18"},
	// 	FilePath: "./73798386-5890-457f-9264-002c98d4acd9",
	// },
	// { // shrug
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test20"},
	// 	FilePath: "./a58f33b9-3e2a-48ef-9ad6-de223fab7f51",
	// },
	// { // mandragore aaaaaaaaaaah
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test22"},
	// 	FilePath: "./aaaa7b0d-930b-4db0-88fc-8ee0db920bd4",
	// },
	// { // shrug
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test24"},
	// 	FilePath: "./cdd6ed2c-c73e-4dba-a375-1595ffee12c7",
	// },
	// { // monte moi
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test26"},
	// 	FilePath: "./d9e369d3-5d76-426c-962d-ed9d2c4457fe",
	// },
	// { // yay onichan peace peace
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test27"},
	// 	FilePath: "./ded6c06c-aad5-4128-b80e-dd2ffdf78daf",
	// },
	// { // abdoula rot
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test28"},
	// 	FilePath: "./e1cc3aa6-3c1c-48bd-9b6b-b75f6df369a2",
	// },
	// { // i hate you
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test29"},
	// 	FilePath: "./e5b7f0b3-8521-4e48-9b42-a5f537a871c7",
	// },
	// { // shrug
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test30"},
	// 	FilePath: "./e9c6a660-466e-444b-876e-9e310e44eee4",
	// },
	// { // shrug
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test31"},
	// 	FilePath: "./edcd4e2a-c4b7-40ed-a1d9-c90a0043cfbf",
	// },
	// { // ohayo
	// 	Name:     "test",
	// 	Weight:   1,
	// 	Commands: []string{"test33"},
	// 	FilePath: "./fd767c93-411e-4dbb-9e89-9ffea8d19e73",
	// },
}
