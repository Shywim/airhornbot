package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/sessions"
	"gitlab.com/Shywim/airhornbot/service"
	"golang.org/x/oauth2"
)

const (
	// Permission Constants
	permAdministrator = 8
	permReadMessages  = 1024
	permSendMessages  = 2048
	permConnect       = 1048576
	permSpeak         = 2097152

	// Base URL of the discord API
	apiBaseURL = "https://discordapp.com/api"
)

var (
	store *sessions.CookieStore

	// Oauth2 config for adding bot to a server
	botOAuthConf *oauth2.Config

	// Oauth2 config for managing bot
	manageOAuthConf *oauth2.Config

	userAudioPath *string
)

func InitSessions(cfg service.Cfg) {
	userAudioPath = &cfg.DataPath
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

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "session")
	// TODO: check error

	return session
}

// GetDiscordToken try to get an existing token stored in the session
func getDiscordToken(r *http.Request) string {
	session := getSession(r)
	token := session.Values["token"]
	if token == nil || token == "" {
		return ""
	}

	return token.(string)
}

// GetDiscordSession connects to the API using a session token
func GetDiscordSession(token string) *discordgo.Session {
	discord, err := discordgo.New(fmt.Sprintf("Bearer %v", token))
	if err != nil {
		log.WithError(err).Warning("Error connecting to discord api or has expired")
		return nil
	}

	return discord
}

func IsDiscordAdmin(session *discordgo.Session, guildID string) (bool, error) {
	userGuilds, err := session.UserGuilds(100, "", "")
	if err != nil {
		return false, err
	}

	for _, g := range userGuilds {
		if g.Permissions&permAdministrator != 0 {
			return true, nil
		}
	}
	return false, nil
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
	client := &http.Client{Timeout: 20 * time.Second}
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
