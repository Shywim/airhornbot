package common

import "github.com/garyburd/redigo/redis"

// Sound represents a sound clip
type Sound struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Link to a gif url
	Gif string `json:"gif"`

	// Weight adjust how likely it is this song will play, higher = more likely
	Weight int `json:"weight"`

	// Command to type in channel
	Command string `json:"command"`

	FilePath string
}

// FilterByCommand filter a sound array by command
func FilterByCommand(c string, s []*Sound) (r []*Sound) {
	for _, sound := range s {
		if c == sound.Command {
			r = append(r, sound)
		}
	}

	return r
}

// UtilGetRedisValuesFor keys
func UtilGetRedisValuesFor(redisPool *redis.Pool, keys []string) (r []interface{}, err error) {
	conn := redisPool.Get()
	for _, key := range keys {
		v, err := conn.Do("GET", key)
		if err != nil {
			return nil, err
		}
		r = append(r, v)
	}
	return r, nil
}

var DefaultSounds = []*Sound{
	&Sound{
		Name:     "airhorn_default",
		Weight:   1000,
		Command:  "airhorn",
		FilePath: "audio/airhorn_default.dca",
	},
	&Sound{
		Name:     "airhorn_reverb",
		Weight:   800,
		Command:  "airhorn",
		FilePath: "audio/airhorn_reverb.dca",
	},
	&Sound{
		Name:     "airhorn_spam",
		Weight:   800,
		Command:  "airhorn",
		FilePath: "audio/airhorn_spam.dca",
	},
	&Sound{
		Name:     "airhorn_tripletap",
		Weight:   800,
		Command:  "airhorn",
		FilePath: "audio/airhorn_tripletap.dca",
	},
	&Sound{
		Name:     "airhorn_fourtap",
		Weight:   800,
		Command:  "airhorn",
		FilePath: "audio/airhorn_fourtap.dca",
	},
	&Sound{
		Name:     "airhorn_distant",
		Weight:   500,
		Command:  "airhorn",
		FilePath: "audio/airhorn_distant.dca",
	},
	&Sound{
		Name:     "airhorn_echo",
		Weight:   500,
		Command:  "airhorn",
		FilePath: "audio/airhorn_echo.dca",
	},
	&Sound{
		Name:     "airhorn_clownfull",
		Weight:   250,
		Command:  "airhorn",
		FilePath: "audio/airhorn_clownfull.dca",
	},
	&Sound{
		Name:     "airhorn_clownshort",
		Weight:   250,
		Command:  "airhorn",
		FilePath: "audio/airhorn_clownshort.dca",
	},
	&Sound{
		Name:     "airhorn_clownspam",
		Weight:   250,
		Command:  "airhorn",
		FilePath: "audio/airhorn_clownspam.dca",
	},
	&Sound{
		Name:     "airhorn_highfartlong",
		Weight:   200,
		Command:  "airhorn",
		FilePath: "audio/airhorn_highfartlong.dca",
	},
	&Sound{
		Name:     "airhorn_highfartshort",
		Weight:   200,
		Command:  "airhorn",
		FilePath: "audio/airhorn_highfartshot.dca",
	},
	&Sound{
		Name:     "airhorn_midshort",
		Weight:   100,
		Command:  "airhorn",
		FilePath: "audio/airhorn_midshort.dca",
	},
	&Sound{
		Name:     "airhorn_truck",
		Weight:   10,
		Command:  "airhorn",
		FilePath: "audio/airhorn_truck.dca",
	},
	&Sound{
		Name:     "another_one",
		Weight:   1,
		Command:  "anotha",
		FilePath: "audio/another_one.dca",
	},
	&Sound{
		Name:     "another_one_classic",
		Weight:   1,
		Command:  "anotha",
		FilePath: "audio/another_one_classic.dca",
	},
	&Sound{
		Name:     "another_one_echo",
		Weight:   1,
		Command:  "anotha",
		FilePath: "audio/another_one_echo.dca",
	},
	&Sound{
		Name:     "jc_realfull",
		Weight:   1,
		Command:  "cena",
		FilePath: "audio/jc_realfull.dca",
	},
	&Sound{
		Name:     "cow_herd",
		Weight:   10,
		Command:  "stan",
		FilePath: "audio/cow_herd.dca",
	},
	&Sound{
		Name:     "cow_moo",
		Weight:   10,
		Command:  "stan",
		FilePath: "audio/cow_moo.dca",
	},
	&Sound{
		Name:     "cow_x3",
		Weight:   1,
		Command:  "stan",
		FilePath: "audio/cow_x3.dca",
	},
	&Sound{
		Name:     "birthday_horn",
		Weight:   50,
		Command:  "bday",
		FilePath: "audio/birthday_horn.dca",
	},
	&Sound{
		Name:     "birthday_horn3",
		Weight:   30,
		Command:  "bday",
		FilePath: "audio/birthday_horn3.dca",
	},
	&Sound{
		Name:     "birthday_sadhorn",
		Weight:   25,
		Command:  "bday",
		FilePath: "audio/birthday_sadhorn.dca",
	},
	&Sound{
		Name:     "birthday_weakhorn",
		Weight:   25,
		Command:  "bday",
		FilePath: "audio/birthday_weakhorn.dca",
	},
	&Sound{
		Name:     "wow_thatscool",
		Weight:   1,
		Command:  "wtc",
		FilePath: "audio/wow_thatscool.dca",
	},
}
