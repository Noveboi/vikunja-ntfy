package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http" // for ntfy.sh API calls
	"net/url"
	"strings"

	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/plugins"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/spf13/viper"
)

type NtfyPlugin struct{}

func (p *NtfyPlugin) Name() string {
	return "ntfy-plugin"
}

func (p *NtfyPlugin) Version() string {
	return "1.0.0"
}

func (p *NtfyPlugin) Init() error {
	events.RegisterListener((&models.TasksOverdueEvent{}).Name(), &TasksOverdueListener{})

	config, err := getConfig()
	if err != nil {
		return err
	}

	if config.shouldLog {
		log.Infof("[ntfy-plugin] Using base URL: %s", config.url)
		log.Infof("[ntfy-plugin] Using topic: %s", config.topic)
	} else {
		log.Infof("[ntfy-plugin] URL/Topic logging is turned off. ntfy URL and/or Topics will be shown as <REDACTED>")
	}

	return nil
}

func (p *NtfyPlugin) Shutdown() error {
	return nil
}

var singleton = &NtfyPlugin{}

func NewPlugin() plugins.Plugin { return singleton }

/*
 * Events!
 */
type TasksOverdueListener struct{}

func (l *TasksOverdueListener) Name() string { return "TasksOverdueListener" }
func (l *TasksOverdueListener) Handle(msg *message.Message) error {
	var event models.TasksOverdueEvent

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}

	log.Debugf("[ntfy-plugin] Received event: %v", event)

	tasks := event.Tasks

	log.Debugf("[ntfy-plugin] Processing %d overdue tasks for %d'", len(tasks), event.User.ID)

	if len(tasks) == 0 {
		return nil
	}

	// For now (v1.0.0), the ntfy topic is not user-specified. I'll perhaps add support for that in the future
	ntfyMsg := NtfyMessage{
		title: "Overdue Tasks!",
		message: fmt.Sprintf("%s has %d overdue tasks", event.User.Username, len(tasks)),
		priority: NtfyPriorityHigh,
		tags: []string{"rotating_light"},
	}

	return sendNtfyMessage(&ntfyMsg)
}

func sendNtfyMessage(m *NtfyMessage) error {
	config, err := getConfig()
	if err != nil {
		return err
	}

	// Parse the raw URL string into a URL struct to validate the structure and ensure it is a proper URL
	ntfyUrl, err := url.Parse(config.url)
	if err != nil {
		return err
	}

	if ntfyUrl.Host == "" || (ntfyUrl.Scheme != "http" && ntfyUrl.Scheme != "https") {
		return errors.New("ntfy URL must be an absolute HTTP/HTTPS URL")
	}

	ntfyUrl = ntfyUrl.JoinPath(config.topic)

	ntfyUrlLog := "<REDACTED>"
	if config.shouldLog {
		ntfyUrlLog = ntfyUrl.String()
	}

	log.Debugf("[ntfy-plugin] Sending message to %s", ntfyUrlLog)

	reqBody := strings.NewReader(m.message)
	req, err := http.NewRequest("POST", ntfyUrl.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Title", m.title)
	req.Header.Set("Priority", string(m.priority))
	if len(m.tags) > 0 {
		req.Header.Set("Tags", strings.Join(m.tags, ","))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Debugf("[ntfy-plugin] Got response %s from %s", resp.Status, ntfyUrlLog)
	return nil
}

func getAndSanitizeString(key string) string {
	return strings.TrimSpace(viper.GetString(key))
}

// Retrieve the required and optional configuration for this plugin.
// Any missing required configuration will trigger an error
func getConfig() (*NtfyPluginConfig, error) {
	url := getAndSanitizeString("plugins.ntfy.url")
	topic := getAndSanitizeString("plugins.ntfy.topic")
	shouldLog := viper.GetBool("plugins.ntfy.logUrlAndTopic")

	if url == "" {
		return nil, errors.New("ntfy URL is not defined")
	}

	if topic == "" {
		return nil, errors.New("ntfy Topic is not defined")
	}

	return &NtfyPluginConfig{url, topic, shouldLog}, nil
}

type NtfyPluginConfig struct {
	url string
	topic string
	shouldLog bool
}

type NtfyPriority string
// https://docs.ntfy.sh/publish/#message-priority
const (
	NtfyPriorityMax NtfyPriority = "max"
	NtfyPriorityHigh NtfyPriority = "high"
	NtfyPriorityDefault NtfyPriority = "default"
	NtfyPriorityLow NtfyPriority = "low"
	NtfyPriorityMin NtfyPriority = "min"
)

type NtfyMessage struct {
	title string
	message string
	priority NtfyPriority
	tags []string // https://docs.ntfy.sh/emojis/
}
