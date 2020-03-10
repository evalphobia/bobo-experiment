package aws

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/evalphobia/bobo-experiment/i18n"

	"github.com/eure/bobo/command"
)

var _ command.CommandTemplate = &SQSPurgeCommand{}

type SQSPurgeCommand struct {
	UseBlacklist    bool
	Blacklist       []string
	UseWhitelist    bool
	Whitelist       []string
	WhitelistRegexp []*regexp.Regexp

	listOnce  sync.Once
	blacklist map[string]struct{}
	whitelist map[string]struct{}
}

func (SQSPurgeCommand) GetMentionCommand() string {
	return "sqs:purge"
}

func (SQSPurgeCommand) GetHelp() string {
	return "Purge messages in the AWS SQS Queue"
}

func (SQSPurgeCommand) HasHelp() bool {
	return true
}

func (SQSPurgeCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (s *SQSPurgeCommand) Exec(d command.CommandData) {
	s.init()
	s.runSQSPurge(d)
}

func (s *SQSPurgeCommand) init() {
	s.listOnce.Do(func() {
		s.whitelist = make(map[string]struct{})
		for _, v := range s.Whitelist {
			s.whitelist[v] = struct{}{}
		}
		s.blacklist = make(map[string]struct{})
		for _, v := range s.Blacklist {
			s.blacklist[v] = struct{}{}
		}
	})
}

func (s SQSPurgeCommand) runSQSPurge(d command.CommandData) {
	queueName := d.TextOther
	switch {
	case s.isInBlacklist(queueName),
		!s.isInWhitelist(queueName):
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Queue Name: [%s] is not permitted to be purged", queueName)).Run()
		return
	}

	sqsCli, err := getOrCreateSQSClient()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[getOrCreateSQSClient]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	// fetch SQS queue.
	command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting sqs stats of [%s] ...", queueName)).Run()
	q, err := sqsCli.GetQueue(queueName)
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[GetQueue]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	// output stats
	attrs, err := q.GetAttributes()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[GetAttributes]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}
	result := []string{
		"```",
		fmt.Sprintf("[%s]", queueName),
		"=====================",
		fmt.Sprintf("Visible\t:\t%d", attrs.ApproximateNumberOfMessages),
		fmt.Sprintf("NotVisible\t:\t%d", attrs.ApproximateNumberOfMessagesNotVisible),
		"```",
	}
	_ = command.NewReplyEngineTask(d.Engine, d.Channel, strings.Join(result, "\n")).Run()

	if err := q.Purge(); err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[Purge]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("SQS: [%s] has been purged!", queueName)).Run()
	return
}

func (s SQSPurgeCommand) isInBlacklist(name string) bool {
	if !s.UseBlacklist {
		return false
	}
	_, ok := s.blacklist[name]
	return ok
}

func (s SQSPurgeCommand) isInWhitelist(name string) bool {
	if !s.UseWhitelist {
		return true
	}

	if _, ok := s.whitelist[name]; ok {
		return true
	}
	for _, re := range s.WhitelistRegexp {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}
