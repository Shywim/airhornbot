package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"plugin"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/garyburd/redigo/redis"
	"github.com/jonas747/dca"
	"gitlab.com/Shywim/airhornbot/service"
)

var (
	// discordgo session
	discord *discordgo.Session

	// Redis client connection (used for stats)
	redisPool *redis.Pool

	// Map of Guild id's to *Play channels, used for queuing and rate-limiting guilds
	queues = sync.Map{}

	// Map of Guild id's to disconnect timers
	dcTimers = sync.Map{}

	// Sound encoding settings
	bitrate = 128

	// Max queue size for each Guild
	maxQueueSize = 5

	// Owner
	owner string

	plugins = make(map[string]*airhornPlugin)
	cfg     service.Cfg
)

type airhornPlugin struct {
	name     string
	handle   func(string) bool
	getSound func(string) [][]byte
}

type play struct {
	GuildID   string
	ChannelID string
	UserID    string
	Sound     *service.Sound

	// The next play to occur after this, only used for chaining sounds like anotha
	Next *play

	// If true, this was a forced play using a specific airhorn sound name
	Forced bool
}

// Create a Sound struct
func createSound(Name string, Weight int, Gif string) *service.Sound {
	return &service.Sound{
		Name:   Name,
		Gif:    Gif,
		Weight: Weight,
	}
}

func random(s []*service.Sound) *service.Sound {
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
		_ = vc.Disconnect()
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

func loadSound(s *service.Sound) (buffer [][]byte, err error) {
	if strings.HasPrefix(s.FilePath, "@plugin/") {
		return loadSoundFromPlugin(strings.TrimPrefix(s.FilePath, "@plugin/"), s.Name)
	}

	file, err := os.Open(cfg.DataPath + string(os.PathSeparator) + s.FilePath)
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
	_ = vc.Speaking(true)
	defer func() {
		err := vc.Speaking(false)
		if err != nil {
			log.WithError(err).Warning("Error while stopping speaking")
		}
	}()

	for _, buff := range soundData {
		vc.OpusSend <- buff
	}
}

// Prepares a play
func createPlay(user *discordgo.User, guild *discordgo.Guild, coll []*service.Sound) *play {
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
	play := &play{
		GuildID:   guild.ID,
		ChannelID: channel.ID,
		UserID:    user.ID,
	}

	// If we didn't get passed a manual sound, generate a random one
	play.Sound = random(coll)

	return play
}

// Prepares and enqueues a play into the ratelimit/buffer guild queue
func enqueuePlay(user *discordgo.User, guild *discordgo.Guild, sounds []*service.Sound, cid string) {
	p := createPlay(user, guild, sounds)
	if p == nil {
		return
	}

	// Check if we already have a connection to this guild
	tmp, ok := queues.Load(guild.ID)

	if ok && tmp != nil {
		queue, _ := tmp.(chan *play)

		t, ok := dcTimers.Load(guild.ID)
		if ok && t != nil {
			// if the bot is waiting to disconnect, relaunch the queue immediately
			timer := t.(*time.Timer)
			cancelDcTimer(timer, guild.ID)
			playSound(p, nil, cid)
		} else if len(queue) < maxQueueSize {
			// else but the play in queue
			queue <- p
		}
	} else {
		queues.Store(guild.ID, make(chan *play, maxQueueSize))
		playSound(p, nil, cid)
	}
}

func trackSoundStats(p *play) {
	if redisPool == nil {
		return
	}

	conn := redisPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.WithError(err).Warning("Couldn't connect to redis to track stats")
		}
	}()

	redisSoundID := p.Sound.ID
	// use sound name for stats only if this is a default sound (no ID)
	if redisSoundID == "" {
		redisSoundID = p.Sound.Name
	}

	var err error
	if err = conn.Send("INCR", fmt.Sprintf("airhorn:total")); err != nil {
		log.WithError(err).Warning("failed to increment total count in redis")
	}
	if err = conn.Send("INCR", fmt.Sprintf("airhorn:guild:%s:plays", p.GuildID)); err != nil {
		log.WithError(err).Warning("failed to increment guild play count in redis")
	}
	if err = conn.Send("INCR", fmt.Sprintf("airhorn:guild:%s:soundstats:%s", p.GuildID, redisSoundID)); err != nil {
		log.WithError(err).Warning("failed to increment guild play count in redis")
	}
	if err = conn.Send("SAdd", fmt.Sprintf("airhorn:users"), p.UserID); err != nil {
		log.WithError(err).Warning("failed to increment user count in redis")
	}
	if err = conn.Send("SAdd", fmt.Sprintf("airhorn:guilds"), p.GuildID); err != nil {
		log.WithError(err).Warning("failed to increment guilds count in redis")
	}
	if err = conn.Send("SAdd", fmt.Sprintf("airhorn:channels"), p.ChannelID); err != nil {
		log.WithError(err).Warning("failed to increment channels count in redis")
	}

	err = conn.Flush()
	if err != nil {
		log.WithError(err).Warning("Failed to track stats in redis")
	}
}

// Play a sound
func playSound(p *play, vc *discordgo.VoiceConnection, cid string) {
	log.WithFields(log.Fields{
		"p": p,
	}).Info("Playing sound")

	// load sound file
	soundData, err := loadSound(p.Sound)
	if err != nil {
		log.WithError(err).Error("Failed to read sound file")
		return
	}

	if vc == nil {
		vc, err = voiceConnect(p.GuildID, p.ChannelID)

		if err != nil {
			log.WithError(err).Error("Failed to play sound")
			queues.Delete(p.GuildID)

			err = vc.Disconnect()
			if err != nil {
				log.WithError(err).Error("Failed to disconnect from voice channel")
			}
			return
		}
	}

	// If we need to change channels, do that now
	if vc.ChannelID != p.ChannelID {
		err = vc.ChangeChannel(p.ChannelID, false, false)
		if err != nil {
			log.WithError(err).Error("Failed to connect to voice channel")
			err = vc.Disconnect()
			if err != nil {
				log.WithError(err).Error("Failed to disconnect from voice channel")
			}
			return
		}
		time.Sleep(time.Millisecond * 125)
	}

	// Track stats for this p in redis
	go trackSoundStats(p)

	// Sleep for a specified amount of time before ping the sound
	time.Sleep(time.Millisecond * 32)

	// Send gif if present
	if p.Sound.Gif != "" {
		_, err = discord.ChannelMessageSend(cid, p.Sound.Gif)
		if err != nil {
			log.WithError(err).Warning("Failed to send gif to text channel")
		}
	}

	// Play the sound
	doPlay(soundData, vc)

	// If there is another song in the queue, recurse and p that
	tmp, exists := queues.Load(p.GuildID)

	if exists {
		queue := tmp.(chan *play)
		if len(queue) > 0 {
			p := <-queue
			playSound(p, vc, cid)
			return
		}

		close(queue)
	}

	// If the queue is empty, delete it
	endQueue(vc, p.GuildID)
	return
}

func disconnect(timer *time.Timer, vc *discordgo.VoiceConnection, gID string) {
	<-timer.C
	queues.Delete(gID)
	voiceDisconnect(vc)
	dcTimers.Delete(timer)
}

func cancelDcTimer(timer *time.Timer, gID string) {
	if !timer.Stop() {
		<-timer.C
	}
}

func endQueue(vc *discordgo.VoiceConnection, gID string) {
	t, ok := dcTimers.Load(gID)
	if ok && t != nil {
		timer := t.(*time.Timer)
		cancelDcTimer(timer, gID)
		timer.Reset(5 * time.Minute)
		return
	}

	timer := time.NewTimer(5 * time.Minute)
	go disconnect(timer, vc, gID)
	dcTimers.Store(gID, timer)
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	err := s.UpdateStatus(0, "airhorn.shywim.fr")
	if err != nil {
		log.WithError(err).Warning("Couldn't set status line")
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
	err := w.Flush()
	if err != nil {
		log.WithError(err).Error("Error while building stats message")
		return
	}

	_, err = discord.ChannelMessageSend(cid, buf.String())
	if err != nil {
		log.WithError(err).Error("Error while sending stats message")
	}
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

	values, err := service.UtilGetRedisValuesFor(redisPool, keys)
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
		log.WithError(err).Error("Error reading stats")
		return
	}

	_, err = discord.ChannelMessageSend(cid, fmt.Sprintf("Total Airhorns: %v", totalAirhorns))
	if err != nil {
		log.WithError(err).Error("Error sending user stats message")
	}
}

func displayServerStats(cid, sid string) {
	conn := redisPool.Get()
	keys, err := conn.Do("KEYS", fmt.Sprintf("airhorn:guild:%s:sound:*", sid))
	if err != nil {
		return
	}

	totalAirhorns, err := utilSumRedisKeys(keys.([]string))
	if err != nil {
		log.WithError(err).Error("Error reading stats")
		return
	}

	_, err = discord.ChannelMessageSend(cid, fmt.Sprintf("Total Airhorns: %v", totalAirhorns))
	if err != nil {
		log.WithError(err).Error("Error sending server stats message")
	}
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
	_, err := discord.ChannelMessageSend(cid, ":ok_hand:"+strings.Repeat(":trumpet:", count))
	if err != nil {
		log.WithError(err).Warning("Error sending bomb message")
	}

	// Cap it at something
	if count > 100 {
		return
	}

	airhornSounds := service.FilterByCommand("airhorn", service.DefaultSounds)

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
		// TODO: help
		//displayBotCommands(m.ChannelID)
	}
}

func findPluginForSound(name string) (sounds []*service.Sound) {
	for _, p := range plugins {
		if p.handle(name) {
			sound := &service.Sound{
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

	channel, _ := s.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": msg,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := s.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": msg,
			"from":    m.Author.ID,
		}).Warning("Failed to grab guild")
		return
	}

	log.WithFields(log.Fields{
		"message": msg,
		"from":    m.Author.ID,
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
	sounds := service.FilterByCommand(command, service.DefaultSounds)
	guildSounds, err := service.GetSoundsByCommand(command, channel.GuildID)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"guildId": channel.GuildID,
		}).Warn("Couldn't get sounds from db")
	}
	sounds = append(sounds, guildSounds...)

	// check plugins
	sounds = append(sounds, findPluginForSound(command)...)

	// if we found at least one sound, play it or them
	if len(sounds) > 0 {
		go enqueuePlay(m.Author, guild, sounds, m.ChannelID)
	} else {
		log.WithField("sound", command).Info("No sound found for this command")
	}
}

func loadPlugins(pluginsPath string) {
	files, err := ioutil.ReadDir(pluginsPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Couldn't load plugins directory")
		return
	}

	for _, file := range files {
		if !file.IsDir() && strings.Contains(file.Name(), ".so") {
			p, err := plugin.Open(pluginsPath + string(os.PathSeparator) + file.Name())
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
				name:     (*name.(*string)),
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
	var err error
	cfg, err = service.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Couldn't load configuration")
	}

	loadPlugins(cfg.PluginPath)

	if cfg.RedisHost != "" {
		// connect to redis
		log.Info("Connecting to redis...")
		redisPool = &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", cfg.RedisHost)
			},
		}
		defer func() {
			err := redisPool.Close()
			if err != nil {
				log.WithError(err).Error("Couldn't close redis pool connection")
			}
		}()

		// test redis connection
		conn := redisPool.Get()
		_, err = conn.Do("PING")
		if err != nil {
			log.WithError(err).Fatal("Can't establish a connection to the redis server")
			return
		}

		err = conn.Close()
		if err != nil {
			log.WithError(err).Error("Couldn't close redis connection")
		}
	}

	// Create a discord session
	log.Info("Starting discord session...")
	discord, err = discordgo.New(fmt.Sprintf("Bot %v", cfg.DiscordToken))
	if err != nil {
		log.WithError(err).Fatal("Failed to create discord session")
		return
	}

	discord.AddHandler(onReady)
	discord.AddHandler(onMessageCreate)

	err = discord.Open()
	if err != nil {
		log.WithError(err).Fatal("Failed to create discord websocket connection")
		return
	}

	// We're running!
	log.Info("AIRHORNBOT is ready to horn it up.")

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c

	err = discord.Close()
	if err != nil {
		log.WithError(err).Error("Couldn't close discord session")
	}
}
