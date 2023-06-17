package main

import (
	//	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	//	"os"
	//	"os/signal"
	"strings"
	//	"syscall"
	"encoding/json"
	"github.com/go-yaml/yaml"
	"github.com/thoj/go-ircevent"
	"net/http"
	"time"
)

type Config struct {
	Server   string `yaml:"server"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Autojoin string `yaml:"autojoin"`

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
	name       string
	rate_limit int // stored in seconds
	mute_until int // unix timestamp seems simplest for this.
	last_seen  int // unix timestamp of last time alert fired.
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
	// TODO: add callbacks here with conn.AddCallback()

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
	// connect to IRC ( Does library handle auth? )
	return b.conn.Connect(b.config.Server)
}

func TimestampToString(timestamp int64) string {
	// takes a unix timestamp and returns a string representation.
	return time.Unix(timestamp, 0).Format(time.UnixDate)
}
func DeltaStringToTimestamp(deltastr string) (int64, error) {
	// parses a string of format [__d][__h][__m][__s] into a time delta and
	// returns the relevant target timestamp, or errors
	// FIXME: time.ParseDuration does not support days, weeks or years.
	delta, err := time.ParseDuration(deltastr)
	if err != nil {
		return -1, err
	}
	target_t := time.Now().Add(delta).Unix()
	return target_t, nil
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

			// TODO: should we check here if an alert name is real / valid ?
			alert_name := fields[2]

			var mute_until int64 // for some reason, math.MaxInt64 isn't 64 bit.
			var err error
			mute_until = math.MaxInt64 // default to muting until forever.

			if len(fields) > 3 { // is there a time?
				mute_until, err = DeltaStringToTimestamp(fields[3])
				if err != nil {
					b.conn.Privmsg(event.Arguments[0], "Invalid time format: "+fields[3])
					return
				}
			}
			b.conn.Privmsg(event.Arguments[0],
				fmt.Sprintf("Muting alert `%s` until %s",
					alert_name,
					TimestampToString(mute_until),
				),
			)
			// TODO: set the mute_until for respective alert.
		case "unmute":
			alert_name := fields[2] // remove mute condition.
			alert := b.alerts[alert_name]
			zero_alert := Alert{}
			if alert != zero_alert {
				alert.mute_until = 0
			}
		case "list": // alerts list
			lst := make([]string, len(b.alerts))
			for k, _ := range b.alerts { // map() ?
				lst = append(lst, k)
			}
			b.conn.Privmsg(event.Arguments[0], strings.Join(lst, " "))
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
	return func(w http.ResponseWriter, r *http.Request) { // handleAlert()
		// callback for http server to handle alerts recieved via post request,
		// from grafana. Takes data, and sends necessary information into a channel
		// for the IRC bot to report.
		var httpAlert HttpAlert
		err := json.NewDecoder(r.Body).Decode(&httpAlert) // unmarshal data into struct for use.
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: send the alert data into a channel for the IRC bot to handle
		for _, innerAlert := range httpAlert.Alerts {
			name := innerAlert.Labels["alertname"]
			if innerAlert.Status == "firing" {
				log.Println("Alert firing: ", name)
				// TODO: build structure, and feed it to a channel for the IRC bot.
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
	//alerts := make([]Alert, 0)
	b.alerts["testAlert"] = Alert{
		name: "testAlert",
	}

	alertsChannel := make(chan InnerAlert)
	// register IRC callbacks
	b.conn.AddCallback("PRIVMSG", func(event *irc.Event) {
		go b.PrivmsgCallback(event)
	})

	b.conn.AddCallback("*", func(event *irc.Event) { // check for/handle alerts channel works
		go func() { // TODO: only need one of these to exist?
			innerAlert := <-alertsChannel
			b.conn.Privmsg(b.config.Autojoin, innerAlert.Labels["alertname"]+" is status : "+innerAlert.Status)
		}()
	})
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
