package main

import (
	//	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	//	"math"
	//	"os"
	//	"os/signal"
	"strings"
	//	"syscall"
	"encoding/json"
	"github.com/go-yaml/yaml"
	"github.com/thoj/go-ircevent"
	"net/http"
	//	"strconv"
	"time"
)

type Config struct {
	Server        string `yaml:"server"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	Autojoin      string `yaml:"autojoin"`
	AlertsChannel string `yaml:"alertsChannel"`

	DatabaseURL string `yaml:"DatabaseURL"`
	DBusername  string `yaml:"DBusername"`
	DBpassword  string `yaml:"DBpassword"`

	WebAddress string `yaml:"webAddress"`
}

type Bot struct {
	conn   *irc.Connection
	config *Config
	alerts map[string]Alert
	//	db        *sql.DB
	startTime time.Time // needed for alert timedeltas.
}

type Alert struct {
	Name          string
	Rate_limit    time.Duration
	Mute_until    time.Time
	Last_seen     time.Time
	Last_reported time.Time
}

////////////////////////////////////////
// IRC stuff
func NewBot(config *Config) (*Bot, error) {
	// build a new IRC bot object(from go-ircevent), given a config
	conn := irc.IRC(config.Username, config.Username)
	//	conn.VerboseCallbackHandler = true
	//	conn.Debug = true
	conn.Password = config.Password
	conn.SASLLogin = config.Username
	conn.SASLPassword = config.Password

	bot := &Bot{
		conn:      conn,
		config:    config, // is this config the right format?
		alerts:    make(map[string]Alert),
		startTime: time.Now(),
	}
	return bot, nil
}

// NOTE: func [class] functionName([arguments]) ([returns])
func (b *Bot) Connect() error {
	return b.conn.Connect(b.config.Server)
}

func (b *Bot) PrivmsgCallback(event *irc.Event) {
	if event.Nick != "DaPinkOne" {
		// TODO: auth function.
		return
	}
	// if user is auth'd:
	fmt.Printf("Command recieved from %s on channel %s for: %s",
		event.Nick,
		event.Arguments[0],
		event.Arguments[1],
	)
	fields := strings.Fields(event.Arguments[1])
	b.conn.Privmsg(event.Arguments[0], "Acknowledged: "+fields[0])
	switch fields[0] { // switch on command.
	case "quit":
		b.conn.Quit()
	case "join":
		if len(fields) >= 1 {
			b.conn.Join(fields[1])
		}
	case "part":
		if len(fields) >= 1 { // error message?
			b.conn.Part(fields[1])
			return
		}
		// default to parting current channel, if one isn't given.
		b.conn.Part(event.Arguments[0])
	case "alert": // list alerts
		switch fields[1] { // list commands
		case "mute":
			// `alert mute xyz [43d23h8m]` NOTE: does not currently support days.
			// when told to mute an alert, mute for a period
			// if period not given, default to max unix timestamp.

			alert_name := fields[2]

			var mute_delta time.Duration // TODO: default value for mute duration?
			var err error

			if len(fields) > 3 {
				mute_delta, err = time.ParseDuration(fields[3])
				if err != nil {
					b.conn.Privmsg(event.Arguments[0], "Invalid time format: "+fields[3])
					return
				}
			}
			mute_until := time.Now().Add(mute_delta)
			b.conn.Privmsg(event.Arguments[0],
				fmt.Sprintf("Muting alert `%s` until %s",
					alert_name,
					mute_until.String(),
				),
			)

			val := b.alerts[alert_name]
			b.alerts[alert_name] = Alert{
				Name:       alert_name,
				Mute_until: mute_until,
				Last_seen:  val.Last_seen,
				Rate_limit: val.Rate_limit,
			}
		case "rate": // set alert rate limit
			alert_name := fields[2]

			var rate_limit time.Duration
			var err error
			if len(fields) > 3 {
				rate_limit, err = time.ParseDuration(fields[3])
				if err != nil {
					b.conn.Privmsg(event.Arguments[0], "Invalid time format: "+fields[3])
					return
				}
			} else {
				b.conn.Privmsg(event.Arguments[0], "Time delta required for rate limit. ")
				return
			}

			log.Printf("%s limited to %d\n", alert_name, rate_limit.String())

			val := b.alerts[alert_name]
			b.alerts[alert_name] = Alert{
				Name:          alert_name,
				Mute_until:    val.Mute_until,
				Last_seen:     val.Last_seen,
				Rate_limit:    rate_limit,
				Last_reported: val.Last_reported,
			}
		case "unmute": // unmute by setting mute_until to current time.
			alert_name := fields[2]
			val := b.alerts[alert_name]
			b.alerts[alert_name] = Alert{
				Name:       alert_name,
				Mute_until: time.Now(),
				Last_seen:  val.Last_seen,
				Rate_limit: val.Rate_limit,
			}
		case "list": // alerts list
			lst := make([]string, len(b.alerts))
			for k, _ := range b.alerts { // map() ?
				lst = append(lst, k)
			}
			b.conn.Privmsg(event.Arguments[0], "Alerts:"+strings.Join(lst, " "))
		case "info": // get information about an alert record.
			alert_name := fields[2]
			record, ok := b.alerts[alert_name]
			var msg string
			if ok {
				msg = fmt.Sprintf("`%s` : mute %s : seen %s : rate %s : reported %s",
					alert_name,
					record.Mute_until.String(),
					record.Last_seen.String(),
					record.Rate_limit.String(),
					record.Last_reported.String(),
				)
			} else {
				msg = fmt.Sprintf("Alert `%s` has no record.", alert_name)
			}
			b.conn.Privmsg(event.Arguments[0], msg)

		default:
			// default case? reply w/ error message?
		}
	}
}

// End of IRC stuff
///////////////////////////

//////////////////////////
// web stuff

type HttpAlert struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []InnerAlert      `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	OrgID             int               `json:"orgId"`
	Title             string            `json:"title"`
	State             string            `json:"state"`
	Message           string            `json:"message"`
}

type InnerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
	SilenceURL   string            `json:"silenceURL"`
	DashboardURL string            `json:"dashboardURL"`
	PanelURL     string            `json:"panelURL"`
	Values       interface{}       `json:"values"` // TODO: what type is "Values" ?
	ValueString  string            `json:"valueString"`
}

func buildAlertHandler(alertsChannel chan<- InnerAlert) func(w http.ResponseWriter, r *http.Request) {
	// takes a channel, and builds a callback which knows about that channel,
	// so the callback can send alert POST data recieved to the channel.
	return func(w http.ResponseWriter, r *http.Request) {
		// callback for http server to handle alerts recieved via post request,
		// from grafana. Takes data, and sends necessary information into a channel
		// for the IRC bot to report.
		var httpAlert HttpAlert
		err := json.NewDecoder(r.Body).Decode(&httpAlert) // unmarshal data into struct for use.
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// for each alert in the list given in the POST, if it's firing, send it to the monitor.
		for _, innerAlert := range httpAlert.Alerts {
			name := innerAlert.Labels["alertname"]
			if innerAlert.Status == "firing" {
				log.Println("Alert firing: ", name)
				alertsChannel <- innerAlert
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Alert recieved."))
	}
}

// End web stuff
/////////////////////////////////

func main() {
	// read yaml config file

	yamlData, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}

	config := Config{}
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}

	// build new bot
	b, err := NewBot(&config)
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}
	err = b.Connect() // connect to IRC server.
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}
	// TODO: get necessary alert data from database here.
	// b.alerts["testAlert"] = Alert{
	// 	Name: "testAlert",
	// }

	alertsChannel := make(chan InnerAlert)
	// register IRC callbacks
	b.conn.AddCallback("PRIVMSG", func(event *irc.Event) {
		go b.PrivmsgCallback(event)
	})

	go func() { // set up a monitor to feed data from the channel to IRC.
		// monitor will handle rate limiting, muting alerts, etc.
		for {
			innerAlert := <-alertsChannel
			now := time.Now()

			alertName := innerAlert.Labels["alertname"]
			record := b.alerts[alertName]
			log.Println(record)

			// if the innerAlert is not muted, and is not within the rate limit, report.
			report := true
			if now.Before(record.Mute_until) || now.Before(record.Last_reported.Add(record.Rate_limit)) {
				// TODO: need not only last_seen, but also Last_Reported for use  in rate limit.
				log.Println("Alert muted/limited, ", alertName)
				report = false
			}

			// if ok { // initialize the record if necessary
			// 	record.Last_seen = now
			// } else {
			//			}
			last_reported := record.Last_reported
			if report {
				last_reported = now
				b.conn.Privmsg(b.config.AlertsChannel, alertName+" is "+innerAlert.Status)
			}
			b.alerts[alertName] = Alert{
				Name:          alertName,
				Last_seen:     now,
				Rate_limit:    record.Rate_limit,
				Mute_until:    record.Mute_until,
				Last_reported: last_reported,
			}

		}
	}()

	fmt.Printf("Connected to IRCServer: %s\n", config.Server)
	fmt.Printf("Connected to IRC w/ Username: %s\n", config.Username)
	b.conn.Join(b.config.Autojoin)

	// register `handleAlert` as a callback with the http server.
	http.HandleFunc("/alerts", buildAlertHandler(alertsChannel))
	webAddr := b.config.WebAddress
	go func() { // start web server loop.
		log.Fatal(http.ListenAndServe(webAddr, nil))
	}()
	log.Println("Web Server started on", webAddr)

	// start IRC bot main loop
	b.conn.Loop()
}
