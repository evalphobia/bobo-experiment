package aws

import (
	"fmt"
	"regexp"

	"github.com/eure/bobo/command"
	"github.com/evalphobia/awscost/cloudwatch"

	"github.com/evalphobia/bobo-experiment/i18n"
)

var _ command.CommandTemplate = CostCommandByCloudWatch{}

type CostCommandByCloudWatch struct {
	Services []string
}

func (CostCommandByCloudWatch) GetMentionCommand() string {
	return "awscost"
}

func (CostCommandByCloudWatch) GetHelp() string {
	return "Get AWS Cost from ClowdWatch"
}

func (CostCommandByCloudWatch) HasHelp() bool {
	return true
}

func (CostCommandByCloudWatch) GetRegexp() *regexp.Regexp {
	return nil
}

func (a CostCommandByCloudWatch) Exec(d command.CommandData) {
	a.runAWSCost(d)
}

// main logic.
func (a CostCommandByCloudWatch) runAWSCost(d command.CommandData) {
	// Use date from the message, or use yesterday.
	endTime, err := getEndTimeFromString(d.TextOther)
	if err != nil {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Invalid date format: [%s]", d.TextOther)).Run()
		return
	}

	_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting costs on [%s]...", endTime.Format("2006-01-02"))).Run()

	// Get costs of AWS services.
	targetSerivces := a.getServices()
	costs, err := cloudwatch.FetchDailyCost(endTime, targetSerivces...)
	if err != nil {
		errMessage := fmt.Sprintf("[ERROR]\t[FetchDailyCost]\t`%s`", err.Error())
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, errMessage).Run()
		return
	}

	// format costs data for Slack message
	costs.SetDate(endTime)
	msg := fmt.Sprintf("```%s```", costs.FormatAsOutputReport())
	_ = command.NewReplyEngineTask(d.Engine, d.Channel, msg).Run()
}

func (a CostCommandByCloudWatch) getServices() []string {
	if len(a.Services) == 0 {
		return defaultAWSServices
	}
	return a.Services
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
