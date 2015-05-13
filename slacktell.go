package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/artyom/autoflags"
)

func main() {
	config := struct {
		SlackURL string `flag:"url,slack hook endpoint ($SLACK_URL)"`
		Channel  string `flag:"channel,slack channel ($SLACK_CHANNEL)"`
		Name     string `flag:"botname,name of slack sender ($SLACK_BOTNAME)"`
		Message  string `flag:"message,message to send, will be read from stdin if not set"`
	}{
		SlackURL: os.Getenv("SLACK_URL"),
		Channel:  os.Getenv("SLACK_CHANNEL"),
		Name:     os.Getenv("SLACK_BOTNAME"),
	}
	if err := autoflags.Define(&config); err != nil {
		panic(err)
	}
	flag.Parse()
	if config.Channel == "" || config.SlackURL == "" {
		flag.Usage()
		os.Exit(1)
	}
	if config.Message == "" {
		// read message from stdin if not provided as argument
		fi, err := os.Stdin.Stat()
		if err != nil {
			log.Fatal(err)
		}
		mode := fi.Mode()
		if (mode&os.ModeNamedPipe != 0) || mode.IsRegular() {
			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			config.Message = string(data)
		}
	}
	if config.Message == "" {
		log.Fatal("empty message")
	}

	slack := NewSlack(config.SlackURL)
	if err := slack.Push(config.Channel, config.Message, config.Name); err != nil {
		log.Fatal(err)
	}
}

type Slack struct {
	url string // webhook url
}

func NewSlack(hookURL string) *Slack { return &Slack{url: hookURL} }

func (s *Slack) Push(channel, text, name string) error {
	buf := new(bytes.Buffer)
	out := struct {
		Icon    string `json:"icon_emoji,omitempty"`
		Name    string `json:"username,omitempty"`
		Text    string `json:"text"`
		Channel string `json:"channel,omitempty"`
	}{
		Name:    name,
		Text:    text,
		Channel: channel,
	}
	if err := json.NewEncoder(buf).Encode(out); err != nil {
		return err
	}
	resp, err := http.Post(s.url, "application/json", buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		b := new(bytes.Buffer)
		io.Copy(b, resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("invalid status on push: %s\n%s", resp.Status, b.String())
	}
	return nil
}

func init() { log.SetFlags(0) }
