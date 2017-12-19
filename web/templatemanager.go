package web

import (
	"html/template"
	"net/http"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/oxtoacart/bpool"
	"gitlab.com/Shywim/airhornbot/service"
)

var (
	templates map[string]*template.Template
	bufpool   *bpool.BufferPool
)

func init() {
	bufpool = bpool.NewBufferPool(64)
	log.Println("buffer allocation successful")
}

// TemplateContext data is used in every templates
type TemplateContext struct {
	NoRedis      bool
	SiteURL      string
	StatsCounter service.CountUpdate
}

func getContext(r *http.Request) TemplateContext {
	return TemplateContext{
		SiteURL:      "//" + r.Host,
		StatsCounter: *service.GetStats(),
	}
}

// TemplateData is used to store data to use in a template
type TemplateData struct {
	Context TemplateContext
	Data    interface{}
}

// LoadTemplates from the specified templatesPath
func LoadTemplates(templatesPath string) {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	layoutFiles, err := filepath.Glob(templatesPath + "/*.gohtml")
	if err != nil {
		log.WithError(err).Fatal("Couldn't load templates")
		return
	}

	partialFiles, err := filepath.Glob(templatesPath + "/partials/*.gohtml")
	if err != nil {
		log.WithError(err).Fatal("Couldn't load partials templates")
		return
	}

	for _, layout := range layoutFiles {
		filename := filepath.Base(layout)
		files := append(partialFiles, layout)
		templates[filename] = template.Must(template.ParseFiles(layout))
		templates[filename] = template.Must(templates[filename].ParseFiles(files...))
	}
}

func renderTemplate(w http.ResponseWriter, name string, data TemplateData) {
	tmpl, ok := templates[name]
	if !ok {
		log.WithFields(log.Fields{
			"name": name,
		}).Warn("The template does not exists")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	buf := bufpool.Get()
	defer bufpool.Put(buf)

	err := tmpl.Execute(buf, data)
	if err != nil {
		log.WithError(err).Error("Could not render the template")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}
