package web

import (
	"fmt"
	"net/http"

	"github.com/Shywim/airhornbot/service"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
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
	// Return a redirect to the ouath provider
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
	http.Redirect(w, r, "/?key_to_success=1", http.StatusTemporaryRedirect)
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
