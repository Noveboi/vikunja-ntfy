package main

import (
	"bufio"
	"regexp"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/models"
)

// Tests for the [getMessageContent] function
func TestOverdueNtfyMessageContent(t *testing.T) {
	sampleTime := time.Date(2026, time.October, 20, 22, 16, 31, 0, time.UTC)

	t.Run("full task name when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{ Title: "Buy Apples" }}
		want := regexp.MustCompile(`\bBuy Apples\b`)
		got := getOverdueMessageContent(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match for %#q, got %q", want, got)
		}
	})

	t.Run("count tasks when more than 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{}, {}}
		want := "You have 2 overdue tasks"
		got := getOverdueMessageContent(tasks, sampleTime)

		if want != got {
			t.Errorf("Want %q. Got %q", want, got)
		}
	})

	t.Run("show overdue day when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{DueDate: sampleTime.Add(-1 * time.Hour)}}
		want := regexp.MustCompile(`\b1 day overdue\b`)
		got := getOverdueMessageContent(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match %#q, got %q", want, got)
		}
	})

	t.Run("show overdue days when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{DueDate: sampleTime.Add(-28 * time.Hour)}}
		want := regexp.MustCompile(`\b2 days overdue\b`)
		got := getOverdueMessageContent(tasks, sampleTime)

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

// Tests for the [humanize] function
func TestDateHumanization(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	cases := []struct{
		duration time.Duration
		want string
	}{
		{1 * time.Minute, "1 minute"},
		{1 * time.Minute, "1 minute"},
		{5 * time.Minute, "5 minutes"},
		{59 * time.Minute, "59 minutes"},
		{1 * time.Hour, "1 hour"},
		{60 * time.Minute, "1 hour"},
		{5 * time.Hour, "5 hours"},
		{23 * time.Hour, "23 hours"},
		{24 * time.Hour, "1 day"},
		{2 * 24 * time.Hour, "2 days"},
		{7 * 24 * time.Hour, "1 week"},
		{31 * 24 * time.Hour, "1 month"},
		{2 * 31 * 24 * time.Hour, "2 months"},
		{18 * 31 * 24 * time.Hour, "18 months"}, // don't use 'year'

		// rounding
		{89 * time.Second, "1 minute"}, // <1.5 minutes
		{90 * time.Second, "2 minutes"}, // =1.5 minutes
		{91 * time.Second, "2 minutes"}, // >1.5 minutes

		// negative
		{-1 * time.Minute, "1 minute"},
		{-10 * time.Minute, "10 minutes"},
		{-1 * time.Hour, "1 hour"},
		{-10 * time.Hour, "10 hours"},
		{-24 * time.Hour, "1 day"},
		{5 * -24 * time.Hour, "5 days"},

		// seconds
		{20 * time.Second, "less than 1 minute"},
		{59 * time.Second, "less than 1 minute"},

		// calculated durations
		{now.Sub(now.Add(-1 * time.Minute)), "1 minute"},
		{now.Sub(now.Add(-10 * time.Minute)), "10 minutes"},
		{now.Sub(now.Add(-4 * time.Hour)), "4 hours"},
		{now.Sub(now.Add(4 * time.Hour)), "4 hours"},
		{now.Sub(now.Add(-3 * 24 * time.Hour)), "3 days"},
	}

	for _, c := range cases {
		t.Run(c.duration.String(), func(t *testing.T) {
			got := humanize(c.duration)

			if got != c.want {
				t.Errorf("expected %q, got %q", c.want, got)
			}
		})
	}
}

// Tests for the content of the Reminder message
func TestReminderContent(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	// Note: here I don't flood with test cases that examine various due dates,
	// because the date humanization (e.g: duration -> 1 minute) is tested ]
	// separately
	cases := []struct{
		name string
		task *models.Task
		want string
	}{
		{
			name: "only task title",
			task: &models.Task{Title: "testing!"},
			want: "testing!",
		},
		{
			name: "task title without due date if due date close",
			task: &models.Task{Title: "test", DueDate: now.Add(59 * time.Second)},
			want: "test",
		},
		{
			name: "task title without due date if at time of due date",
			task: &models.Task{Title: "test", DueDate: now},
			want: "test",
		},
		{
			name: "task title without due date if due date passed",
			task: &models.Task{Title: "test", DueDate: now.Add(-5 * time.Minute)},
			want: "test",
		},
		{
			name: "task title with due date duration in minutes (singular)",
			task: &models.Task{Title: "test", DueDate: now.Add(1 * time.Minute)},
			want: "'test' due in 1 minute",
		},
		{
			name: "task title with due date duration in days (singular)",
			task: &models.Task{Title: "test", DueDate: now.Add(28 * time.Hour)},
			want: "'test' due in 1 day",
		},
		{
			name: "task title with due date duration in days (plural)",
			task: &models.Task{Title: "test", DueDate: now.Add(55 * time.Hour)},
			want: "'test' due in 2 days",
		},
		{
			name: "task title with due date duration in weeks (singular)",
			task: &models.Task{Title: "test", DueDate: now.Add(9 * 24 * time.Hour)},
			want: "'test' due in 1 week",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := getReminder(c.task, &now)

			if got != c.want {
				t.Errorf("expected %q, got %q", c.want, got)
			}
		})
	}
}
