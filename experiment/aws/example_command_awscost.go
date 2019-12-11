package aws

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/evalphobia/aws-sdk-go-wrapper/cloudwatch"
	"github.com/evalphobia/aws-sdk-go-wrapper/config"

	"github.com/eure/bobo/command"
)

var _ command.CommandTemplate = AWSCostCommand{}

type AWSCostCommand struct {
	Services []string
}

func (AWSCostCommand) GetMentionCommand() string {
	return "awscost"
}

func (AWSCostCommand) GetHelp() string {
	return "Get AWS Cost from CloudWatch"
}

func (AWSCostCommand) HasHelp() bool {
	return true
}

func (AWSCostCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (a AWSCostCommand) Exec(d command.CommandData) {
	a.runAWSCost(d)
}

// main logic.
func (a AWSCostCommand) runAWSCost(d command.CommandData) {
	// Validation for AWS client
	// e.g.) AWS_ACCESS_KEY_ID is empty
	if err := validateCloudWatchClient(); err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[validateCloudWatchClient]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	// Use date from the message, or use yesterday.
	endTime, err := getEndTime(d.TextOther)
	if err != nil {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf("Invalid date format: [%s]", d.TextOther)).Run()
		return
	}

	// Get costs of the target services.
	_ = command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf("Getting costs on [%s]...", endTime.Format("2006-01-02"))).Run()
	targetSerivces := a.getServices()
	costs, err := fetchAllCosts(endTime, targetSerivces)
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[fetchAllCosts]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	// format costs data for Slack message
	msg := costs.formatAsOutputReport(endTime)
	_ = command.NewReplyEngineTask(d.Engine, d.Channel, msg).Run()
}

// Return given date of 23:59:59.
// If text is empty, return yesterday.
func getEndTime(text string) (time.Time, error) {
	dt := time.Now().In(time.UTC).AddDate(0, 0, -1) // yesterday
	if text != "" {
		var err error
		dt, err = time.Parse("2006-01-02", text)
		if err != nil {
			return dt, err
		}
	}
	endTime := time.Date(dt.Year(), dt.Month(), dt.Day(), 23, 59, 59, 0, time.UTC)
	return endTime, nil
}

func fetchAllCosts(endTime time.Time, targetServices []string) (Costs, error) {
	c := Costs{}
	total, err := fetchCosts(endTime)
	if err != nil {
		return c, err
	}

	svcTotal := 0.0
	svcCost := make(map[string]float64, len(targetServices))
	for _, s := range targetServices {
		cost, err := fetchCosts(endTime, s)
		if err != nil {
			return c, err
		}
		svcCost[s] = cost
		svcTotal += cost
	}
	c.Total = total
	c.Other = total - svcTotal
	c.Services = svcCost
	return c, nil
}

func (a AWSCostCommand) getServices() []string {
	if len(a.Services) == 0 {
		return defaultAWSServices
	}
	return a.Services
}

func fetchCosts(endTime time.Time, serviceName ...string) (float64, error) {
	resp1, err := fetchMetrics(endTime, serviceName...)
	if err != nil {
		return 0, err
	}
	costToday := getFirstMaximum(resp1.Datapoints)
	// 1st day does not need a diff of the previous day.
	if endTime.Day() == 1 {
		return costToday, nil
	}

	resp2, err := fetchMetrics(endTime.AddDate(0, 0, -1), serviceName...)
	if err != nil {
		return 0, err
	}
	costYesterday := getFirstMaximum(resp2.Datapoints)
	return costToday - costYesterday, nil
}

func fetchMetrics(endTime time.Time, serviceName ...string) (*cloudwatch.MetricStatisticsResponse, error) {
	cli, _ := getOrCreateCloudWatchClient()
	input := cloudwatch.MetricStatisticsInput{
		Namespace:  "AWS/Billing",
		MetricName: "EstimatedCharges",
		DimensionsMap: map[string]string{
			"Currency": "USD",
		},
		StartTime:  endTime.AddDate(0, 0, -1),
		EndTime:    endTime,
		Period:     86400,
		Statistics: []string{"Maximum"},
	}
	if len(serviceName) != 0 {
		input.DimensionsMap["ServiceName"] = serviceName[0]
	}

	return cli.GetMetricStatistics(input)
}

func getFirstMaximum(list []cloudwatch.Datapoint) float64 {
	if len(list) == 0 {
		return 0
	}
	return list[0].Maximum
}

var cwOnce sync.Once
var cwCli *cloudwatch.CloudWatch

func getOrCreateCloudWatchClient() (*cloudwatch.CloudWatch, error) {
	var err error
	cwOnce.Do(func() {
		cwCli, err = cloudwatch.New(config.Config{})
	})
	return cwCli, err
}

func validateCloudWatchClient() error {
	_, err := getOrCreateCloudWatchClient()
	return err
}

var defaultAWSServices = []string{
	"AmazonApiGateway",
	"AmazonCloudWatch",
	"AmazonEC2",
	"AmazonECR",
	"AmazonDynamoDB",
	"AmazonElastiCache",
	"AmazonES",
	"AmazonGuardDuty",
	"AmazonInspector",
	"AmazonKinesis",
	"AmazonKinesisFirehose",
	"AmazonRDS",
	"AmazonRekognition",
	"AmazonRoute53",
	"AmazonS3",
	"AmazonSageMaker",
	"AmazonSES",
	"AmazonSNS",
	"AWSDataTransfer",
	"AWSIoT",
	"AWSLambda",
	"AWSQueueService",
	"CodeBuild",
}
