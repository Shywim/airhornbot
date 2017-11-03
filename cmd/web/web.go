package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/antage/eventsource"
	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
	"github.com/jonas747/dca"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	"github.com/shywim/airhornbot/common"
	"golang.org/x/oauth2"
)

var (
	// Permission Constants
	permAdministrator = 8
	permReadMessages  = 1024
	permSendMessages  = 2048
	permConnect       = 1048576
	permSpeak         = 2097152

	// Redis client
	redisPool *redis.Pool

	// Oauth2 config for adding bot to a server
	botOAuthConf *oauth2.Config

	// Oauth2 config for managing bot
	manageOAuthConf *oauth2.Config

	// Used for storing session information in a cookie
	store *sessions.CookieStore

	// Used for pushing live stat updates to the client
	es eventsource.EventSource

	// Base URL of the discord API
	apiBaseURL = "https://discordapp.com/api"

	userAudioPath *string
)

// Represents a JSON struct of stats that are updated every second and pushed to the client
type countUpdate struct {
	Total          string `json:"total"`
	UniqueUsers    string `json:"unique_users"`
	UniqueGuilds   string `json:"unique_guilds"`
	UniqueChannels string `json:"unique_channels"`
}

type guildInfo struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Plays  int64           `json:"plays"`
	Sounds []*common.Sound `json:"sounds"`
	Icon   string          `json:"icon"`
}

func (c *countUpdate) ToJSON() []byte {
	data, _ := json.Marshal(c)
	return data
}

func newCountUpdate() *countUpdate {
	var (
		total  int64
		users  int64
		guilds int64
		chans  int64
	)

	conn := redisPool.Get()
	defer conn.Close()

	r, err := conn.Do("GET", "airhorn:total")
	if r != nil || err != nil {
		total, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	r, err = conn.Do("SCARD", "airhorn:users")
	if r != nil || err != nil {
		users, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	r, err = conn.Do("SCARD", "airhorn:guilds")
	if r != nil || err != nil {
		guilds, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	r, err = conn.Do("SCARD", "airhorn:channels")
	if r != nil || err != nil {
		chans, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	return &countUpdate{
		Total:          strconv.FormatInt(total, 10),
		UniqueUsers:    strconv.FormatInt(users, 10),
		UniqueGuilds:   strconv.FormatInt(guilds, 10),
		UniqueChannels: strconv.FormatInt(chans, 10),
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Return a random character sequence of n length
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Returns the current session or aborts the request
func getSessionOrAbort(w http.ResponseWriter, r *http.Request) *sessions.Session {
	session, err := store.Get(r, "session")

	if session == nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to get session")
		http.Error(w, "Invalid or corrupted session", http.StatusInternalServerError)
		return nil
	}

	return session
}

func verifyAndOpenSession(w http.ResponseWriter, r *http.Request, s *sessions.Session) bool {
	// Check the state string is correct
	state := r.FormValue("state")
	if state != s.Values["state"] {
		log.WithFields(log.Fields{
			"expected": s.Values["state"],
			"received": state,
		}).Error("Invalid OAuth state")
		http.Redirect(w, r, "/?key_to_success=0", http.StatusTemporaryRedirect)
		return false
	}

	errorMsg := r.FormValue("error")
	if errorMsg != "" {
		log.WithFields(log.Fields{
			"error": errorMsg,
		}).Error("Received OAuth error from provider")
		http.Redirect(w, r, "/?key_to_success=0", http.StatusTemporaryRedirect)
		return false
	}

	token, err := botOAuthConf.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"token": token,
		}).Error("Failed to exchange token with provider")
		http.Redirect(w, r, "/?key_to_success=0", http.StatusTemporaryRedirect)
		return false
	}

	body, _ := json.Marshal(map[interface{}]interface{}{})
	req, err := http.NewRequest("GET", apiBaseURL+"/users/@me", bytes.NewBuffer(body))
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"req":   req,
			"error": err,
		}).Error("Failed to create @me request")
		http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
		return false
	}

	req.Header.Set("Authorization", token.Type()+" "+token.AccessToken)
	client := &http.Client{Timeout: (20 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"client": client,
			"resp":   resp,
		}).Error("Failed to request @me data")
		http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
		return false
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  resp.Body,
		}).Error("Failed to read data from HTTP response")
		http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
		return false
	}

	user := discordgo.User{}
	err = json.Unmarshal(respBody, &user)
	if err != nil {
		log.WithFields(log.Fields{
			"data":  respBody,
			"error": err,
		}).Error("Failed to parse JSON payload from HTTP response")
		http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
		return false
	}

	// Finally write some information to the session store
	s.Values["token"] = token.AccessToken
	s.Values["username"] = user.Username
	s.Values["tag"] = user.Discriminator
	delete(s.Values, "state")
	s.Save(r, w)

	return true
}

// Redirects to the oauth2
func handleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session := getSessionOrAbort(w, r)
	if session == nil {
		return
	}

	// Create a random state
	session.Values["state"] = randSeq(32)
	session.Save(r, w)

	// OR the permissions we want
	perms := permReadMessages | permSendMessages | permConnect | permSpeak

	noBot := r.URL.Query()["nobot"]
	if noBot != nil && noBot[0] == "1" {
		url := manageOAuthConf.AuthCodeURL(session.Values["state"].(string), oauth2.AccessTypeOnline)
		http.Redirect(w, r, url+fmt.Sprintf("&permissions=%v", perms), http.StatusTemporaryRedirect)
		return
	}

	guildID := r.URL.Query()["guild_id"]
	opts := []oauth2.AuthCodeOption{}
	opts = append(opts, oauth2.AccessTypeOnline)
	if guildID != nil {
		guildIDParam := oauth2.SetAuthURLParam("guild_id", guildID[0])
		opts = append(opts, guildIDParam)
	}
	// Return a redirect to the ouath provider
	url := botOAuthConf.AuthCodeURL(session.Values["state"].(string), opts...)
	http.Redirect(w, r, url+fmt.Sprintf("&permissions=%v", perms), http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session := getSessionOrAbort(w, r)
	if session == nil {
		return
	}

	success := verifyAndOpenSession(w, r, session)
	if !success {
		return
	}

	// Store the guild id in redis
	conn := redisPool.Get()
	_, err := conn.Do("SADD", "airhorn:guilds:list", r.FormValue("guild_id"))
	if err != nil {
		log.WithError(err).Error("Failed to save guild in store")
	}

	// And redirect the user back to the dashboard
	http.Redirect(w, r, "/?key_to_success=1", http.StatusTemporaryRedirect)
}

func handleMe(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, _ := store.Get(r, "session")

	body, err := json.Marshal(map[string]interface{}{
		"username": session.Values["username"],
		"tag":      session.Values["tag"],
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func handleManage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, _ := store.Get(r, "session")

	token := session.Values["token"]
	if token == nil || token == "" {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	discord, err := discordgo.New(discordTokenFmt(string(token.(string))))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	guilds, err := discord.UserGuilds(100, "", "")
	if err != nil {
		if strings.HasPrefix(err.Error(), "HTTP 401") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conn := redisPool.Get()
	defer conn.Close()

	var adminGuilds []*guildInfo
	var boringGuilds []*guildInfo
	for _, g := range guilds {
		guild := &guildInfo{
			ID:     g.ID,
			Name:   g.Name,
			Icon:   "https://cdn.discordapp.com/icons/" + g.ID + "/" + g.Icon + ".png",
			Sounds: []*common.Sound{},
		}

		if g.Permissions&permAdministrator != 0 {
			r, err := conn.Do("SISMEMBER", "airhorn:guilds:list", g.ID)
			hasAirhorn, err := redis.Int64(r, err)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if hasAirhorn == 0 {
				boringGuilds = append(boringGuilds, guild)
				continue
			}

			sounds, err := common.GetSoundsByGuild(g.ID)
			guild.Sounds = append(guild.Sounds, sounds...)

			adminGuilds = append(adminGuilds, guild)
		}
	}

	body, err := json.Marshal(map[string]interface{}{
		"airhorn": adminGuilds,
		"boring":  boringGuilds,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func handleNewSound(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	session, _ := store.Get(r, "session")
	guildID := ps.ByName("guildId")
	token := session.Values["token"]
	soundID := uuid.NewV4()

	hasPerm, err := checkIsGuildAdmin(guildID, string(token.(string)))
	if err != nil {
		if strings.HasPrefix(err.Error(), "HTTP 401") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hasPerm == false {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseMultipartForm(0)
	sndFile, sndFileH, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	commands := r.MultipartForm.Value["command"][0]
	if commands == "" {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	// read file size
	var sndFileSize int64
	switch t := sndFile.(type) {
	case *os.File:
		sndFileInfo, err := t.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sndFileSize = sndFileInfo.Size()
		break
	default:
		sndFileSize, err = sndFile.Seek(0, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		break
	}

	// check file > 200kB
	if sndFileSize > 200000 {
		http.Error(w, "File too large", http.StatusNotAcceptable)
		return
	}

	var dcaData io.Reader
	if !strings.HasSuffix(sndFileH.Filename, ".dca") {
		// convert file if (presumably) not a dca file
		dcaSession, err := dca.EncodeMem(sndFile, dca.StdEncodeOptions)
		defer dcaSession.Cleanup()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dcaData = dcaSession
	} else {
		dcaData = sndFile
	}

	err = saveAudio(dcaData, soundID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	weight, err := strconv.Atoi(r.MultipartForm.Value["weight"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sound := common.Sound{
		Name:     r.MultipartForm.Value["name"][0],
		Weight:   weight,
		FilePath: soundID.String(),
	}

	err = common.SaveSound(guildID, &sound, strings.Split(commands, ","))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func discordTokenFmt(token string) string {
	return fmt.Sprintf("Bearer %v", token)
}

func checkIsGuildAdmin(guildID, token string) (bool, error) {
	discord, err := discordgo.New(discordTokenFmt(token))
	if err != nil {
		return false, err
	}

	guilds, err := discord.UserGuilds(100, "", "")
	if err != nil {
		return false, err
	}

	guild := findGuild(guilds, guildID)
	if guild == nil {
		return false, errors.New("Not a user of guild")
	}

	return guild.Permissions&permAdministrator != 0, nil
}

func findGuild(guilds []*discordgo.UserGuild, guildID string) *discordgo.UserGuild {
	for _, g := range guilds {
		if g.ID == guildID {
			return g
		}
	}
	return nil
}

func saveAudio(a io.Reader, n string) error {
	// check user directory exists
	_, err := os.Stat(*userAudioPath)
	if os.IsNotExist(err) {
		os.Mkdir(*userAudioPath, os.ModePerm)
	} else if err != nil {
		return err
	}

	// create file
	out, err := os.Create(filepath.Join(*userAudioPath, n))
	if err != nil {
		return err
	}

	// encore file
	io.Copy(out, a)

	return nil
}

func handleEditSound(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	guildID := ps.ByName("guildId")
	soundID := ps.ByName("soundId")
	session, _ := store.Get(r, "session")
	token := session.Values["token"]

	isAdmin, err := checkIsGuildAdmin(guildID, string(token.(string)))
	if err != nil {
		if strings.HasPrefix(err.Error(), "HTTP 401") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isAdmin == false {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// parse form data
	r.ParseMultipartForm(0)

	commands := r.MultipartForm.Value["command"][0]
	if commands == "" {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	weight, err := strconv.Atoi(r.MultipartForm.Value["weight"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sound := common.Sound{
		ID:     soundID,
		Name:   r.MultipartForm.Value["name"][0],
		Weight: weight,
	}

	err = common.UpdateSound(guildID, soundID, &sound, strings.Split(commands, ","))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteSound(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	guildID := ps.ByName("guildId")
	soundID := ps.ByName("soundId")
	session, _ := store.Get(r, "session")
	token := session.Values["token"]

	isAdmin, err := checkIsGuildAdmin(guildID, string(token.(string)))
	if err != nil {
		if strings.HasPrefix(err.Error(), "HTTP 401") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isAdmin == false {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	err = common.DeleteSound(guildID, soundID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fileServer := http.FileServer(http.Dir("web-app/public"))

	// golang use the old "application/x-javascript" by default, we override that
	if strings.HasSuffix(r.URL.String(), ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	}

	// let FileServer handle the end of the response
	fileServer.ServeHTTP(w, r)
}

func server() {
	server := httprouter.New()
	server.GET("/me", handleMe)
	server.GET("/login", handleLogin)
	server.GET("/callback", handleCallback)
	server.GET("/me/guilds", handleManage)
	server.POST("/manage/:guildId/new", handleNewSound)
	server.PUT("/manage/:guildId/:soundId", handleEditSound)
	server.DELETE("/manage/:guildId/:soundId", handleDeleteSound)
	server.NotFound = defaultHandler

	// Only add this route if we have stats to push (e.g. redis connection)
	if es != nil {
		server.Handler("GET", "/events", es)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "14000"
	}

	log.WithFields(log.Fields{
		"port": port,
	}).Info("Starting HTTP Server")

	// If the requests log doesnt exist, make it
	if _, err := os.Stat("requests.log"); os.IsNotExist(err) {
		ioutil.WriteFile("requests.log", []byte{}, 0600)
	}

	// Open the log file in append mode
	logFile, err := os.OpenFile("requests.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to open requests log file")
		return
	}
	defer logFile.Close()

	// Actually start the server
	loggedRouter := handlers.LoggingHandler(logFile, server)
	http.ListenAndServe(":"+port, loggedRouter)
}

func broadcastLoop() {
	var id int
	for {
		time.Sleep(time.Second * 1)

		es.SendEventMessage(string(newCountUpdate().ToJSON()), "message", strconv.Itoa(id))
		id++
	}
}

func connectToRedis(connStr string) (err error) {
	log.WithFields(log.Fields{
		"host": connStr,
	}).Info("Connecting to redis")

	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", connStr)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	// test redis connection
	conn := redisPool.Get()
	_, err = conn.Do("PING")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Can't establish a connection to the redis server")
		return err
	}
	conn.Close()

	return nil
}

func main() {
	cfg := common.LoadConfig()

	userAudioPath = &cfg.DataPath

	if cfg.RedisHost != "" {
		// First, open a redis connection we use for stats
		if connectToRedis(cfg.RedisHost) != nil {
			return
		}
		defer redisPool.Close()

		// Now start the eventsource loop for client-side stat update
		es = eventsource.New(nil, func(req *http.Request) [][]byte {
			return [][]byte{
				[]byte("X-Accel-Buffering: no"),
				[]byte("Access-Control-Allow-Origin: *"),
			}
		},
		)

		defer es.Close()
		go broadcastLoop()
	}

	// Create a cookie store
	store = sessions.NewCookieStore([]byte(cfg.DiscordClientSecret))

	// Setup the OAuth2 Configuration
	endpoint := oauth2.Endpoint{
		AuthURL:  apiBaseURL + "/oauth2/authorize",
		TokenURL: apiBaseURL + "/oauth2/token",
	}

	botOAuthConf = &oauth2.Config{
		ClientID:     cfg.DiscordClientID,
		ClientSecret: cfg.DiscordClientSecret,
		Scopes:       []string{"bot", "identify", "guilds"},
		Endpoint:     endpoint,
		RedirectURL:  "http://airhorn.shywim.fr/callback",
	}

	manageOAuthConf = &oauth2.Config{
		ClientID:     cfg.DiscordClientID,
		ClientSecret: cfg.DiscordClientSecret,
		Scopes:       []string{"identify", "guilds"},
		Endpoint:     endpoint,
		RedirectURL:  "http://airhorn.shywim.fr/callback",
	}

	server()
}
