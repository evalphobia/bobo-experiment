bobo-experiment
----

[![GoDoc][1]][2] [![License: MIT][3]][4] [![Release][5]][6] [![Build Status][7]][8] [![Co decov Coverage][11]][12] [![Go Report Card][13]][14] [![Code Climate][19]][20] [![BCH compliance][21]][22] [![Downloads][15]][16]

[1]: https://godoc.org/github.com/evalphobia/bobo-experiment?status.svg
[2]: https://godoc.org/github.com/evalphobia/bobo-experiment
[3]: https://img.shields.io/badge/License-MIT-blue.svg
[4]: LICENSE.md
[5]: https://img.shields.io/github/release/evalphobia/bobo-experiment.svg
[6]: https://github.com/evalphobia/bobo-experiment/releases/latest
[7]: https://travis-ci.org/evalphobia/bobo-experiment.svg?branch=master
[8]: https://travis-ci.org/evalphobia/bobo-experiment
[9]: https://coveralls.io/repos/evalphobia/bobo-experiment/badge.svg?branch=master&service=github
[10]: https://coveralls.io/github/evalphobia/bobo-experiment?branch=master
[11]: https://codecov.io/github/evalphobia/bobo-experiment/coverage.svg?branch=master
[12]: https://codecov.io/github/evalphobia/bobo-experiment?branch=master
[13]: https://goreportcard.com/badge/github.com/evalphobia/bobo-experiment
[14]: https://goreportcard.com/report/github.com/evalphobia/bobo-experiment
[15]: https://img.shields.io/github/downloads/evalphobia/bobo-experiment/total.svg?maxAge=1800
[16]: https://github.com/evalphobia/bobo-experiment/releases
[17]: https://img.shields.io/github/stars/evalphobia/bobo-experiment.svg
[18]: https://github.com/evalphobia/bobo-experiment/stargazers
[19]: https://codeclimate.com/github/evalphobia/bobo-experiment/badges/gpa.svg
[20]: https://codeclimate.com/github/evalphobia/bobo-experiment
[21]: https://bettercodehub.com/edge/badge/evalphobia/bobo-experiment?branch=master
[22]: https://bettercodehub.com/



Experimental Bot Commands for [eure/bobo](https://github.com/eure/bobo)


# Install

```bash
$ go get -u github.com/evalphobia/bobo-experiment
```

# Build

```bash
$ make build
```

for Raspberry Pi

```bash
$ make build-arm6
```

# Run

```bash
SLACK_RTM_TOKEN=xoxb-0000... ./bin/bobo
```

## Environment variables

|Name|Description|
|:--|:--|
| `SLACK_RTM_TOKEN` | [Slack Bot Token](https://slack.com/apps/A0F7YS25R-bots) |
| `SLACK_BOT_TOKEN` | [Slack Bot Token](https://slack.com/apps/A0F7YS25R-bots) |
| `SLACK_TOKEN` | [Slack Bot Token](https://slack.com/apps/A0F7YS25R-bots) |
| `BOBO_DEBUG` | Flag for debug logging. Set [boolean like value](https://golang.org/pkg/strconv/#ParseBool). |
| `BOBO_LANG` | Language setting for bot. Set it as [ISO 639-1 code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes). |
| `AWS_ACCESS_KEY_ID` | [AWS Access Key ID](https://github.com/aws/aws-sdk-go/blob/bef02444773a49eaf30cdd615920b56896827c06/aws/credentials/env_provider.go) |
| `AWS_SECRET_ACCESS_KEY` | [AWS Secret Access Key](https://github.com/aws/aws-sdk-go/blob/bef02444773a49eaf30cdd615920b56896827c06/aws/credentials/env_provider.go) |
| `FACEPP_API_KEY` | [API Key of Face++](https://github.com/evalphobia/go-face-plusplus). |
| `FACEPP_API_SECRET` | [API Secret of Face++](https://github.com/evalphobia/go-face-plusplus). |
| `GOOGLE_API_OAUTH_CREDENTIALS` | [Google API OAuth credentials path](https://developers.google.com/calendar/quickstart/go). |
| `GOOGLE_API_OAUTH_TOKEN_FILE` | [Google API OAuth Token path](https://developers.google.com/calendar/quickstart/go). |
| `OPENAI_API_KEY` | [OepnAI API Key](https://github.com/tmc/langchaingo/blob/7ea734523e39f59ebdec85796d9307573db4fbda/llms/openai/openaillm_option.go#L4) |


## Experimental Commands

- AWS
    - Cost
    - SQS Queue stats
- Face++
    - MergeFace
- Google
    - Calendar
- [LangChain](https://github.com/tmc/langchaingo)
    - OpenAI GPT
