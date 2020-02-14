package aws

import (
	"fmt"
	"regexp"

	"github.com/eure/bobo/command"
	"github.com/evalphobia/awscost/costexplorer"

	"github.com/evalphobia/bobo-experiment/i18n"
)

var _ command.CommandTemplate = CostCommandByCostExplorer{}

type CostCommandByCostExplorer struct {
	Services []string
}

func (CostCommandByCostExplorer) GetMentionCommand() string {
	return "awscost"
}

func (CostCommandByCostExplorer) GetHelp() string {
	return "Get AWS Cost from CostExplorer"
}

func (CostCommandByCostExplorer) HasHelp() bool {
	return true
}

func (CostCommandByCostExplorer) GetRegexp() *regexp.Regexp {
	return nil
}

func (a CostCommandByCostExplorer) Exec(d command.CommandData) {
	a.runAWSCost(d)
}

// main logic.
func (a CostCommandByCostExplorer) runAWSCost(d command.CommandData) {
	// Use date from the message, or use yesterday.
	endTime, err := getEndTimeFromString(d.TextOther)
	if err != nil {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Invalid date format: [%s]", d.TextOther)).Run()
		return
	}

	_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Getting costs on [%s]...", endTime.Format("2006-01-02"))).Run()

	// Get costs of AWS services.
	costs, err := costexplorer.FetchDailyCost(endTime)
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
