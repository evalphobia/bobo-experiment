package google

import (
	"fmt"
	"strings"
	"time"

	"github.com/evalphobia/google-api-go-wrapper/calendar"

	"github.com/eure/bobo/command"
	"github.com/evalphobia/bobo-experiment/i18n"
)

var WhereCommand = command.BasicCommandTemplate{
	Help:           "Get current location from Google Calendar",
	MentionCommand: "where",
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

		command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting location of [%s] ...", email)).Run()
		list, err := calendarCli.EventListWithOption(email, calendar.EventListOption{
			TimeMin:      time.Now().Add(-1 * time.Hour),
			SingleEvents: true,
			OrderBy:      calendar.OrderByStartTime,
			MaxResults:   10,
		})
		if err != nil {
			errMessage := fmt.Sprintf("[ERROR]\t[EventList]\t`%s`", err.Error())
			task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
			c.Add(task)
			return c
		}

		res := getCalendarEvent(list.List)
		if !res.hasTimeEvent && !res.hasAllDayEvent {
			task := command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Somewhere around there"))
			c.Add(task)
			return c
		}

		msg := makeMessage(res)
		task := command.NewReplyEngineTask(d.Engine, d.Channel, msg)
		c.Add(task)
		return c
	},
}

type eventResult struct {
	List           []*calendar.Event
	allDay         *calendar.Event
	prev           *calendar.Event
	current        *calendar.Event
	next           *calendar.Event
	hasAllDayEvent bool
	hasTimeEvent   bool
}

func getCalendarEvent(list []calendar.Event) eventResult {
	now := time.Now()
	utcNow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.UTC)

	result := eventResult{}
	for _, vv := range list {
		if !vv.IsStatusConfirmed() {
			continue
		}
		ev := vv // change pointer

		if ev.IsAllDayEvent {
			if ev.StartTime.Before(utcNow) && ev.EndTime.After(utcNow) {
				result.allDay = &ev
				result.hasAllDayEvent = true
			}
			continue
		}

		if ev.StartTime.Before(now) && ev.EndTime.After(now) {
			result.List = append(result.List, &ev)
			result.hasTimeEvent = true
			continue
		}

		// Get a event which was finished within 30min.
		if ev.StartTime.Before(now) && ev.EndTime.Add(30*time.Minute).After(now) {
			result.prev = &ev
			result.hasTimeEvent = true
			continue
		}

		// Get a event which will start within 30min.
		if ev.StartTime.Add(-30*time.Minute).Before(now) && ev.EndTime.After(now) {
			result.next = &ev
			result.hasTimeEvent = true
			continue
		}
	}

	size := len(result.List)
	switch {
	case size == 1:
		result.current = result.List[0]
	case size > 1:
		result.prev = result.List[size-2]
		result.current = result.List[size-1]
	}

	return result
}

func makeMessage(r eventResult) string {
	list := make([]string, 0, 10)
	if r.prev != nil {
		list = append(list, makeSentence(i18n.Message("prev"), r.prev))
	}
	if r.current != nil {
		list = append(list, makeSentence(i18n.Message("now"), r.current))
	}
	if r.next != nil {
		list = append(list, makeSentence(i18n.Message("next"), r.next))
	}

	// all-day event is only used when (1) 'No normal event' or (2) 'all day event with location'
	if r.hasAllDayEvent {
		if r.allDay.Location != "" || !r.hasTimeEvent {
			list = append(list, makeSentence(i18n.Message("all-day"), r.allDay))
		}
	}
	return strings.Join(list, "\n")
}

func makeSentence(title string, ev *calendar.Event) string {
	if ev.Location != "" {
		return i18n.Message("[%s] | Doing [%s] at [%s] (%s - %s)", title, ev.Summary, ev.Location, formatLocalDateTime(ev.StartTime), formatLocalTime(ev.EndTime))
	}
	return i18n.Message("Doing [%s] in somewhere around there (%s - %s)", ev.Summary, formatLocalDateTime(ev.StartTime), formatLocalTime(ev.EndTime))
}
