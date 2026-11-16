package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http" // for ntfy.sh API calls
	"net/url"
	"strconv"
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
const CENSOR = "<REDACTED>"
const CONF_PREFIX = "plugins.ntfy"

// Vikunja plugin interface, adhering to the contract of [plugins.Plugin] from "code.vikunja.io/api/pkg/plugins"
type NtfyPlugin struct{}

// Custom plugin configuration
type NtfyPluginConfig struct {
	url string
	topic string
	shouldLog bool
	customOverdueTitle string
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
	// The header of the notification
	title string

	// The description of the notification, can be empty
	content string

	// How important the notification is
	priority NtfyPriority

	// One or more of ntfy's tags: https://docs.ntfy.sh/emojis/
	tags []string
}

// Event listener for Vikunja's [models.TasksOverdueEvent]
type TasksOverdueListener struct{}
// Event listener for Vikunja's [models.TaskReminderFiredEvent]
type TaskReminderListener struct{}

type NamedEvent interface {
	Name() string
}

/*
 * 2. Plugin Setup
 */

func (p *NtfyPlugin) Name() string {
	return "ntfy-plugin"
}

func (p *NtfyPlugin) Version() string {
	return "1.3.1"
}

func (p *NtfyPlugin) Init() error {
	config, err := getConfig()
	if err != nil {
		return err
	}

	tryRegisterListener(&models.TasksOverdueEvent{}, &TasksOverdueListener{})
	tryRegisterListener(&models.TaskReminderFiredEvent{}, &TaskReminderListener{})

	if config.shouldLog {
		log.Infof("[ntfy-plugin] Using base URL: %s", config.url)
		log.Infof("[ntfy-plugin] Using topic: %s", config.topic)
	} else {
		log.Infof("[ntfy-plugin] URL/Topic logging is turned off. ntfy URL and/or Topics will be shown as %s", CENSOR)
	}

	if config.customOverdueTitle != "" {
		log.Debugf("[ntfy-plugin] Using custom overdue task notification title: %q", config.customOverdueTitle)
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

// Listener: Reminders
func (l *TaskReminderListener) Name() string { return "TaskReminderListener" }
func (l *TaskReminderListener) Handle(msg *message.Message) error {
	var event models.TaskReminderFiredEvent

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}

	c, err := getConfig()
	if err != nil {
		return err
	}

	log.Debugf("Processing reminder for u:%d", event.User.ID)

	now := time.Now()
	reminderContent := getReminder(event.Task, &now)

	ntfyMsg := NtfyMessage{
		title: "Reminder",
		content: reminderContent,
		priority: NtfyPriorityDefault,
		tags: []string{"bulb"},
	}

	return sendNtfyMessage(&ntfyMsg, c)
}

// Listener: Overdue Tasks
func (l *TasksOverdueListener) Name() string { return "TasksOverdueListener" }
func (l *TasksOverdueListener) Handle(msg *message.Message) error {
	var event models.TasksOverdueEvent

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}

	c, err := getConfig()
	if err != nil {
		return err
	}

	tasks := event.Tasks

	if len(tasks) == 0 {
		log.Debugf("[ntfy-plugin] No overdue tasks. Exiting")
		return nil
	}

	log.Debugf("[ntfy-plugin] Processing %d overdue tasks for u:%d'", len(tasks), event.User.ID)

	ntfyMsg := NtfyMessage{
		title: getCustomOrDefault(c.customOverdueTitle, "Overdue Tasks!"),
		content: getOverdueMessageContent(tasks, time.Now()),
		priority: NtfyPriorityHigh,
		tags: []string{"rotating_light"},
	}

	return sendNtfyMessage(&ntfyMsg, c)
}

func getCustomOrDefault(custom string, def string) string {
	if custom == "" {
		return def
	}

	return custom
}

// Contact the ntfy.sh API and try to publish the notification message
func sendNtfyMessage(m *NtfyMessage, c *NtfyPluginConfig) error {
	ntfyURL, err := buildNtfyUrl(c.url, c.topic)
	if err != nil {
		return err
	}

	req, err := buildNtfyRequest(ntfyURL, m)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Debugf("[ntfy-plugin] Got response %q from %s", resp.Status, formatSensitiveData(c, ntfyURL))
	return nil
}

// Constructs the HTTP POST request for ntfy.sh to publish a notification.
// The URL is considered to contain the base ntfy URL as well as the topic.
func buildNtfyRequest(URL string, m *NtfyMessage) (*http.Request, error) {
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
	joined := CONF_PREFIX + "." + key
	return strings.TrimSpace(viper.GetString(joined))
}

func getBool(key string) bool {
	joined := CONF_PREFIX + "." + key
	return viper.GetBool(joined)
}

// Retrieve the required and optional configuration for this plugin.
// Any missing required configuration will return an error
func getConfig() (*NtfyPluginConfig, error) {
	url := getAndSanitizeString("url")
	topic := getAndSanitizeString("topic")
	customOverdueTitle := getAndSanitizeString("overdue_title")
	shouldLog := getBool("log_sensitive")

	if url == "" {
		return nil, errors.New("ntfy URL is not defined")
	}

	if topic == "" {
		return nil, errors.New("ntfy Topic is not defined")
	}

	return &NtfyPluginConfig{
		url: url,
		topic: topic,
		shouldLog: shouldLog,
		customOverdueTitle: customOverdueTitle,
	}, nil
}

// Joins the URL with the Topic to form a valid ntfy.sh URL.
func buildNtfyUrl(rawURL string, topic string) (string, error) {
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
func getOverdueMessageContent(tasks []*models.Task, now time.Time) string {
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

	return CENSOR
}

// Registers the `listener` for the `event`
func tryRegisterListener(event NamedEvent, listener events.Listener) {
	configName := CONF_PREFIX + "." + event.Name() + ".disable"

	isDisabled := viper.GetBool(configName)

	if isDisabled {
		log.Infof("[ntfy-plugin] Listener %s disabled", listener.Name())
		return
	}

	events.RegisterListener(event.Name(), listener)
	log.Debugf("[ntfy-plugin] Listener %s registered", listener.Name())
}

// Get a verbal and informative description of task to-be-reminded-of.
func getReminder(t *models.Task, now *time.Time) string {
	zeroTime := time.Time{}

	if t.DueDate == zeroTime {
		return t.Title
	}

	dueRemaining := now.Sub(t.DueDate)
	if dueRemaining.Seconds() < 60 {
		return t.Title
	}

	duration := humanize(dueRemaining)

	return "'" + t.Title + "'" + " due in " + duration
}

// Auxiliary function that joins the given `unit` and `noun` into one string.
//
// For formatting, it rounds the `unit` to the nearest integer and naively sets the
// singular/plural of the `noun`.
//
// The "naivity" is due to the function only appending 's' if appropriate, and not
// actually being context-aware. This is fine for its intended use case however!
func roundStr(unit float64, singularNoun string) string {
	rUnit := int(math.Round(unit))

	noun := singularNoun
	if rUnit > 1 {
		noun = noun + "s"
	}

	return strconv.Itoa(rUnit) + " " + noun
}

// Return the provided duration in a human-readable format.
//
// Example: Given the duration `8 * [time.Minute]`, the function will return "8 minutes"
func humanize(d time.Duration) string {
	d = d.Abs()
	mins := d.Minutes()
	hours := d.Hours()

	const oneWeek = 7*24
	const oneMonth = 31*24

	switch {
	case mins >= 1 && hours < 1:
		return roundStr(mins, "minute")
	case hours >= 1 && hours < 24:
		return roundStr(hours, "hour")
	case hours >= 24 && hours < oneWeek:
		return roundStr(hours / 24, "day")
	case hours >= oneWeek && hours < oneMonth:
		return roundStr(hours / oneWeek, "week")
	case hours >= oneMonth:
		return roundStr(hours / oneMonth, "month")
	default:
		return "less than 1 minute"
	}
}
