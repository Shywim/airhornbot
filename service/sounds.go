package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Sound represents a sound clip
type Sound struct {
	ID      string `json:"id"`
	GuildID string `json:"guildId"`
	Name    string `json:"name"`

	// Link to a gif url
	Gif string `json:"gif"`

	// Weight adjust how likely it is this song will play, higher = more likely
	Weight int `json:"weight"`

	// Command to type in channel
	Commands       []string `json:"commands"`
	CommandsString string

	FilePath string `json:"filepath"`
}

// Save saves a sound to the db
func (s *Sound) Save() error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	isNew := s.ID == ""

	if isNew {
		q := tx.Rebind(`INSERT INTO sound (guildID, name, gif, weight, filepath) VALUES (?, ?, ?, ?, ?)`)
		s.ID, err = insertGetID(tx, q, s.GuildID, s.Name, s.Gif, s.Weight, s.FilePath)
	} else {
		q := tx.Rebind("UPDATE sound SET name = ?, gif = ?, weight = ? WHERE id = ? AND guildId = ?")
		_, err = tx.Exec(q, s.Name, s.Gif, s.Weight, s.ID, s.GuildID)
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	if !isNew {
		// delete every command associated to the sound
		q := tx.Rebind("DELETE FROM command WHERE soundId = ? AND guildId = ?")
		_, err = tx.Exec(q, s.ID, s.GuildID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	for _, command := range s.Commands {
		q := tx.Rebind("INSERT INTO command (soundId, guildId, command) VALUES (?, ?, ?)")
		_, err = tx.Exec(q, s.ID, s.GuildID, strings.TrimSpace(command))
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
	// TODO: delete also the sound file?
	q := db.Rebind("DELETE FROM sound WHERE id = ? AND guildId = ?")
	_, err := db.Exec(q, s.ID, s.GuildID)

	return err
}

// getCommands retrieve a sound's commands from database
func (s *Sound) getCommands() error {
	q := db.Rebind("SELECT command FROM command WHERE soundId = ?")
	rows, err := db.Query(q, s.ID)
	if err != nil {
		return err
	}

	for rows.Next() {
		var command string
		rows.Scan(&command)
		s.Commands = append(s.Commands, command)
	}

	s.CommandsString = strings.Join(s.Commands[:], ", ")
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
	q := db.Rebind("SELECT soundId FROM command WHERE guildId = ? AND command = ?")
	rows, err := db.Queryx(q, guildID, command)
	if err != nil {
		return nil, err
	}

	q = db.Rebind("SELECT id, name, weight, filepath FROM sound WHERE id = ?")
	var sounds []*Sound
	for rows.Next() {
		var soundID int
		rows.Scan(&soundID)

		row := db.QueryRowx(q, soundID)

		sound := Sound{}
		row.StructScan(&sound)
		sounds = append(sounds, &sound)
	}

	return sounds, nil
}

// GetSoundsByGuild return all sounds for a given Guild
func GetSoundsByGuild(guildID string) ([]*Sound, error) {
	q := db.Rebind("SELECT id, name, weight, filepath FROM sound WHERE guildId = ?")
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
			if c == command {
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
		Weight:   1000,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_default.dca",
	},
	{
		Name:     "airhorn_reverb",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_reverb.dca",
	},
	{
		Name:     "airhorn_spam",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_spam.dca",
	},
	{
		Name:     "airhorn_tripletap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_tripletap.dca",
	},
	{
		Name:     "airhorn_fourtap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_fourtap.dca",
	},
	{
		Name:     "airhorn_distant",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_distant.dca",
	},
	{
		Name:     "airhorn_echo",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_echo.dca",
	},
	{
		Name:     "airhorn_clownfull",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_clownfull.dca",
	},
	{
		Name:     "airhorn_clownshort",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_clownshort.dca",
	},
	{
		Name:     "airhorn_clownspam",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_clownspam.dca",
	},
	{
		Name:     "airhorn_highfartlong",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_highfartlong.dca",
	},
	{
		Name:     "airhorn_highfartshort",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_highfartshot.dca",
	},
	{
		Name:     "airhorn_midshort",
		Weight:   100,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_midshort.dca",
	},
	{
		Name:     "airhorn_truck",
		Weight:   10,
		Commands: []string{"airhorn"},
		FilePath: "./audio/airhorn_truck.dca",
	},
	{
		Name:     "another_one",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "./audio/another_one.dca",
	},
	{
		Name:     "another_one_classic",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "./audio/another_one_classic.dca",
	},
	{
		Name:     "another_one_echo",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "./audio/another_one_echo.dca",
	},
	{
		Name:     "jc_realfull",
		Weight:   1,
		Commands: []string{"cena"},
		FilePath: "./audio/jc_realfull.dca",
	},
	{
		Name:     "cow_herd",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "./audio/cow_herd.dca",
	},
	{
		Name:     "cow_moo",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "./audio/cow_moo.dca",
	},
	{
		Name:     "cow_x3",
		Weight:   1,
		Commands: []string{"stan"},
		FilePath: "./audio/cow_x3.dca",
	},
	{
		Name:     "birthday_horn",
		Weight:   50,
		Commands: []string{"bday"},
		FilePath: "./audio/birthday_horn.dca",
	},
	{
		Name:     "birthday_horn3",
		Weight:   30,
		Commands: []string{"bday"},
		FilePath: "./audio/birthday_horn3.dca",
	},
	{
		Name:     "birthday_sadhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "./audio/birthday_sadhorn.dca",
	},
	{
		Name:     "birthday_weakhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "./audio/birthday_weakhorn.dca",
	},
	{
		Name:     "wow_thatscool",
		Weight:   1,
		Commands: []string{"wtc"},
		FilePath: "./audio/wow_thatscool.dca",
	},
}
