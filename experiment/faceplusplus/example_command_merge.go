package faceplusplus

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/evalphobia/go-face-plusplus/beautify"
	"github.com/evalphobia/go-face-plusplus/config"

	"github.com/eure/bobo/command"
	"github.com/eure/bobo/library"
)

var _ command.CommandTemplate = &MergeCommand{}

type MergeCommand struct {
	MergeRate    int
	UseWhitelist bool
	Whitelist    []string
	UseBlacklist bool
	Blacklist    []string

	listOnce  sync.Once
	blacklist map[string]struct{}
	whitelist map[string]struct{}
}

func (*MergeCommand) GetMentionCommand() string {
	return "merge"
}

func (*MergeCommand) GetHelp() string {
	return "Merge face images by Face++"
}

func (*MergeCommand) HasHelp() bool {
	return true
}

func (*MergeCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (m *MergeCommand) Exec(d command.CommandData) {
	m.init()
	m.runMergeFace(d)
}

func (m *MergeCommand) init() {
	m.listOnce.Do(func() {
		m.whitelist = make(map[string]struct{})
		for _, s := range m.Whitelist {
			m.whitelist[s] = struct{}{}
		}
		m.blacklist = make(map[string]struct{})
		for _, s := range m.Blacklist {
			m.blacklist[s] = struct{}{}
		}
	})
}

// main logic
func (m *MergeCommand) runMergeFace(d command.CommandData) {
	switch {
	case m.isInBlacklist(d.SenderName),
		!m.isInWhitelist(d.SenderName):
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, "No!").Run()
		return
	}

	url := strings.Fields(d.TextOther)
	if len(url) < 2 {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, "Set Two URLs").Run()
		return
	}

	// url validation
	url1 := library.TrimSigns(url[0])
	url2 := library.TrimSigns(url[1])
	switch {
	case !strings.HasPrefix(url1, "http"),
		!strings.HasPrefix(url2, "http"):
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, "Invalid URL. It must begin with [http/https]").Run()
		return
	}

	_ = command.NewReplyEngineTask(d.Engine, d.Channel, "Merging...").Run()

	resp, err := mergeFaceImage(url2, url1, m.getMergeRate())
	if err != nil {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf("[ERROR] [mergeFaceImage] `%s`", err.Error())).Run()
		return
	}

	buf := bytes.NewBuffer(resp)
	err = command.NewUploadEngineTask(d.Engine, d.Channel, buf, "result.jpg").Run()
	if err != nil {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf("[ERROR] [NewUploadEngineTask] `%s`", err.Error())).Run()
		return
	}
}

func (m *MergeCommand) isInBlacklist(name string) bool {
	if !m.UseBlacklist {
		return false
	}
	_, ok := m.blacklist[name]
	return ok
}

func (m *MergeCommand) isInWhitelist(name string) bool {
	if !m.UseWhitelist {
		return true
	}
	_, ok := m.whitelist[name]
	return ok
}

func (m *MergeCommand) getMergeRate() int {
	if m.MergeRate > 0 {
		return m.MergeRate
	}
	const defaultMergeRate = 75 // 75%
	return defaultMergeRate
}

// execute merge face API.
func mergeFaceImage(fromURL, toURL string, mergeRate int) ([]byte, error) {
	svc, err := beautify.New(config.Config{})
	if err != nil {
		return nil, err
	}

	resp, err := svc.MergeFace(beautify.MergeFaceRequest{
		TemplateURL: toURL,
		MergeURL:    fromURL,
		MergeRate:   mergeRate,
	})
	switch {
	case err != nil:
		return nil, err
	case resp.ErrorMessage != "":
		return nil, errors.New(resp.ErrorMessage)
	}

	return resp.GetResultImage()
}
