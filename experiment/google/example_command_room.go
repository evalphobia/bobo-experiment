package google

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/evalphobia/bobo-experiment/i18n"
	"github.com/evalphobia/google-api-go-wrapper/calendar"

	"github.com/eure/bobo/command"
)

var _ command.CommandTemplate = &RoomCommand{}

type RoomCommand struct {
	roomIDs []string
}

func (RoomCommand) GetMentionCommand() string {
	return "room"
}

func (RoomCommand) GetHelp() string {
	return "Get empty rooms from Google Calendar"
}

func (RoomCommand) HasHelp() bool {
	return true
}

func (RoomCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (a *RoomCommand) Exec(d command.CommandData) {
	c := a.runRoom(d)
	c.Exec()
}

// main logic.
func (a *RoomCommand) runRoom(d command.CommandData) command.Command {
	c := command.Command{}

	_, err := getGoogleCalendarClient()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[getGoogleCalendarClient]\t`%s`", err.Error())
		task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
		c.Add(task)
		return c
	}

	if len(a.roomIDs) == 0 {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting rooms...")).Run()
		ids, err := fetchAllResourceCalendarIDs()
		if err != nil {
			task := command.NewReplyEngineTask(d.Engine, d.Channel, err.Error())
			c.Add(task)
			return c
		}
		a.roomIDs = ids
	}
	if len(a.roomIDs) == 0 {
		task := command.NewReplyEngineTask(d.Engine, d.Channel, "[ERROR]\t[CalendarList]\t`No valid resources`")
		c.Add(task)
		return c
	}

	_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting room events...")).Run()
	events, err := fetchEventsOfRooms(a.roomIDs)
	if err != nil {
		task := command.NewReplyEngineTask(d.Engine, d.Channel, err.Error())
		c.Add(task)
		return c
	}

	sort.Sort(events)
	msg := events.makeMessage()
	msg = "```\n" + msg + "\n```"
	task := command.NewReplyEngineTask(d.Engine, d.Channel, msg)
	c.Add(task)
	return c
}

func fetchAllResourceCalendarIDs() ([]string, error) {
	const resourceSuffix = "@resource.calendar.google.com"

	cli, _ := getGoogleCalendarClient()

	result := make([]string, 0, 1024)
	nextPageToken := ""
	for {
		resp, err := cli.CalendarListWithOption(calendar.CalendarListOption{
			PageToken:  nextPageToken,
			MaxResults: 250,
		})
		if err != nil {
			return nil, fmt.Errorf("[ERROR]\t[CalendarList]\t`%s`", err.Error())
		}

		for _, v := range resp.List {
			if strings.HasSuffix(v.ID, resourceSuffix) {
				result = append(result, v.ID)
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		nextPageToken = resp.NextPageToken
	}
	return result, nil
}

func fetchEventsOfRooms(ids []string) (RoomEvents, error) {
	cli, _ := getGoogleCalendarClient()
	now := time.Now()
	timeBorder := now.Add(10 * time.Minute)

	results := make([]RoomEvent, 0, len(ids))
	for _, id := range ids {
		resp, err := cli.EventListWithOption(id, calendar.EventListOption{
			TimeMin:      now,
			SingleEvents: true,
			OrderBy:      calendar.OrderByStartTime,
			MaxResults:   1,
		})
		if err != nil {
			return nil, fmt.Errorf("[ERROR]\t[EventList]\t`%s`", err.Error())
		}
		if len(resp.List) == 0 {
			continue
		}

		ev := resp.List[0]
		result := RoomEvent{
			Room:      ev.Location,
			Event:     ev.Summary,
			StartTime: ev.StartTime,
			EndTime:   ev.EndTime,
		}
		if ev.StartTime.After(timeBorder) {
			result.IsEmpty = true
		}
		results = append(results, result)
	}
	return results, nil
}

type RoomEvents []RoomEvent

func (l RoomEvents) Len() int {
	return len(l)
}

func (l RoomEvents) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l RoomEvents) Less(i, j int) bool {
	return l[i].Room > l[j].Room
}

func (l RoomEvents) makeMessage() string {
	results := make([]string, len(l))
	for i, e := range l {
		results[i] = e.makeSentence()
	}
	return strings.Join(results, "\n")
}

type RoomEvent struct {
	Room      string
	Event     string
	StartTime time.Time
	EndTime   time.Time
	IsEmpty   bool
}

func (e RoomEvent) makeSentence() string {
	if e.IsEmpty {
		return i18n.Message("[%s]\t\t(Available)", e.Room)
	}
	return i18n.Message("[%s]\t\t%s (%s - %s)", e.Room, e.Event, formatLocalDateTime(e.StartTime), formatLocalTime(e.EndTime))
}
