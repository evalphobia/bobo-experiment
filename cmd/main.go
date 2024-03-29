package main

import (
	"regexp"

	"github.com/eure/bobo"
	"github.com/eure/bobo/command"
	"github.com/eure/bobo/engine/slack"
	"github.com/eure/bobo/log"

	"github.com/evalphobia/bobo-experiment/experiment/aws"
	"github.com/evalphobia/bobo-experiment/experiment/faceplusplus"
	"github.com/evalphobia/bobo-experiment/experiment/google"
	"github.com/evalphobia/bobo-experiment/experiment/langchain"
)

// Entry Point
func main() {
	bobo.Run(bobo.RunOption{
		Engine: &slack.SlackEngine{},
		Logger: &log.StdLogger{
			IsDebug: bobo.IsDebug(),
		},
		CommandSet: command.NewCommandSet(
			command.PingCommand,
			command.ParrotCommand,
			command.GoodMorningCommand,
			command.ReactEmojiCommand,
			command.HelpCommand,
			// aws.CostCommandByCloudWatch{
			// 	Services: nil,
			// },
			aws.CostCommandByCostExplorer{
				Services: nil,
			},
			aws.SQSCommand{
				Metrics: nil,
			},
			&aws.SQSPurgeCommand{
				UseBlacklist: true,
				Blacklist: []string{
					"funky-queue",
				},
				UseWhitelist: true,
				Whitelist: []string{
					"temp-queue",
				},
				WhitelistRegexp: []*regexp.Regexp{
					regexp.MustCompile("^test-.*"),
					regexp.MustCompile("^dev-.*"),
				},
			},
			aws.DynamoDBCommand{
				Metrics: nil,
			},
			&faceplusplus.MergeCommand{
				UseBlacklist: true,
				Blacklist: []string{
					"evalphobia",
				},
				UseWhitelist: false,
				Whitelist:    nil,
			},
			&faceplusplus.MergeTargetCommand{
				TargetName: "obama",
				TargetURLs: []string{
					"https://upload.wikimedia.org/wikipedia/commons/8/8d/President_Barack_Obama.jpg",
					"https://upload.wikimedia.org/wikipedia/commons/c/c6/Official_portrait_of_Barack_Obama-2.jpg",
					"https://www.obamalibrary.gov/sites/default/files/uploads/portals/the-obamas-potus.jpg",
				},
			},
			google.CalendarCommand,
			google.WhereCommand,
			&google.RoomCommand{},
			&langchain.OpenAIGPTCommand{
				Command: "gpt",
			},
		),
	})
}
