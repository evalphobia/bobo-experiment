package aws

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/evalphobia/aws-sdk-go-wrapper/cloudwatch"
	"github.com/evalphobia/aws-sdk-go-wrapper/config"
	"github.com/evalphobia/aws-sdk-go-wrapper/dynamodb"
	"github.com/evalphobia/bobo-experiment/i18n"

	"github.com/eure/bobo/command"
)

var _ command.CommandTemplate = DynamoDBCommand{}

type DynamoDBCommand struct {
	Metrics       []string
	MaxBorder     int
	ChartEndpoint string
}

func (DynamoDBCommand) GetMentionCommand() string {
	return "dynamodb"
}

func (DynamoDBCommand) GetHelp() string {
	return "Get stats of AWS DynamoDB Table"
}

func (DynamoDBCommand) HasHelp() bool {
	return true
}

func (DynamoDBCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (s DynamoDBCommand) Exec(d command.CommandData) {
	c := s.run(d)
	c.Exec()
}

func (s DynamoDBCommand) run(d command.CommandData) command.Command {
	c := command.Command{}

	ddbCli, err := getOrCreateDynamoDBClient()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[getOrCreateDynamoDBClient]\t`%s`", err.Error())
		task := command.NewReplyEngineTask(d.Engine, d.Channel, errMessage)
		c.Add(task)
		return c
	}

	// fetch dynamodb table list.
	text := d.TextOther
	command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting dynamodb stats of [%s] ...", text)).Run()
	list, err := ddbCli.ListTables()
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[ListTables]\t`%s`", err.Error())
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

func (s DynamoDBCommand) createGraph(stats ddbStats) (string, error) {
	dp, err := stats.FetchDetail(s.Metrics...)
	if err != nil {
		return "", fmt.Errorf("[ERROR]\t[FetchDetail]\t`%s`", err.Error())
	}
	if len(dp) == 0 {
		return "", nil
	}

	title := i18n.Message("DynamoDB Metrics (Maximum): %s", stats.getFirstTableName())
	url, err := createChartURL(s.ChartEndpoint, title, dp)
	if err != nil {
		return "", fmt.Errorf("[ERROR]\t[createChartURL]\t`%s`", err.Error())
	}
	return url, nil
}

func (s DynamoDBCommand) createStats(target string, nameList []string) ddbStats {
	stats := ddbStats{
		target: target,
		border: s.MaxBorder,
	}

	const defaultBorder = 30
	if stats.border == 0 {
		stats.border = defaultBorder
	}

	// filter target tables
	list := make([]ddbStat, 0, len(nameList))
	for _, name := range nameList {
		if !strings.Contains(name, target) {
			continue
		}

		data := ddbStat{
			Name: name,
		}
		// exact match will contain only one data.
		if name == target {
			list = []ddbStat{data}
			break
		}

		list = append(list, data)
	}
	stats.tables = list
	return stats
}

type ddbStats struct {
	target string
	border int

	tables []ddbStat
}

type ddbStat struct {
	Name        string
	ItemCount   int
	TableSizeMB int
	Status      string

	GSIs []ddbStat
}

func (s *ddbStats) ShouldFetchDetail(url string) bool {
	return len(s.tables) == 1 && canCreateChart(url)
}

func (s *ddbStats) FetchDetail(metrics ...string) (Datapoints, error) {
	return fetchDynamoDBMetrics(s.getFirstTableName(), metrics...)
}

func (s *ddbStats) MakeMessage() (string, error) {
	switch {
	case s.isEmpty():
		return i18n.Message("[%s] does not match any tables.", s.target), nil
	case s.hasTooMany():
		return s.outputOnlyNames(), nil
	}

	// fetching message size
	ddbCli, _ := getOrCreateDynamoDBClient()
	for i, ss := range s.tables {
		desc, err := ddbCli.DescribeTable(ss.Name)
		if err != nil {
			return "", fmt.Errorf("[ERROR]\t[DescribeTable]\t`%s`", err.Error())
		}
		ss.ItemCount = int(desc.ItemCount)
		ss.TableSizeMB = int(desc.TableSizeBytes) / (1024 ^ 2)
		ss.Status = desc.TableStatus
		for _, gsi := range desc.GlobalSecondaryIndexes {
			ss.GSIs = append(ss.GSIs, ddbStat{
				Name:        gsi.IndexName,
				ItemCount:   int(gsi.ItemCount),
				TableSizeMB: int(gsi.IndexSizeBytes) / (1024 ^ 2),
				Status:      gsi.IndexStatus,
			})
		}
		s.tables[i] = ss
	}
	return s.outputStats(), nil
}

func (s *ddbStats) outputOnlyNames() string {
	result := make([]string, len(s.tables))
	for i, t := range s.tables {
		result[i] = t.Name
	}

	return "```\n" + strings.Join(result, "\n") + "\n```"
}

func (s *ddbStats) outputStats() string {
	result := make([]string, 0, len(s.tables)*2)
	result = append(result, "Name\t|\tStatus\t|\tCount (MB)")
	result = append(result, "====================================")

	for _, t := range s.tables {
		result = append(result, fmt.Sprintf("%s\t|\t%s\t|\t%s (%s)", t.Name, t.Status, i18n.CommaNumber(t.ItemCount), i18n.CommaNumber(t.TableSizeMB)))
		for _, gsi := range t.GSIs {
			result = append(result, fmt.Sprintf("\t- %s\t|\t%s\t|\t%s (%s)", gsi.Name, gsi.Status, i18n.CommaNumber(gsi.ItemCount), i18n.CommaNumber(gsi.TableSizeMB)))
		}
	}

	return "```\n" + strings.Join(result, "\n") + "\n```"
}

func (s *ddbStats) isEmpty() bool {
	return len(s.tables) == 0
}

func (s *ddbStats) hasTooMany() bool {
	return len(s.tables) > s.border
}

func (s *ddbStats) getFirstTableName() string {
	if len(s.tables) == 0 {
		return ""
	}
	return s.tables[0].Name
}

func fetchDynamoDBMetrics(tableName string, metrics ...string) (Datapoints, error) {
	endTime := time.Now()
	startTime := endTime.Add(-300 * time.Minute)
	baseInput := cloudwatch.MetricStatisticsInput{
		Namespace: "AWS/DynamoDB",
		DimensionsMap: map[string]string{
			"TableName": tableName,
		},
		StartTime:  startTime,
		EndTime:    endTime,
		Period:     300,
		Statistics: []string{"Sum", "Maximum", "Average"},
	}

	if len(metrics) == 0 {
		metrics = defaultDynamoDBMetrics
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

var defaultDynamoDBMetrics = []string{
	"ConditionalCheckFailedRequests",
	"ConsumedReadCapacityUnits",
	"ConsumedWriteCapacityUnits",
	"OnlineIndexConsumedWriteCapacity",
	"OnlineIndexPercentageProgress",
	"OnlineIndexThrottleEvents",
	"ReadThrottleEvents",
	"ReturnedItemCount",
	"SystemErrors",
	"TimeToLiveDeletedItemCount",
	"ThrottledRequests",
	"TransactionConflict",
	"UserErrors",
	"WriteThrottleEvents",
	"SuccessfulRequestLatency",
	"ProvisionedReadCapacityUnits",
	"ProvisionedWriteCapacityUnits",
	"MaxProvisionedTableReadCapacityUtilization",
	"MaxProvisionedTableWriteCapacityUtilization",
}

var ddbOnce sync.Once
var ddbCli *dynamodb.DynamoDB

func getOrCreateDynamoDBClient() (*dynamodb.DynamoDB, error) {
	var err error
	ddbOnce.Do(func() {
		ddbCli, err = dynamodb.New(config.Config{})
	})
	return ddbCli, err
}
