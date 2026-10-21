package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http" // for ntfy.sh API calls
	"net/url"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/plugins"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/spf13/viper"
)

/*
 * 1. Types
 */

// Vikunja plugin interface, adhering to the contract of [plugins.Plugin] from "code.vikunja.io/api/pkg/plugins"
type NtfyPlugin struct{}

// Custom plugin configuration
type NtfyPluginConfig struct {
	url string
	topic string
	shouldLog bool
}

// Enum-like type modelling the various message priorities of ntfy.sh.
// For more information see https://docs.ntfy.sh/publish/#message-priority
type NtfyPriority string
const (
	NtfyPriorityMax NtfyPriority = "max"
	NtfyPriorityHigh NtfyPriority = "high"
	NtfyPriorityDefault NtfyPriority = "default"
	NtfyPriorityLow NtfyPriority = "low"
	NtfyPriorityMin NtfyPriority = "min"
)

// The structure of a ntfy.sh message
type NtfyMessage struct {
	title string
	content string
	priority NtfyPriority
	tags []string // https://docs.ntfy.sh/emojis/
}

// Event listener for Vikunja's [models.TasksOverdueEvent]
type TasksOverdueListener struct{}

/*
 * 2. Plugin Setup
 */

func (p *NtfyPlugin) Name() string {
	return "ntfy-plugin"
}

func (p *NtfyPlugin) Version() string {
	return "1.1.0"
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
 * 3. Plugin Functionality
 */

func (l *TasksOverdueListener) Name() string { return "TasksOverdueListener" }
func (l *TasksOverdueListener) Handle(msg *message.Message) error {
	var event models.TasksOverdueEvent

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}

	tasks := event.Tasks

	if len(tasks) == 0 {
		log.Debugf("[ntfy-plugin] No overdue tasks. Exiting")
		return nil
	}

	log.Debugf("[ntfy-plugin] Processing %d overdue tasks for u:%d'", len(tasks), event.User.ID)

	// For now (v1.0.0), the ntfy topic is not user-specified. I'll perhaps add support for that in the future
	ntfyMsg := NtfyMessage{
		title: "Overdue Tasks!",
		content: getMessageContent(tasks, time.Now()),
		priority: NtfyPriorityHigh,
		tags: []string{"rotating_light"},
	}

	return sendNtfyMessage(&ntfyMsg)
}

// Contact the ntfy.sh API and try to publish the notification message
func sendNtfyMessage(m *NtfyMessage) error {
	c, err := getConfig()
	if err != nil {
		return err
	}

	ntfyURL, err := constructNtfyUrl(c.url, c.topic)
	if err != nil {
		return err
	}

	log.Debugf("[ntfy-plugin] Sending message to %s", formatSensitiveData(c, ntfyURL))

	req, err := buildRequest(ntfyURL, m)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Debugf("[ntfy-plugin] Got response %q from %s", resp.Status, ntfyURL)
	return nil
}


func buildRequest(URL string, m *NtfyMessage) (*http.Request, error) {
	req, err := http.NewRequest("POST", URL, strings.NewReader(m.content))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Title", m.title)
	req.Header.Set("Priority", string(m.priority))

	if len(m.tags) > 0 {
		req.Header.Set("Tags", strings.Join(m.tags, ","))
	}

	return req, nil
}

// Wrapper around [viper.GetString] that trims leading and trailing whitespace
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

// Joins the URL with the Topic to form a valid ntfy.sh URL.
func constructNtfyUrl(rawURL string, topic string) (string, error) {
	if strings.TrimSpace(topic) == "" {
		return "", errors.New("Empty topic is not allowed")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	scheme := parsedURL.Scheme
	host := parsedURL.Host

	// allow URLs of the form
	// - http://<host>
	// - https://<host>
	if host == "" {
		return "", errors.New("Invalid URL: No host")
	}

	if scheme != "https" && scheme != "http" {
		return "", errors.New("Invalid URL: Only HTTP(S) protocol is supported")
	}

	return parsedURL.JoinPath(topic).String(), nil
}

// Creates the content for the ntfy message.
func getMessageContent(tasks []*models.Task, now time.Time) string {
	taskLen := len(tasks)

	if taskLen == 1 {
		task := tasks[0]
		overdueDays := int(math.Ceil(now.Sub(task.DueDate).Hours() / 24))

		if overdueDays == 1 {
			return fmt.Sprintf("'%s' is 1 day overdue", task.Title)
		}

		return fmt.Sprintf("'%s' is %d days overdue", task.Title, overdueDays)
	}

	// 2026-10-20: Could consider richer descriptions for multiple tasks as well.
	return fmt.Sprintf("You have %d overdue tasks", len(tasks))
}

// Censors the `data` parameter if sensitive data logging is turned off in the configuration
func formatSensitiveData(c *NtfyPluginConfig, data string) string {
	if c.shouldLog {
		return data
	}

	return "<REDACTED>"
}
