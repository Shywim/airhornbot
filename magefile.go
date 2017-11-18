// +build mage

package main

import (
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	basePackage = "github.com/Shywim/airhornbot"
	botPackage  = basePackage + "/cmd/bot"
	botBinary   = "airhornbot"
	webPackage  = basePackage + "/cmd/web"
	webBinary   = "airhornweb"
)

var goexe = "go"

func getSrcDir() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	path := filepath.Dir(ex)
	return path, nil
}

func getDep() error {
	return sh.Run(goexe, "get", "-u", "github.com/golang/dep/cmd/dep")
}

// Install go dep and ensure dependencies are installed
func Vendor() error {
	mg.Deps(getDep)
	return sh.Run("dep", "ensure")
}

func BuildAll() {
	mg.Deps(AirhornBot, AirhornWeb)
}

// Build the bot binary
func AirhornBot() error {
	mg.Deps(Vendor)
	return sh.Run(goexe, "build", "-o", botBinary, botPackage)
}

// Build the server binary and the web application
func AirhornWeb() error {
	mg.Deps(Vendor)
	return sh.Run(goexe, "build", "-o", webBinary, webPackage)
}

func WebApp() error {
	webAppPath := os.Getenv("GOPATH") + "/src/" + basePackage + "/web-app/"
	os.Chdir(webAppPath)
	return sh.Run("npm", "install")
}
