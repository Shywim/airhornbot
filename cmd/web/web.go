package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/antage/eventsource"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/shywim/airhornbot/service"
	"github.com/shywim/airhornbot/web"
)

const (
	permAdministrator = 8
)

var (
	// Used for storing session information in a cookie
	store *sessions.CookieStore

	// Used for pushing live stat updates to the client
	es eventsource.EventSource
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

func findGuild(guilds []*discordgo.UserGuild, guildID string) *discordgo.UserGuild {
	for _, g := range guilds {
		if g.ID == guildID {
			return g
		}
	}
	return nil
}

func handleDeleteSound(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	/*	guildID := ps.ByName("guildId")
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

	*/

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
	server.HandleMethodNotAllowed = false
	server.GET("/", web.HomeRoute)
	server.GET("/login", web.LoginRoute)
	server.GET("/callback", web.CallbackRoute)
	server.GET("/manage", web.ManageRoute)
	server.GET("/manage/:guildID/sound/:soundID", web.EditSoundRoute)
	server.POST("/manage/:guildID/sound/:soundID", web.EditSoundPostRoute)
	server.GET("/manage/:guildID", web.ManageGuildRoute)
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
