package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/garyburd/redigo/redis"
	"github.com/jonas747/dca"
	"github.com/orcaman/concurrent-map"
	"github.com/shywim/airhornbot/common"
)

const (
	pluginsDir = "./plugins/"
)

var (
	// discordgo session
	discord *discordgo.Session

	// Redis client connection (used for stats)
	redisPool *redis.Pool

	// Map of Guild id's to *Play channels, used for queuing and rate-limiting guilds
	queues = cmap.New()

	// Sound encoding settings
	bitrate = 128

	// Max queue size for each Guild
	maxQueueSize = 5

	// Owner
	owner string

	userAudioPath *string

	plugins map[string]*airhornPlugin
)

type airhornPlugin struct {
	name     string
	handle   func(string) bool
	getSound func(string) [][]byte
}

// Play represents an individual use of the !airhorn command
type Play struct {
	GuildID   string
	ChannelID string
	UserID    string
	Sound     *common.Sound

	// The next play to occur after this, only used for chaining sounds like anotha
	Next *Play

	// If true, this was a forced play using a specific airhorn sound name
	Forced bool
}

// Create a Sound struct
func createSound(Name string, Weight int, Gif string) *common.Sound {
	return &common.Sound{
		Name:   Name,
		Gif:    Gif,
		Weight: Weight,
	}
}

func random(s []*common.Sound) *common.Sound {
	var (
		i     int
		total int
	)

	for _, sound := range s {
		total += sound.Weight
	}
	number := randomRange(0, total)
	for _, sound := range s {
		i += sound.Weight

		if number < i {
			return sound
		}
	}
	return nil
}

// Attempt to join a voice channel
func voiceConnect(gid, cid string) (vc *discordgo.VoiceConnection, err error) {
	log.WithFields(log.Fields{
		"guildId":   gid,
		"channelId": cid,
	}).Info("Connecting to voice channel")
	return discord.ChannelVoiceJoin(gid, cid, false, false)
}

// Attempt to close the active voice connection
func voiceDisconnect(vc *discordgo.VoiceConnection) {
	if vc != nil {
		log.Info("Disconnecting active voice connection")
		vc.Disconnect()
		return
	}

	log.Warning("Disconnect called but there were no active voice connection")
}

// Attempts to find the current users voice channel inside a given guild
func getCurrentVoiceChannel(user *discordgo.User, guild *discordgo.Guild) *discordgo.Channel {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == user.ID {
			channel, _ := discord.State.Channel(vs.ChannelID)
			return channel
		}
	}
	return nil
}

// Returns a random integer between min and max
func randomRange(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min) + min
}

func loadSoundFromPlugin(pluginName, name string) (buffer [][]byte, err error) {
	plugin := plugins[pluginName]
	if plugin == nil {
		return nil, errors.New("Couldn't find a matching plugin for sound")
	}

	soundData := plugin.getSound(name)
	if soundData == nil {
		return nil, errors.New("Failed to get sound from plugin")
	}

	return soundData, nil
}

func loadSound(s *common.Sound) (buffer [][]byte, err error) {
	if strings.HasPrefix(s.FilePath, "@plugin/") {
		return loadSoundFromPlugin(strings.TrimPrefix(s.FilePath, "@plugin/"), s.Name)
	}

	file, err := os.Open(s.FilePath)
	if err != nil {
		return nil, err
	}

	decoder := dca.NewDecoder(file)

	for {
		frame, err := decoder.OpusFrame()
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return buffer, nil
			}

			fmt.Println("error reading from dca file :", err)
			return nil, err
		}

		buffer = append(buffer, frame)
	}
}

func doPlay(soundData [][]byte, vc *discordgo.VoiceConnection) {
	vc.Speaking(true)
	defer vc.Speaking(false)

	for _, buff := range soundData {
		vc.OpusSend <- buff
	}
}

// Prepares a play
func createPlay(user *discordgo.User, guild *discordgo.Guild, coll []*common.Sound) *Play {
	// Grab the users voice channel
	channel := getCurrentVoiceChannel(user, guild)
	if channel == nil {
		log.WithFields(log.Fields{
			"user":  user.ID,
			"guild": guild.ID,
		}).Warning("Failed to find channel to play sound in")
		return nil
	}

	// Create the play
	play := &Play{
		GuildID:   guild.ID,
		ChannelID: channel.ID,
		UserID:    user.ID,
	}

	// If we didn't get passed a manual sound, generate a random one
	play.Sound = random(coll)

	return play
}

// Prepares and enqueues a play into the ratelimit/buffer guild queue
func enqueuePlay(user *discordgo.User, guild *discordgo.Guild, sounds []*common.Sound, cid string) {
	play := createPlay(user, guild, sounds)
	if play == nil {
		return
	}

	// Check if we already have a connection to this guild
	tmp, exists := queues.Get(guild.ID)

	if exists {
		queue := tmp.(chan *Play)
		if len(queue) < maxQueueSize {
			queue <- play
		}
	} else {
		queues.Set(guild.ID, make(chan *Play, maxQueueSize))
		playSound(play, nil, cid)
	}
}

func trackSoundStats(play *Play) {
	if redisPool == nil {
		return
	}

	conn := redisPool.Get()
	defer conn.Close()

	redisSoundID := play.Sound.ID
	// use sound name for stats only if this is a default sound (no ID)
	if redisSoundID == "" {
		redisSoundID = play.Sound.Name
	}

	conn.Send("INCR", fmt.Sprintf("airhorn:total"))
	conn.Send("INCR", fmt.Sprintf("airhorn:guild:%s:plays", play.GuildID))
	conn.Send("INCR", fmt.Sprintf("airhorn:guild:%s:soundstats:%s", play.GuildID, redisSoundID))
	conn.Send("SAdd", fmt.Sprintf("airhorn:users"), play.UserID)
	conn.Send("SAdd", fmt.Sprintf("airhorn:guilds"), play.GuildID)
	conn.Send("SAdd", fmt.Sprintf("airhorn:channels"), play.ChannelID)
	err := conn.Flush()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warning("Failed to track stats in redis")
	}
}

// Play a sound
func playSound(play *Play, vc *discordgo.VoiceConnection, cid string) (err error) {
	log.WithFields(log.Fields{
		"play": play,
	}).Info("Playing sound")

	// load sound file
	soundData, err := loadSound(play.Sound)
	if err != nil {
		log.WithError(err).Error("Failed to read sound file")
		return
	}

	if vc == nil {
		vc, err = voiceConnect(play.GuildID, play.ChannelID)
		// vc.Receive = false
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Failed to play sound")
			queues.Remove(play.GuildID)
			return err
		}
	}

	// If we need to change channels, do that now
	if vc.ChannelID != play.ChannelID {
		vc.ChangeChannel(play.ChannelID, false, false)
		time.Sleep(time.Millisecond * 125)
	}

	// Track stats for this play in redis
	go trackSoundStats(play)

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(time.Millisecond * 32)

	// Send gif if present
	if play.Sound.Gif != "" {
		discord.ChannelMessageSend(cid, play.Sound.Gif)
	}

	// Play the sound
	doPlay(soundData, vc)

	// If there is another song in the queue, recurse and play that
	tmp, exists := queues.Get(play.GuildID)

	if exists {
		queue := tmp.(chan *Play)
		if len(queue) > 0 {
			play := <-queue
			playSound(play, vc, cid)
			return nil
		}
	}

	// If the queue is empty, delete it
	time.Sleep(time.Millisecond * time.Duration(250))
	queues.Remove(play.GuildID)
	voiceDisconnect(vc)
	return nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	s.UpdateStatus(0, "airhorn.shywim.fr")
}

func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			return
		}
	}
}

func scontains(key string, options ...string) bool {
	for _, item := range options {
		if item == key {
			return true
		}
	}
	return false
}

func displayBotStats(cid string) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	users := 0
	for _, guild := range discord.State.Ready.Guilds {
		users += len(guild.Members)
	}

	w := &tabwriter.Writer{}
	buf := &bytes.Buffer{}

	w.Init(buf, 0, 4, 0, ' ', 0)
	fmt.Fprintf(w, "```\n")
	fmt.Fprintf(w, "Discordgo: \t%s\n", discordgo.VERSION)
	fmt.Fprintf(w, "Go: \t%s\n", runtime.Version())
	fmt.Fprintf(w, "Memory: \t%s / %s (%s total allocated)\n", humanize.Bytes(stats.Alloc), humanize.Bytes(stats.Sys), humanize.Bytes(stats.TotalAlloc))
	fmt.Fprintf(w, "Tasks: \t%d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "Servers: \t%d\n", len(discord.State.Ready.Guilds))
	fmt.Fprintf(w, "Users: \t%d\n", users)
	fmt.Fprintf(w, "```\n")
	w.Flush()
	discord.ChannelMessageSend(cid, buf.String())
}

/*func displayBotCommands(cid string) {
	w := &tabwriter.Writer{}
	buf := &bytes.Buffer{}

	w.Init(buf, 0, 4, 0, ' ', 0)
	fmt.Fprint(w, "```\n")
	for _, coll := range COLLECTIONS {
		if len(coll.Commands) == 1 {
			fmt.Fprintf(w, "%s: ", coll.Commands[0])
		} else {
			for i, comm := range coll.Commands {
				if i == 0 {
					fmt.Fprintf(w, "%s", comm)
				} else {
					fmt.Fprintf(w, ", %s", comm)
				}
			}
			fmt.Fprint(w, ": ")
		}

		for _, sound := range coll.Sounds {
			fmt.Fprintf(w, "\t%s\n", sound.Name)
		}
	}
	fmt.Fprint(w, "```\n")
	w.Flush()
	discord.ChannelMessageSend(cid, buf.String())
}*/

func utilSumRedisKeys(keys []string) (int, error) {
	var total int64

	values, err := common.UtilGetRedisValuesFor(redisPool, keys)
	if err != nil {
		return 0, err
	}
	for _, v := range values {
		total += int64(v.(int64))
	}
	return int(total), nil
}

func displayUserStats(cid, uid string) {
	conn := redisPool.Get()
	keys, err := conn.Do("KEYS", fmt.Sprintf("airhorn:user:%s:sound:*", uid))
	if err != nil {
		return
	}

	totalAirhorns, err := utilSumRedisKeys(keys.([]string))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error reading stats")
		return
	}
	discord.ChannelMessageSend(cid, fmt.Sprintf("Total Airhorns: %v", totalAirhorns))
}

func displayServerStats(cid, sid string) {
	conn := redisPool.Get()
	keys, err := conn.Do("KEYS", fmt.Sprintf("airhorn:guild:%s:sound:*", sid))
	if err != nil {
		return
	}

	totalAirhorns, err := utilSumRedisKeys(keys.([]string))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error reading stats")
		return
	}
	discord.ChannelMessageSend(cid, fmt.Sprintf("Total Airhorns: %v", totalAirhorns))
}

func utilGetMentioned(s *discordgo.Session, m *discordgo.MessageCreate) *discordgo.User {
	for _, mention := range m.Mentions {
		if mention.ID != s.State.Ready.User.ID {
			return mention
		}
	}
	return nil
}

func airhornBomb(cid string, guild *discordgo.Guild, user *discordgo.User, cs string) {
	count, _ := strconv.Atoi(cs)
	discord.ChannelMessageSend(cid, ":ok_hand:"+strings.Repeat(":trumpet:", count))

	// Cap it at something
	if count > 100 {
		return
	}

	airhornSounds := common.FilterByCommand("airhorn", common.DefaultSounds)

	play := createPlay(user, guild, airhornSounds)
	vc, err := voiceConnect(play.GuildID, play.ChannelID)
	if err != nil {
		return
	}

	for i := 0; i < count; i++ {
		//random(airhornSounds).Play(vc)
	}

	voiceDisconnect(vc)
}

// Handles bot operator messages, should be refactored (lmao)
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {
	if scontains(parts[1], "status") {
		displayBotStats(m.ChannelID)
	} else if scontains(parts[1], "stats") {
		if len(m.Mentions) >= 2 {
			displayUserStats(m.ChannelID, utilGetMentioned(s, m).ID)
		} else if len(parts) >= 3 {
			displayUserStats(m.ChannelID, parts[2])
		} else {
			displayServerStats(m.ChannelID, g.ID)
		}
	} else if scontains(parts[1], "bomb") && len(parts) >= 4 {
		airhornBomb(m.ChannelID, g, utilGetMentioned(s, m), parts[3])
	}
}

func handleMentionMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {
	if scontains(parts[1], "help") {
		//displayBotCommands(m.ChannelID)
	}
}

func findPluginForSound(name string) (sounds []*common.Sound) {
	for _, p := range plugins {
		if p.handle(name) {
			sound := &common.Sound{
				FilePath: "@plugin/" + p.name,
				Name:     name,
				Weight:   1,
			}
			sounds = append(sounds, sound)
		}
	}

	return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Content) <= 0 || (m.Content[0] != '!' && len(m.Mentions) < 1) {
		return
	}

	msg := strings.Replace(m.ContentWithMentionsReplaced(), s.State.Ready.User.Username, "username", 1)
	parts := strings.Split(strings.ToLower(msg), " ")

	channel, _ := discord.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": m.ID,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := discord.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": m.ID,
		}).Warning("Failed to grab guild")
		return
	}

	log.WithFields(log.Fields{
		"message": m.ID,
		"owner":   m.Author.ID,
	}).Info("Received message")

	// If this is a mention
	if len(m.Mentions) > 0 && len(parts) > 0 {
		mentioned := false
		for _, mention := range m.Mentions {
			mentioned = (mention.ID == s.State.Ready.User.ID)
			if mentioned {
				break
			}
		}

		if mentioned {
			// Bot control messages come from owner
			if m.Author.ID == owner {
				handleBotControlMessages(s, m, parts, guild)
			}
			handleMentionMessages(s, m, parts, guild)
		}
		return
	}

	command := strings.TrimPrefix(parts[0], "!")

	// filter default sounds
	sounds := common.FilterByCommand(command, common.DefaultSounds)
	conn := redisPool.Get()

	// get keys from redis
	r, err := conn.Do("KEYS", fmt.Sprintf("airhorn:guild:%s:sound:*", channel.GuildID))
	keys, err := redis.Strings(r, err)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot get sound keys for this guild")
		return
	}
	// get values from redis
	values, err := common.UtilGetRedisValuesFor(redisPool, keys)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot get sound values for this guild")
	}

	// unmarshal and check for command
	for _, s := range values {
		sound := common.Sound{}
		json.Unmarshal([]byte(s.([]byte)), &sound)
		sound.FilePath = filepath.Join(*userAudioPath, sound.ID)
		if sound.Command == command {
			sounds = append(sounds, &sound)
		}
	}

	// check plugins
	sounds = append(sounds, findPluginForSound(command)...)

	// if we found at least one sound, play it or them
	if len(sounds) > 0 {
		go enqueuePlay(m.Author, guild, sounds, m.ChannelID)
	} else {
		log.WithField("sound", command).Info("No sound found for this command")
	}
}

func loadPlugins() {
	files, err := ioutil.ReadDir(pluginsDir)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Couldn't load plugins directory")
	}

	for _, file := range files {
		if !file.IsDir() && strings.Contains(file.Name(), ".so") {
			p, err := plugin.Open(pluginsDir + file.Name())
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"plugin": file.Name(),
				}).Warn("Couldn't load plugin")
				continue
			}

			name, err := p.Lookup("Name")
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"plugin": file.Name(),
				}).Error("Couldn't load Name from plugin")
				continue
			}

			handleFunc, err := p.Lookup("Handle")
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"plugin": name,
				}).Warn("Couldn't load the Handle function from plugin")
			}

			getSoundFunc, err := p.Lookup("GetSound")
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"plugin": name,
				}).Warn("Couldn't load the GetSound function from plugin")
			}

			plug := &airhornPlugin{
				name:     name.(string),
				handle:   handleFunc.(func(string) bool),
				getSound: getSoundFunc.(func(string) [][]byte),
			}
			plugins[plug.name] = plug
			log.WithFields(log.Fields{
				"plugin": file.Name(),
			}).Info("Loaded plugin")
		}
	}
}

func main() {
	var (
		Token      = flag.String("t", "", "Discord Authentication Token")
		DataPath   = flag.String("d", "", "User uploaded audio path")
		Redis      = flag.String("r", "", "Redis Connection String")
		Shard      = flag.String("s", "", "Shard ID")
		ShardCount = flag.String("c", "", "Number of shards")
		Owner      = flag.String("o", "", "Owner ID")
		err        error
	)
	flag.Parse()

	loadPlugins()

	if *Owner != "" {
		owner = *Owner
	}
	userAudioPath = DataPath

	if *DataPath == "" {
		panic("A data directory must be passed!")
	}

	// If we got passed a redis server, try to connect
	if *Redis == "" {
		panic("A redis server is required")
	}

	// connect to redis
	log.Info("Connecting to redis...")
	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", *Redis)
		},
	}
	defer redisPool.Close()

	// test redis connection
	conn := redisPool.Get()
	_, err = conn.Do("PING")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Can't establish a connection to the redis server")
		return
	}
	conn.Close()

	// Create a discord session
	log.Info("Starting discord session...")
	discord, err = discordgo.New(fmt.Sprintf("Bot %v", *Token))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord session")
		return
	}

	// Set sharding info
	discord.ShardID, _ = strconv.Atoi(*Shard)
	discord.ShardCount, _ = strconv.Atoi(*ShardCount)

	if discord.ShardCount <= 0 {
		discord.ShardCount = 1
	}

	discord.AddHandler(onReady)
	discord.AddHandler(onGuildCreate)
	discord.AddHandler(onMessageCreate)

	err = discord.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord websocket connection")
		return
	}

	// We're running!
	log.Info("AIRHORNBOT is ready to horn it up.")

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
