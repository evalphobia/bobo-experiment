package aws

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/evalphobia/aws-sdk-go-wrapper/cloudwatch"
	"github.com/evalphobia/aws-sdk-go-wrapper/config"
	"github.com/evalphobia/aws-sdk-go-wrapper/sqs"
	"github.com/evalphobia/bobo-experiment/i18n"

	"github.com/eure/bobo/command"
)

var _ command.CommandTemplate = SQSCommand{}

type SQSCommand struct {
	Metrics       []string
	MaxBorder     int
	ChartEndpoint string
}

func (SQSCommand) GetMentionCommand() string {
	return "sqs"
}

func (SQSCommand) GetHelp() string {
	return "Get stats of AWS SQS Queue"
}

func (SQSCommand) HasHelp() bool {
	return true
}

func (SQSCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (s SQSCommand) Exec(d command.CommandData) {
	c := s.runSQS(d)
	c.Exec()
}

func (s SQSCommand) runSQS(d command.CommandData) command.Command {
	c := command.Command{}

	sqsCli, err := getOrCreateSQSClient()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[getOrCreateSQSClient]\t`%s`", err.Error())
		task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
		c.Add(task)
		return c
	}

	// fetch SQS queue list.
	text := d.TextOther
	command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting sqs stats of [%s] ...", text)).Run()
	list, err := sqsCli.ListAllQueues()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[ListAllQueues]\t`%s`", err.Error())
		task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
		c.Add(task)
		return c
	}

	stats := s.createStats(text, list)
	msg, err := stats.MakeMessage()
	if err != nil {
		task := command.NewReplyEngineTask(d.Engine, d.Channel, err.Error())
		c.Add(task)
		return c
	}

	// format and output events to slack
	command.NewReplyEngineTask(d.Engine, d.Channel, msg).Run()
	if !stats.ShouldFetchDetail(s.ChartEndpoint) {
		return c
	}

	// get detailed metrics from CloudWatch
	url, err := s.createGraph(stats)
	if err != nil {
		task := command.NewReplyEngineTask(d.Engine, d.Channel, err.Error())
		c.Add(task)
		return c
	}

	task := command.NewReplyEngineTask(d.Engine, d.Channel, url)
	c.Add(task)
	return c
}

func (s SQSCommand) createGraph(stats sqsStats) (string, error) {
	dp, err := stats.FetchDetail(s.Metrics...)
	if err != nil {
		return "", fmt.Errorf("[ERROR]\t[FetchDetail]\t`%s`", err.Error())
	}
	if len(dp) == 0 {
		return "", nil
	}

	title := i18n.Message("SQS Metrics (Maximum): %s", stats.getFirstQueueName())
	url, err := createChartURL(s.ChartEndpoint, title, dp)
	if err != nil {
		return "", fmt.Errorf("[ERROR]\t[createChartURL]\t`%s`", err.Error())
	}
	return url, nil
}

func (s SQSCommand) createStats(target string, urlList []string) sqsStats {
	stats := sqsStats{
		target: target,
		border: s.MaxBorder,
	}

	const defaultBorder = 30
	if stats.border == 0 {
		stats.border = defaultBorder
	}

	// filter target queues
	list := make([]sqsStat, 0, len(urlList))
	for _, url := range urlList {
		parts := strings.Split(url, "/")
		name := parts[len(parts)-1]

		if !strings.Contains(name, target) {
			continue
		}

		data := sqsStat{
			URL:  url,
			Name: name,
		}
		// exact match will contain only one data.
		if name == target {
			list = []sqsStat{data}
			break
		}

		list = append(list, data)
	}
	stats.queues = list
	return stats
}

type sqsStats struct {
	target string
	border int

	queues []sqsStat
}

type sqsStat struct {
	URL        string
	Name       string
	Visible    int
	NotVisible int
}

func (s *sqsStats) ShouldFetchDetail(url string) bool {
	return len(s.queues) == 1 && canCreateChart(url)
}

func (s *sqsStats) FetchDetail(metrics ...string) (Datapoints, error) {
	return fetchSQSMetrics(s.getFirstQueueName(), metrics...)
}

func (s *sqsStats) MakeMessage() (string, error) {
	switch {
	case s.isEmpty():
		return i18n.Message("[%s] does not match any queues.", s.target), nil
	case s.hasTooMany():
		return s.outputOnlyNames(), nil
	}

	// fetching message size
	sqsCli, _ := getOrCreateSQSClient()
	for i, ss := range s.queues {
		attrs, err := sqsCli.GetQueueAttributes(ss.URL,
			sqs.AttributeApproximateNumberOfMessages,
			sqs.AttributeApproximateNumberOfMessagesNotVisible,
		)
		if err != nil {
			return "", fmt.Errorf("[ERROR]\t[GetQueueAttributes]\t`%s`", err.Error())
		}
		ss.Visible = attrs.ApproximateNumberOfMessages
		ss.NotVisible = attrs.ApproximateNumberOfMessagesNotVisible
		s.queues[i] = ss
	}
	return s.outputStats(), nil
}

func (s *sqsStats) outputOnlyNames() string {
	result := make([]string, len(s.queues))
	for i, q := range s.queues {
		result[i] = q.Name
	}

	return "```\n" + strings.Join(result, "\n") + "\n```"
}

func (s *sqsStats) outputStats() string {
	result := make([]string, 0, len(s.queues)+2)
	result = append(result, "Name\t|\tVisible (NotVisible)")
	result = append(result, "====================================")

	for _, q := range s.queues {
		result = append(result, fmt.Sprintf("%s\t|\t%d (%d)", q.Name, q.Visible, q.NotVisible))
	}

	return "```\n" + strings.Join(result, "\n") + "\n```"
}

func (s *sqsStats) isEmpty() bool {
	return len(s.queues) == 0
}

func (s *sqsStats) hasTooMany() bool {
	return len(s.queues) > s.border
}

func (s *sqsStats) getFirstQueueName() string {
	if len(s.queues) == 0 {
		return ""
	}
	return s.queues[0].Name
}

func fetchSQSMetrics(queueName string, metrics ...string) (Datapoints, error) {
	endTime := time.Now()
	startTime := endTime.Add(-300 * time.Minute)
	baseInput := cloudwatch.MetricStatisticsInput{
		Namespace: "AWS/SQS",
		DimensionsMap: map[string]string{
			"QueueName": queueName,
		},
		StartTime:  startTime,
		EndTime:    endTime,
		Period:     300,
		Statistics: []string{"Maximum"},
	}

	if len(metrics) == 0 {
		metrics = defaultSQSMetrics
	}

	dataList := make([]Datapoint, 0, 1024)
	for _, metric := range metrics {
		input := baseInput
		input.MetricName = metric
		dp, err := fetchCloudWatchMetrics(input)
		if err != nil {
			return nil, err
		}
		dataList = append(dataList, dp...)
	}

	return dataList, nil
}

var defaultSQSMetrics = []string{
	"NumberOfEmptyReceives",
	"NumberOfMessagesDeleted",
	"NumberOfMessagesReceived",
	"NumberOfMessagesSent",
	"ApproximateNumberOfMessagesVisible",
	"ApproximateNumberOfMessagesNotVisible",
	"ApproximateAgeOfOldestMessage",
	"ApproximateNumberOfMessagesDelayed",
}

var sqsOnce sync.Once
var sqsCli *sqs.SQS

func getOrCreateSQSClient() (*sqs.SQS, error) {
	var err error
	sqsOnce.Do(func() {
		sqsCli, err = sqs.New(config.Config{})
	})
	return sqsCli, err
}
