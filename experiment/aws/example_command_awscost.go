package aws

import (
	"fmt"
	"regexp"
	"time"

	"github.com/evalphobia/aws-sdk-go-wrapper/cloudwatch"
	"github.com/evalphobia/bobo-experiment/i18n"

	"github.com/eure/bobo/command"
)

var _ command.CommandTemplate = CostCommand{}

type CostCommand struct {
	Services []string
}

func (CostCommand) GetMentionCommand() string {
	return "awscost"
}

func (CostCommand) GetHelp() string {
	return "Get AWS Cost from CloudWatch"
}

func (CostCommand) HasHelp() bool {
	return true
}

func (CostCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (a CostCommand) Exec(d command.CommandData) {
	a.runAWSCost(d)
}

// main logic.
func (a CostCommand) runAWSCost(d command.CommandData) {
	// Validation for AWS client
	// e.g.) AWS_ACCESS_KEY_ID is empty
	if err := validateCloudWatchClient(); err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[validateCloudWatchClient]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	// Use date from the message, or use yesterday.
	endTime, err := getEndTimeFromString(d.TextOther)
	if err != nil {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Invalid date format: [%s]", d.TextOther)).Run()
		return
	}

	// Get costs of the target services.
	_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting costs on [%s]...", endTime.Format("2006-01-02"))).Run()
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

func (a CostCommand) getServices() []string {
	if len(a.Services) == 0 {
		return defaultAWSServices
	}
	return a.Services
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

func fetchCosts(endTime time.Time, serviceName ...string) (float64, error) {
	resp1, err := fetchMetricsForCosts(endTime, serviceName...)
	if err != nil {
		return 0, err
	}
	costToday := resp1.GetFirstValue()
	// 1st day does not need a diff of the previous day.
	if endTime.Day() == 1 {
		return costToday, nil
	}

	resp2, err := fetchMetricsForCosts(endTime.AddDate(0, 0, -1), serviceName...)
	if err != nil {
		return 0, err
	}
	costYesterday := resp2.GetFirstValue()
	return costToday - costYesterday, nil
}

func fetchMetricsForCosts(endTime time.Time, serviceName ...string) (Datapoints, error) {
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

	return fetchCloudWatchMetrics(input)
}

// Return given date of 23:59:59.
// If text is empty, return yesterday.
func getEndTimeFromString(text string) (time.Time, error) {
	dt := time.Now().In(time.UTC).AddDate(0, 0, -1) // yesterday
	if text != "" {
		var err error
		dt, err = time.Parse("2006-01-02", text)
		if err != nil {
			return dt, err
		}
	}
	return getEndTime(dt), nil
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
