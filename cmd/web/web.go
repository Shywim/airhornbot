package main

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Shywim/airhornbot/service"
	"github.com/Shywim/airhornbot/web"
	log "github.com/Sirupsen/logrus"
	"github.com/antage/eventsource"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
	"github.com/jonas747/dca"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
)

const (
	permAdministrator = 8
)

var (
	// Used for storing session information in a cookie
	store *sessions.CookieStore

	// Used for pushing live stat updates to the client
	es            eventsource.EventSource
	userAudioPath *string
)

type guildInfo struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Plays  int64            `json:"plays"`
	Sounds []*service.Sound `json:"sounds"`
	Icon   string           `json:"icon"`
}

func newCountUpdate() *service.CountUpdate {
	return service.GetStats()
}

func handleEditSound(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	session, _ := store.Get(r, "session")
	guildID := ps.ByName("guildId")
	token := session.Values["token"]

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

	weight, err := strconv.Atoi(r.MultipartForm.Value["weight"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	soundID := ps.ByName("soundID")
	if soundID == "new" {
		soundID = uuid.NewV4().String()
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

		err = saveAudio(dcaData, soundID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sound := service.Sound{
			Name:     r.MultipartForm.Value["name"][0],
			Weight:   weight,
			FilePath: soundID,
		}

		err = service.SaveSound(guildID, &sound, strings.Split(commands, ","))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		sound := service.Sound{
			Name:     r.MultipartForm.Value["name"][0],
			Weight:   weight,
			FilePath: soundID,
		}

		err = service.UpdateSound(guildID, soundID, &sound, strings.Split(commands, ","))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func checkIsGuildAdmin(guildID, token string) (bool, error) {
	discord := web.GetDiscordSession(token)

	guilds, err := discord.UserGuilds(100, "", "")
	if err != nil {
		return false, err
	}

	guild := findGuild(guilds, guildID)
	if guild == nil {
		return false, errors.New("not a user of guild")
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

	err = service.DeleteSound(guildID, soundID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fileServer := http.FileServer(http.Dir("public"))

	// golang use the old "application/x-javascript" by default, we override that
	if strings.HasSuffix(r.URL.String(), ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	}

	// let FileServer handle the end of the response
	fileServer.ServeHTTP(w, r)
}

func server() {
	server := httprouter.New()
	server.GET("/", web.HomeRoute)
	server.GET("/login", web.LoginRoute)
	server.GET("/callback", web.CallbackRoute)
	server.GET("/manage", web.ManageRoute)
	server.GET("/manage/:guildID", web.ManageGuildRoute)
	server.GET("/manage/:guildId/sound/:soundId", web.EditSoundRoute)
	server.POST("/manage/:guildId/sound/:soundId", handleEditSound)
	server.DELETE("/manage/:guildId/:soundId", handleDeleteSound)

	// Only add this route if we have stats to push (e.g. redis connection)
	if es != nil {
		server.Handler("GET", "/events", es)
	}

	server.NotFound = defaultHandler

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
		time.Sleep(time.Second * 5)

		es.SendEventMessage(string(newCountUpdate().ToJSON()), "message", strconv.Itoa(id))
		id++
	}
}

func main() {
	cfg, err := service.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Could not load the configuration file")
	}

	web.LoadTemplates("templates")

	userAudioPath = &cfg.DataPath

	hasRedis := service.InitRedis(cfg)
	if hasRedis {
		defer service.CloseRedis()
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

	web.InitSessions(cfg)

	server()
}
