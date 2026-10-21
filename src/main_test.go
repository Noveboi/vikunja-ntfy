package main

import (
	"bufio"
	"regexp"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/models"
)

// Tests for the [getMessageContent] function
func TestNtfyMessageContent(t *testing.T) {
	sampleTime := time.Date(2026, time.October, 20, 22, 16, 31, 0, time.UTC)

	t.Run("full task name when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{ Title: "Buy Apples" }}
		want := regexp.MustCompile(`\bBuy Apples\b`)
		got := getMessageContent(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match for %#q, got %q", want, got)
		}
	})

	t.Run("count tasks when more than 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{}, {}}
		want := "You have 2 overdue tasks"
		got := getMessageContent(tasks, sampleTime)

		if want != got {
			t.Errorf("Want %q. Got %q", want, got)
		}
	})

	t.Run("show overdue day when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{DueDate: sampleTime.Add(-1 * time.Hour)}}
		want := regexp.MustCompile(`\b1 day overdue\b`)
		got := getMessageContent(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match %#q, got %q", want, got)
		}
	})

	t.Run("show overdue days when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{DueDate: sampleTime.Add(-28 * time.Hour)}}
		want := regexp.MustCompile(`\b2 days overdue\b`)
		got := getMessageContent(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match %#q, got %q", want, got)
		}
	})
}

// Tests for the [buildNtfyUrl] function
func TestNtfyUrlConstruction(t *testing.T) {
	t.Run("error for empty URL", func (t *testing.T) {
		url, err := buildNtfyUrl("", "topic")

		if err == nil {
			t.Errorf(`Want ("", error), got (%q, nil)`, url)
		}
	})

	t.Run("error for empty topic", func (t *testing.T) {
		url, err := buildNtfyUrl("https://example.com", "")

		if err == nil {
			t.Errorf(`Want ("", error), got (%q, nil)`, url)
		}
	})

	t.Run("merge base HTTP URL with topic", func (t *testing.T) {
		want := "http://example.com/topic";
		got, err := buildNtfyUrl("http://example.com", "topic")

		if err != nil {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}

		if got != want {
			t.Errorf("Want (%q, nil), got (%q, nil)", want, got)
		}
	})

	t.Run("merge base HTTPS URL with topic", func (t *testing.T) {
		want := "https://example.com/topic";
		got, err := buildNtfyUrl("https://example.com", "topic")

		if err != nil {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}

		if got != want {
			t.Errorf("Want (%q, nil), got (%q, nil)", want, got)
		}
	})

	t.Run("valid with IPv4 address", func (t *testing.T) {
		want := "http://192.168.1.77/topic"
		got, err := buildNtfyUrl("http://192.168.1.77", "topic")

		if want != got {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}
	})

	t.Run("valid with IPv6 address", func (t *testing.T) {
		want := "http://[654a:f4ac:70cc:9569:1b30:afe1:db9c:8c5f]/topic"
		got, err := buildNtfyUrl("http://[654a:f4ac:70cc:9569:1b30:afe1:db9c:8c5f]", "topic")

		if want != got {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}
	})

	t.Run("valid with IPv4 address and port", func (t *testing.T) {
		want := "http://192.168.1.77:9876/topic"
		got, err := buildNtfyUrl("http://192.168.1.77:9876", "topic")

		if want != got {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}
	})

	t.Run("valid with IPv6 address and port", func (t *testing.T) {
		want := "http://[654a:f4ac:70cc:9569:1b30:afe1:db9c:8c5f]:9876/topic"
		got, err := buildNtfyUrl("http://[654a:f4ac:70cc:9569:1b30:afe1:db9c:8c5f]:9876", "topic")

		if want != got {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}
	})

	t.Run("valid with path after authority", func (t *testing.T) {
		want := "https://example.com/notifications/topic"
		got, err := buildNtfyUrl("https://example.com/notifications", "topic")

		if want != got {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}
	})

	t.Run("valid with subdomain", func (t *testing.T) {
		want := "https://notifications.example.com/topic"
		got, err := buildNtfyUrl("https://notifications.example.com", "topic")

		if want != got {
			t.Errorf("Want (%q, nil), got (%q, %q)", want, got, err)
		}
	})

	t.Run("error when URL scheme is not HTTP or HTTPS", func (t *testing.T) {
		wantMatch := regexp.MustCompile(`(?i)only http\(s\)`)
		_, err := buildNtfyUrl("ftp://example.com", "topic")

		if !wantMatch.MatchString(err.Error()) {
			t.Errorf("Want error matching %#q, got %q", wantMatch, err)
		}
	})
}

// Tests for the [buildNtfyRequest] function
func TestNtfyRequestConstruction(t *testing.T) {
	URL := "http://example.com/topic"
	message := &NtfyMessage{
		title: "Test!",
		content: "Hello",
		priority: NtfyPriorityDefault,
		tags: []string{"tag1", "tag2", "tag4", "tag3"},
	}

	t.Run("method is POST", func (t *testing.T) {
		req, err := buildNtfyRequest(URL, message)

		if req.Method != "POST" {
			t.Errorf("Want (&{Method:POST,...}, nil), got (%+v, %q)", req, err)
		}
	})

	t.Run("add message content to request body", func (t *testing.T) {
		req, err := buildNtfyRequest(URL, message)
		scanner := bufio.NewScanner(req.Body)
		_ = scanner.Scan()
		body := scanner.Text()

		if body != message.content {
			t.Errorf("Want %q, got (%q, %q)", message.content, body, err)
		}
	})

	t.Run("add title to 'Title' HTTP header", func (t *testing.T) {
		req, err := buildNtfyRequest(URL, message)

		if req.Header.Get("Title") != message.title {
			t.Errorf("Want (${Header:map[Title:[Test!] ...]}, nil), got (%+v, %q)", req, err)
		}
	})

	t.Run("add priority to 'Priority' HTTP header and converts to string", func (t *testing.T) {
		req, err := buildNtfyRequest(URL, message)

		if req.Header.Get("Priority") != string(NtfyPriorityDefault) {
			t.Errorf("Want (${Header:map[Priority:[default] ...]}, nil), got (%+v, %q)", req, err)
		}
	})

	t.Run("add tags to 'Tags' HTTP header separated with commas", func (t *testing.T) {
		req, err := buildNtfyRequest(URL, message)

		if req.Header.Get("Tags") != "tag1,tag2,tag4,tag3" {
			t.Errorf("Want (${Header:map[Tags:[tag1,tag2,tag4,tag3] ...]}, nil), got (%+v, %q)", req, err)
		}
	})
}
