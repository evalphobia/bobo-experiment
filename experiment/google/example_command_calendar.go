package google

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/evalphobia/google-api-go-wrapper/calendar"
	"github.com/evalphobia/google-api-go-wrapper/config"

	"github.com/eure/bobo/command"
	"github.com/eure/bobo/library"
	"github.com/evalphobia/bobo-experiment/i18n"
)

// To use Calendar Command, you should setup, (1) Credentials for Google API (2) OAuth Token for Calendar.
// (1): https://cloud.google.com/docs/authentication/
// (2): https://developers.google.com/calendar/quickstart/go
var CalendarCommand = command.BasicCommandTemplate{
	Help:           "Get Events from Google Calendar",
	MentionCommand: "calendar",
	GenerateFn: func(d command.CommandData) command.Command {
		c := command.Command{}

		calendarCli, err := getGoogleCalendarClient()
		if err != nil {
			errMessage := fmt.Sprintf("[ERROR]\t[getGoogleCalendarClient]\t`%s`", err.Error())
			task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
			c.Add(task)
			return c
		}

		// get email address for target calendar
		email, err := getEmailAddress(d)
		if err != nil {
			errMessage := fmt.Sprintf("[ERROR]\t[getEmailAddress]\t`%s`", err.Error())
			task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
			c.Add(task)
			return c
		}

		// fetch events from google calendar
		command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting events of [%s] ...", email)).Run()
		list, err := calendarCli.EventList(email, 10)
		if err != nil {
			errMessage := fmt.Sprintf("[ERROR]\t[EventList]\t`%s`", err.Error())
			task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
			c.Add(task)
			return c
		}

		// format and output events to slack
		msg := formatCalendarAsSlackMessage(list.List)
		task := command.NewReplyEngineTask(d.Engine, d.Channel, msg)
		c.Add(task)
		return c
	},
}

// returns email address in d.TextOther.
// if it's empty returns sender's email address .
func getEmailAddress(d command.CommandData) (string, error) {
	// add atmark as a mention when d.TextOther is empty
	text := library.TrimSigns(d.TextOther)
	if text == "" {
		text = "@" + d.SenderID
	}

	switch {
	case strings.HasPrefix(text, "@"):
		// Get email from Slack User API
		u, err := d.Engine.GetUserByID(strings.TrimPrefix(text, "@"))
		if err != nil {
			return "", err
		}
		return u.Email, nil
	default:
		// Validate to correct email format
		e, err := mail.ParseAddress(text)
		if err != nil {
			return "", errors.New(i18n.Message("Target format is invalid, use @mention or correct email address."))
		}
		return e.Address, nil
	}
}

func formatCalendarAsSlackMessage(list []calendar.Event) string {
	now := time.Now()
	borderTime := now.AddDate(0, 0, 1)

	result := make([]string, 0, len(list))
	result = append(result, "```")
	for _, ev := range list {
		// Skip events after tommorow
		if ev.StartTime.After(borderTime) {
			continue
		}

		// for a one-line message
		msg := make([]string, 0, 10)

		// add datetime
		switch {
		case ev.IsAllDayEvent:
			msg = append(msg, i18n.Message("[AllDay] [%s - %s]", formatLocalDate(ev.StartTime), formatLocalDate(ev.EndTime)))
		default:
			msg = append(msg, fmt.Sprintf("[%s - %s]", formatLocalDateTime(ev.StartTime), formatLocalTime(ev.EndTime)))
		}

		// Add summary and location
		msg = append(msg, ev.Summary)
		if ev.Location != "" {
			msg = append(msg, " ("+ev.Location+")")
		}
		result = append(result, strings.Join(msg, " "))
	}
	result = append(result, "```")

	return strings.Join(result, "\n")
}

func formatLocalDateTime(dt time.Time) string {
	return dt.Format("01/02 15:04")
}

func formatLocalTime(dt time.Time) string {
	return dt.Format("15:04")
}

func formatLocalDate(dt time.Time) string {
	return dt.Format("01/02")
}

var calendarOnce sync.Once
var calendarCli *calendar.Calendar

func getGoogleCalendarClient() (*calendar.Calendar, error) {
	var err error
	calendarOnce.Do(func() {
		calendarCli, err = calendar.New(config.Config{})
	})
	return calendarCli, err
}
