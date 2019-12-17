package aws

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/evalphobia/bobo-experiment/i18n"
)

type Costs struct {
	Total    float64
	Other    float64
	Services map[string]float64
}

type KV struct {
	Key   string
	Value float64
}

type KVList []KV

func (l KVList) Len() int {
	return len(l)
}

func (l KVList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l KVList) Less(i, j int) bool {
	if l[i].Value == l[j].Value {
		return l[i].Key < l[j].Key
	}
	return l[i].Value > l[j].Value
}

func (c Costs) formatAsOutputReport(endTime time.Time) string {
	svc := c.Services
	kvList := KVList{}
	for key, val := range svc {
		kvList = append(kvList, KV{
			Key:   key,
			Value: val,
		})
	}
	sort.Sort(kvList)

	results := make([]string, 0, len(svc)*2)
	results = append(results, "```")
	results = append(results, i18n.Message("[AWS Estimate Costs] %s", endTime.Format("2006-01-02")))
	results = append(results, fmt.Sprintf("- Total:\t$%.2f", c.Total))
	results = append(results, "------------------------")
	for _, kv := range kvList {
		results = append(results, fmt.Sprintf("- %s:\t$%.2f", kv.Key, kv.Value))
	}
	results = append(results, fmt.Sprintf("- (Other):\t$%.2f", c.Other))
	results = append(results, "```")
	return strings.Join(results, "\n")
}
