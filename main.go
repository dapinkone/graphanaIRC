package main

import (
	//	"database/sql"
	"fmt"
	"io/ioutil"
	//	"log"
	//	"os"
	//	"os/signal"
	"strings"
	//	"syscall"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/thoj/go-ircevent"
)

type Config struct {
	Server   string `yaml:"server"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	DatabaseURL string `yaml:"DatabaseURL"`
	DBusername  string `yaml:"DBusername"`
	DBpassword  string `yaml:"DBpassword"`
}

type Bot struct {
	conn   *irc.Connection
	config *Config
	//	alerts    map[string]Alert
	//	db        *sql.DB
	startTime time.Time // needed for alert timedeltas.
}

type Alert struct {
	name        string
	rate_limit  int // stored in seconds
	blacklisted bool
	muted_until int // unix timestamp seems simplest for this.
}

func NewBot(config *Config) (*Bot, error) {
	// build a new IRC bot object(from go-ircevent), given a config
	conn := irc.IRC(config.Username, config.Username)
	conn.VerboseCallbackHandler = true
	conn.Debug = true
	conn.Password = config.Password
	conn.SASLLogin = config.Username
	conn.SASLPassword = config.Password
	// TODO: add callbacks here with conn.AddCallback()

	bot := &Bot{
		conn:   conn,
		config: config, // is this config the right format?
		//	alerts:    make(map[string]Alert),
		startTime: time.Now(),
	}
	return bot, nil
}

// NOTE: func [class] functionName([arguments]) ([returns])
func (b *Bot) Connect() error {
	// connect to IRC ( Does library handle auth? )
	return b.conn.Connect(b.config.Server)
}
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

	// build new bot.
	b, err := NewBot(&config)
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}
	err = b.Connect()
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}
	alerts := make([]Alert, 0)
	alerts = append(alerts,
		Alert{
			name:        "testAlert",
			rate_limit:  60,
			blacklisted: false,
			muted_until: 0,
		},
	)

	// register callbacks
	b.conn.AddCallback("PRIVMSG", func(event *irc.Event) { // first class functions?
		go func(event *irc.Event) {
			if event.Nick == "DaPinkOne" { // TODO: auth function.
				fmt.Println("Command recieved from %s on channel %s for: %s",
					event.Nick,
					event.Arguments[0],
					event.Arguments[1],
				)
				fields := strings.Fields(event.Arguments[1])
				b.conn.Privmsg(event.Nick, "Acknowledged "+fields[0])
				switch fields[0] {
				case "quit":
					b.conn.Quit()
				case "list":
					{
						lst := make([]string, len(alerts))
						for i, _ := range alerts { // map() ?
							lst[i] = alerts[i].name
						}
						b.conn.Privmsg(event.Nick, strings.Join(lst, " "))
					}
				}
			}
		}(event)
	})
	fmt.Printf("Connected to IRCServer: %s\n", config.Server)
	fmt.Printf("Connected to IRC w/ Username: %s\n", config.Username)
	b.conn.Loop()
}
