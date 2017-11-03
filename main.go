package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/alphagov/paas-rubbernecker/pkg/pagerduty"
	"github.com/alphagov/paas-rubbernecker/pkg/pivotal"
	"github.com/alphagov/paas-rubbernecker/pkg/rubbernecker"
	"github.com/carlescere/scheduler"
	"github.com/gorilla/mux"
)

var (
	etag    time.Time
	cards   *rubbernecker.Cards
	members *rubbernecker.Members
	support *rubbernecker.SupportRota

	verbose = flag.Bool("verbose", false, "Will enable the DEBUG logging level.")
)

func setupPivotal() (*pivotal.Tracker, error) {
	projectID, err := strconv.Atoi(os.Getenv("PIVOTAL_TRACKER_PROJECT_ID"))
	if err != nil {
		return nil, fmt.Errorf("pivotal project id: %s", err)
	}

	return pivotal.New(projectID, os.Getenv("PIVOTAL_TRACKER_API_TOKEN"))
}

func fetchStories(pt *pivotal.Tracker) error {
	if members == nil {
		return fmt.Errorf("rubbernecker: could not find any members")
	}

	err := pt.FetchCards(rubbernecker.StatusAll)
	if err != nil {
		return err
	}

	c, err := pt.FlattenStories()
	if err != nil {
		return err
	}

	for s, story := range *c {
		if story.Assignees == nil {
			continue
		}

		for i, a := range *story.Assignees {
			(*(*c)[s].Assignees)[i] = (*members)[a.ID]
		}
	}

	if !reflect.DeepEqual(cards, c) {
		cards = c
		etag = time.Now()
	}

	return nil
}

func fetchUsers(pt *pivotal.Tracker) error {
	err := pt.FetchMembers()
	if err != nil {
		return err
	}

	m, err := pt.FlattenMembers()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(members, m) {
		members = m
		etag = time.Now()
	}

	return nil
}

func fetchSupport(pd *pagerduty.Schedule) error {
	err := pd.FetchSupport()
	if err != nil {
		return err
	}

	s, err := pd.FlattenSupport()
	if err != nil {
		return err
	}

	s = formatSupportNames(*s)

	if !reflect.DeepEqual(support, s) {
		support = s
		etag = time.Now()
	}

	return nil
}

func formatSupportNames(s rubbernecker.SupportRota) *rubbernecker.SupportRota {
	return &rubbernecker.SupportRota{
		"in-hours":     s["PaaS team rota - in hours"],
		"out-of-hours": s["PaaS team rota - out of hours"],
		"escalations":  s["PaaS team Escalations - out of hours"],
	}
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := rubbernecker.Response{Message: "OK"}
	resp.JSON(200, w)
}

func stylesHandler(w http.ResponseWriter, r *http.Request) {
	css, err := ioutil.ReadFile("./styles/app.css")
	if err != nil {
		log.Error(err)
	}

	w.Header().Set("Content-Type", "text/css; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(css)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	resp := rubbernecker.Response{}
	et := strconv.FormatInt(etag.Unix(), 10)

	if r.Header.Get("If-None-Match") == et {
		resp.
			JSON(http.StatusNotModified, w)

		return
	}

	resp.
		WithConfig(&rubbernecker.Config{
			ReviewalLimit: 4,
			ApprovalLimit: 5,
		}).
		WithCards(cards, false).
		WithTeamMembers(members).
		WithFreeTeamMembers().
		WithSupport(support)

	if strings.Contains(r.Header.Get("Accept"), "json") {
		w.Header().Set("ETag", et)

		err = resp.JSON(http.StatusOK, w)
	} else {
		err = resp.Template(http.StatusOK, w, "./views/index.html", "./views/card.html")
	}

	if err != nil {
		log.Error(err)
	}
}

func setupLogger() {
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}

	log.SetFormatter(formatter)
}

func init() {
	flag.BoolVar(verbose, "v", false, "Will enable the DEBUG logging level.")
}

func main() {
	flag.Parse()
	setupLogger()

	pd := pagerduty.New(os.Getenv("PAGERDUTY_AUTHTOKEN"))
	pt, err := setupPivotal()
	if err != nil {
		log.Fatal(err)
	}

	scheduler.Every(1).Hours().Run(func() {
		err := fetchUsers(pt)
		if err != nil {
			log.Error(err)
		}

		log.Debug("Team Members have been fetched.")
	})

	scheduler.Every(6).Hours().Run(func() {
		err := fetchSupport(pd)
		if err != nil {
			log.Error(err)
		}

		log.Debug("Support Rota have been fetched.")
	})

	// This is only a procaution as the stories rely on the members to be fetched
	// first. Applying NotImmediately() method to the scheduler will make sure,
	// there isn't a race condition between the two.
	log.Info("Will fetch stories in 20 seconds.")
	scheduler.Every(20).Seconds().NotImmediately().Run(func() {
		err := fetchStories(pt)
		if err != nil {
			log.Error(err)
		}

		log.Debug("Stories have been fetched.")
	})

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/style.css", stylesHandler)
	r.HandleFunc("/health-check", healthcheckHandler)

	http.ListenAndServe(":8080", r)
}
