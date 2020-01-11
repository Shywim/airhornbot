package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jonas747/dca"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	"github.com/shywim/airhornbot/service"
	"golang.org/x/oauth2"
)

// HomeRoute serves home.gohtml
func HomeRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tmplCtx := getContext(r)
	tmplData := TemplateData{
		Context: tmplCtx,
	}
	renderTemplate(w, "home.gohtml", tmplData)
}

// LoginRoute handles login to Discord
func LoginRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session := getSession(r)
	if session == nil {
		// TODO: error
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
	var opts []oauth2.AuthCodeOption
	opts = append(opts, oauth2.AccessTypeOnline)
	if guildID != nil {
		guildIDParam := oauth2.SetAuthURLParam("guild_id", guildID[0])
		opts = append(opts, guildIDParam)
	}
	// Return a redirect to the oauth provider
	url := botOAuthConf.AuthCodeURL(session.Values["state"].(string), opts...)
	http.Redirect(w, r, url+fmt.Sprintf("&permissions=%v", perms), http.StatusTemporaryRedirect)
}

// CallbackRoute handles return from Discord login
func CallbackRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session := getSession(r)
	if session == nil {
		// TODO: error
		return
	}

	success := verifyAndOpenSession(w, r, session)
	if !success {
		return
	}

	err := service.AddGuild(r.FormValue("guild_id"))
	if err != nil {
		log.WithError(err).Error("Failed to save guild in store")
	}

	// And redirect the user back to the dashboard
	http.Redirect(w, r, "/manage", http.StatusTemporaryRedirect)
}

// AskLoginRoute serves login.gohtml
func AskLoginRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tmplCtx := getContext(r)
	tmplData := TemplateData{
		Context: tmplCtx,
	}
	renderTemplate(w, "login.gohtml", tmplData)
}

// ManageRoute serves manage.gohtml
func ManageRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	token := getDiscordToken(r)
	if token == "" {
		AskLoginRoute(w, r, nil)
		return
	}

	session := GetDiscordSession(token)
	if session == nil {
		AskLoginRoute(w, r, nil)
		return
	}
	defer session.Close()

	userGuilds, err := service.GetGuildsWithSounds(session)
	if err != nil {
		// TODO: error
		log.WithError(err).Error("Error retrieving user's guilds")
		return
	}

	tmplCtx := getContext(r)
	tmplData := TemplateData{
		Context: tmplCtx,
		Data:    userGuilds,
	}
	renderTemplate(w, "manage.gohtml", tmplData)
}

// ManageGuildRoute serves manage.gohtml
func ManageGuildRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	token := getDiscordToken(r)
	if token == "" {
		AskLoginRoute(w, r, nil)
		return
	}

	session := GetDiscordSession(token)
	if session == nil {
		AskLoginRoute(w, r, nil)
		return
	}
	defer session.Close()

	guild, err := service.GetGuildWithSounds(session, ps.ByName("guildID"))
	if err != nil {
		// TODO: error
		log.WithFields(log.Fields{
			"error":   err,
			"guildID": ps.ByName("guildID"),
		}).Error("Error retrieving user's guild")
		return
	}

	tmplCtx := getContext(r)
	tmplData := TemplateData{
		Context: tmplCtx,
		Data:    guild,
	}
	renderTemplate(w, "guild.gohtml", tmplData)
}

func EditSoundRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var sound *service.Sound
	soundID := ps.ByName("soundID")
	guildID := ps.ByName("guildID")

	token := getDiscordToken(r)
	if token == "" {
		AskLoginRoute(w, r, nil)
		return
	}

	session := GetDiscordSession(token)
	if session == nil {
		AskLoginRoute(w, r, nil)
		return
	}
	defer session.Close()

	isAdmin, err := IsDiscordAdmin(session, guildID)
	if err != nil || !isAdmin {
		// TODO: error
		AskLoginRoute(w, r, nil)
		return
	}

	if strings.Compare(soundID, "new") == 0 {
		sound = &service.Sound{
			ID:      "new",
			GuildID: guildID,
		}
	} else {
		sound, err = service.GetSound(soundID)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"guildID": ps.ByName("guildID"),
			}).Error("Error retrieving sound")
			// TODO: error
			return
		}
		sound.GuildID = guildID
	}

	tmplCtx := getContext(r)
	tmplData := TemplateData{
		Context: tmplCtx,
		Data:    sound,
	}
	renderTemplate(w, "sound.gohtml", tmplData)
}

func EditSoundPostRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	guildID := ps.ByName("guildID")
	token := getDiscordToken(r)
	session := GetDiscordSession(token)
	if session == nil {
		AskLoginRoute(w, r, nil)
		return
	}
	defer session.Close()

	hasPerm, err := IsDiscordAdmin(session, guildID)
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

	/*weight, err := strconv.Atoi(r.MultipartForm.Value["weight"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}*/

	commandsString := r.MultipartForm.Value["commands"]
	if len(commandsString) == 0 {
		http.Error(w, "At least one command is required", http.StatusNotAcceptable)
		return
	}
	commands := []service.Command{}
	for _, c := range commandsString {
		commands = append(commands, service.Command{Command: c, Weight: 1})
	}

	soundID := ps.ByName("soundID")
	if soundID == "new" {
		sndFile, sndFileH, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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

		err = service.SaveAudio(dcaData, soundID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sound := service.Sound{
			Name:     r.MultipartForm.Value["name"][0],
			FilePath: soundID,
			GuildID:  guildID,
			Commands: commands,
		}

		err = sound.Save()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		sound := service.Sound{
			ID:       soundID,
			Name:     r.MultipartForm.Value["name"][0],
			FilePath: soundID,
			GuildID:  guildID,
			Commands: commands,
		}

		err = sound.Save()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	ManageGuildRoute(w, r, ps)
}
