package main

import (
	"testing"
	"time"
	"regexp"

	"code.vikunja.io/api/pkg/models"
)

func TestNotificationMessages(t *testing.T) {
	sampleTime := time.Date(2026, time.October, 20, 22, 16, 31, 0, time.UTC)

	t.Run("full task name when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{ Title: "Buy Apples" }}
		want := regexp.MustCompile(`\bBuy Apples\b`)
		got := getNotificationMessage(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match for %#q, got %q", want, got)
		}
	})

	t.Run("count tasks when more than 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{}, {}}
		want := "You have 2 overdue tasks"
		got := getNotificationMessage(tasks, sampleTime)

		if want != got {
			t.Errorf("Want %q. Got %q", want, got)
		}
	})

	t.Run("show overdue day when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{DueDate: sampleTime.Add(-1 * time.Hour)}}
		want := regexp.MustCompile(`\b1 day overdue\b`)
		got := getNotificationMessage(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match %#q, got %q", want, got)
		}
	})

	t.Run("show overdue days when 1 overdue", func (t *testing.T) {
		tasks := []*models.Task{{DueDate: sampleTime.Add(-28 * time.Hour)}}
		want := regexp.MustCompile(`\b2 days overdue\b`)
		got := getNotificationMessage(tasks, sampleTime)

		if !want.MatchString(got) {
			t.Errorf("Want match %#q, got %q", want, got)
		}
	})
}
