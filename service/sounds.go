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

	FilePath string `json:"filepath"`
}

// Command represents a
type Command struct {
	ID      string `json:"id"`
	GuildID string `json:"guildId"`
	SoundID string `json:"soundId"`

	// Command to type in channel
	Command string `json:"command"`
	// Weight adjust how likely it is this song that will play, higher = more likely
	Weight int `json:"weight"`
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
	// TODO: also delete the sound file?
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
		FilePath: "./default_sounds/airhorn_default.dca",
	},
	{
		Name:     "airhorn_reverb",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_reverb.dca",
	},
	{
		Name:     "airhorn_spam",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_spam.dca",
	},
	{
		Name:     "airhorn_tripletap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_tripletap.dca",
	},
	{
		Name:     "airhorn_fourtap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_fourtap.dca",
	},
	{
		Name:     "airhorn_distant",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_distant.dca",
	},
	{
		Name:     "airhorn_echo",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_echo.dca",
	},
	{
		Name:     "airhorn_clownfull",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_clownfull.dca",
	},
	{
		Name:     "airhorn_clownshort",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_clownshort.dca",
	},
	{
		Name:     "airhorn_clownspam",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_clownspam.dca",
	},
	{
		Name:     "airhorn_highfartlong",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_highfartlong.dca",
	},
	{
		Name:     "airhorn_highfartshort",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_highfartshot.dca",
	},
	{
		Name:     "airhorn_midshort",
		Weight:   100,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_midshort.dca",
	},
	{
		Name:     "airhorn_truck",
		Weight:   10,
		Commands: []string{"airhorn"},
		FilePath: "./default_sounds/airhorn_truck.dca",
	},
	{
		Name:     "another_one",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "./default_sounds/another_one.dca",
	},
	{
		Name:     "another_one_classic",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "./default_sounds/another_one_classic.dca",
	},
	{
		Name:     "another_one_echo",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "./default_sounds/another_one_echo.dca",
	},
	{
		Name:     "jc_realfull",
		Weight:   1,
		Commands: []string{"cena"},
		FilePath: "./default_sounds/jc_realfull.dca",
	},
	{
		Name:     "cow_herd",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "./default_sounds/cow_herd.dca",
	},
	{
		Name:     "cow_moo",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "./default_sounds/cow_moo.dca",
	},
	{
		Name:     "cow_x3",
		Weight:   1,
		Commands: []string{"stan"},
		FilePath: "./default_sounds/cow_x3.dca",
	},
	{
		Name:     "birthday_horn",
		Weight:   50,
		Commands: []string{"bday"},
		FilePath: "./default_sounds/birthday_horn.dca",
	},
	{
		Name:     "birthday_horn3",
		Weight:   30,
		Commands: []string{"bday"},
		FilePath: "./default_sounds/birthday_horn3.dca",
	},
	{
		Name:     "birthday_sadhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "./default_sounds/birthday_sadhorn.dca",
	},
	{
		Name:     "birthday_weakhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "./default_sounds/birthday_weakhorn.dca",
	},
	{
		Name:     "wow_thatscool",
		Weight:   1,
		Commands: []string{"wtc"},
		FilePath: "./default_sounds/wow_thatscool.dca",
	},

	{ // quel beau canasson
		Name:     "test",
		Weight:   1,
		Commands: []string{"test1"},
		FilePath: "./0a35492f-0175-457f-983d-c69ea1a17d8c",
	},
	{ // k-on ohayo
		Name:     "test",
		Weight:   1,
		Commands: []string{"test2"},
		FilePath: "./0e6fa4a8-7ad1-4931-86fe-d721fd58c4ef",
	},
	{ // chat chelou
		Name:     "test",
		Weight:   1,
		Commands: []string{"test3"},
		FilePath: "./2d8da8da-ef9d-48a0-b7bb-a9c7ba6708dc",
	},
	{ // NON mario
		Name:     "test",
		Weight:   1,
		Commands: []string{"test4"},
		FilePath: "./3ae4948e-2844-4225-b2c7-c98bd26d3d6d",
	},
	{ // yay ononoki
		Name:     "test",
		Weight:   1,
		Commands: []string{"test5"},
		FilePath: "./3d4400d5-4f67-4b1c-949d-178ecd1b5f7a",
	},
	{ // jpc merde
		Name:     "test",
		Weight:   1,
		Commands: []string{"test6"},
		FilePath: "./5bc2772b-42b0-482b-84a0-5abed1699c5b",
	},
	{ // phoque aaaaah
		Name:     "test",
		Weight:   1,
		Commands: []string{"test7"},
		FilePath: "./6c2d8ab0-8b89-4eb9-8224-3b79b9fcad5c",
	},
	{ // COIN
		Name:     "test",
		Weight:   1,
		Commands: []string{"test8"},
		FilePath: "./6cfca518-cd2f-4921-b7fa-78b416184689",
	},
	{ // aaaaaaaaaaaaaaaaaaaaah du cul
		Name:     "test",
		Weight:   1,
		Commands: []string{"test9"},
		FilePath: "./7ae0e157-6a69-4cc6-9c41-c84a7ccf8094",
	},
	{ // blblbl
		Name:     "test",
		Weight:   1,
		Commands: []string{"test10"},
		FilePath: "./37f3f4fb-4310-4177-b3d6-bfd6bf0c284f",
	},
	{ // denis brogniard
		Name:     "test",
		Weight:   1,
		Commands: []string{"test11"},
		FilePath: "./57c32237-d733-4a0f-a85c-035b4ef1b8a3",
	},
	{ // vas y monte moi
		Name:     "test",
		Weight:   1,
		Commands: []string{"test12"},
		FilePath: "./89a3320f-ff51-4913-9b44-bb5d0ad29cf5",
	},
	{ // tu viens d'appuyer sur mon ventre et j'ai des gazs
		Name:     "test",
		Weight:   1,
		Commands: []string{"test13"},
		FilePath: "./745ebcbb-ff43-437f-8d81-543190fa04c5",
	},
	{ // kuwah
		Name:     "test",
		Weight:   1,
		Commands: []string{"test14"},
		FilePath: "./0919d54a-7c25-4ab8-b947-8822034890af",
	},
	{ // margarine svp
		Name:     "test",
		Weight:   1,
		Commands: []string{"test15"},
		FilePath: "./53123f52-d5a0-4be1-af6b-2f3ac98f16ef",
	},
	{ // merry christmas
		Name:     "test",
		Weight:   1,
		Commands: []string{"test16"},
		FilePath: "./58078b89-3468-4c54-a106-e6cf3ec39e6f",
	},
	{ // nononononono
		Name:     "test",
		Weight:   1,
		Commands: []string{"test17"},
		FilePath: "./8058044c-075e-46fe-9a46-9319f1d9a864",
	},
	{ // GET BENT
		Name:     "test",
		Weight:   1,
		Commands: []string{"test18"},
		FilePath: "./73798386-5890-457f-9264-002c98d4acd9",
	},
	{ // unacceptable
		Name:     "test",
		Weight:   1,
		Commands: []string{"test19"},
		FilePath: "./93533969-6ae7-495e-9d85-87e397814550",
	},
	{ // shrug
		Name:     "test",
		Weight:   1,
		Commands: []string{"test20"},
		FilePath: "./a58f33b9-3e2a-48ef-9ad6-de223fab7f51",
	},
	{ // monte moi cet obstacle
		Name:     "test",
		Weight:   1,
		Commands: []string{"test21"},
		FilePath: "./a026789d-3db1-4f55-82de-8730b11cf25b",
	},
	{ // mandragore aaaaaaaaaaah
		Name:     "test",
		Weight:   1,
		Commands: []string{"test22"},
		FilePath: "./aaaa7b0d-930b-4db0-88fc-8ee0db920bd4",
	},
	{ // PUTEUH
		Name:     "test",
		Weight:   1,
		Commands: []string{"test23"},
		FilePath: "./c0753143-2e19-4429-84f7-55e5ced7f19c",
	},
	{ // shrug
		Name:     "test",
		Weight:   1,
		Commands: []string{"test24"},
		FilePath: "./cdd6ed2c-c73e-4dba-a375-1595ffee12c7",
	},
	{ // yay onichan peace peace
		Name:     "test",
		Weight:   1,
		Commands: []string{"test25"},
		FilePath: "./d5fb9e4c-6ce1-4ee0-9d5a-dc1f4e6f3d9c",
	},
	{ // monte moi
		Name:     "test",
		Weight:   1,
		Commands: []string{"test26"},
		FilePath: "./d9e369d3-5d76-426c-962d-ed9d2c4457fe",
	},
	{ // yay onichan peace peace
		Name:     "test",
		Weight:   1,
		Commands: []string{"test27"},
		FilePath: "./ded6c06c-aad5-4128-b80e-dd2ffdf78daf",
	},
	{ // abdoula rot
		Name:     "test",
		Weight:   1,
		Commands: []string{"test28"},
		FilePath: "./e1cc3aa6-3c1c-48bd-9b6b-b75f6df369a2",
	},
	{ // i hate you
		Name:     "test",
		Weight:   1,
		Commands: []string{"test29"},
		FilePath: "./e5b7f0b3-8521-4e48-9b42-a5f537a871c7",
	},
	{ // shrug
		Name:     "test",
		Weight:   1,
		Commands: []string{"test30"},
		FilePath: "./e9c6a660-466e-444b-876e-9e310e44eee4",
	},
	{ // shrug
		Name:     "test",
		Weight:   1,
		Commands: []string{"test31"},
		FilePath: "./edcd4e2a-c4b7-40ed-a1d9-c90a0043cfbf",
	},
	{ // ui
		Name:     "test",
		Weight:   1,
		Commands: []string{"test32"},
		FilePath: "./f04a4278-7158-46e7-bbc9-cb7afe3a67c4",
	},
	{ // ohayo
		Name:     "test",
		Weight:   1,
		Commands: []string{"test33"},
		FilePath: "./fd767c93-411e-4dbb-9e89-9ffea8d19e73",
	},
}
