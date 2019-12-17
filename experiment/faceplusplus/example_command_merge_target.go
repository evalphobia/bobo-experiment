package faceplusplus

import (
	"bytes"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/eure/bobo/command"
	"github.com/eure/bobo/library"
	"github.com/evalphobia/bobo-experiment/i18n"
)

var _ command.CommandTemplate = &MergeTargetCommand{}

type MergeTargetCommand struct {
	TargetName      string
	TargetURLs      []string // url list to merge
	MergeFromTarget bool

	MergeRate    int
	UseWhitelist bool
	Whitelist    []string
	UseBlacklist bool
	Blacklist    []string

	listOnce  sync.Once
	blacklist map[string]struct{}
	whitelist map[string]struct{}
}

func (m *MergeTargetCommand) GetMentionCommand() string {
	return "merge-" + m.TargetName
}

func (m *MergeTargetCommand) GetHelp() string {
	return "Merge face images with " + m.TargetName
}

func (*MergeTargetCommand) HasHelp() bool {
	return true
}

func (*MergeTargetCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (m *MergeTargetCommand) Exec(d command.CommandData) {
	m.init()
	m.runMergeFace(d)
}

func (m *MergeTargetCommand) init() {
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
func (m *MergeTargetCommand) runMergeFace(d command.CommandData) {
	switch {
	case m.isInBlacklist(d.SenderName),
		!m.isInWhitelist(d.SenderName):
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("No!")).Run()
		return
	}

	url := strings.Fields(d.TextOther)
	if len(url) < 1 {
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Set a URL")).Run()
		return
	}

	// url validation
	url1 := library.TrimSigns(url[0])
	switch {
	case !strings.HasPrefix(url1, "http"):
		_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Invalid URL. It must begin with [http/https]")).Run()
		return
	}

	_ = command.NewReplyEngineTask(d.Engine, d.Channel, i18n.Message("Merging...")).Run()

	var mergeRate int
	if len(url) >= 2 {
		mergeRateStr := url[1]
		mergeRate, _ = strconv.Atoi(mergeRateStr)
	}

	resp, err := m.mergeFaceImage(url1, m.getMergeRate(mergeRate))
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

func (m *MergeTargetCommand) isInBlacklist(name string) bool {
	if !m.UseBlacklist {
		return false
	}
	_, ok := m.blacklist[name]
	return ok
}

func (m *MergeTargetCommand) isInWhitelist(name string) bool {
	if !m.UseWhitelist {
		return true
	}
	_, ok := m.whitelist[name]
	return ok
}

func (m *MergeTargetCommand) getMergeRate(mergeRate int) int {
	switch {
	case mergeRate > 0:
		return mergeRate
	case m.MergeRate > 0:
		return m.MergeRate
	}
	const defaultMergeRate = 75 // 75%
	return defaultMergeRate
}

func (m *MergeTargetCommand) mergeFaceImage(url1 string, mergeRate int) ([]byte, error) {
	imgList := m.TargetURLs
	url2 := imgList[rand.Intn(len(imgList))]

	fromURL, toURL := url1, url2
	if m.MergeFromTarget {
		fromURL, toURL = toURL, fromURL
	}

	return mergeFaceImage(fromURL, toURL, mergeRate)
}
