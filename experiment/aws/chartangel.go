package aws

import (
	"os"

	"github.com/evalphobia/httpwrapper/request"
)

var defaultChartURL = os.Getenv("CHART_ANGEL_ENDPOINT")

func canCreateChart(url string) bool {
	switch {
	case url != "",
		defaultChartURL != "":
		return true
	}
	return false
}

func getChartURL(url string) string {
	if url != "" {
		return url
	}
	return defaultChartURL
}

func createChartURL(endpoint, title string, list Datapoints) (string, error) {
	params := make(map[string]interface{})
	params["title"] = title
	params["label_x"] = "time"
	params["label_y"] = "value"
	params["type"] = "line"

	data := make(map[string]map[string]interface{})
	for _, dp := range list {
		category := dp.MetricName
		catData, ok := data[category]
		if !ok {
			catData = make(map[string]interface{})
			data[category] = catData
		}
		catData[dp.Time.Format("2006-01-02 15:04:05")] = dp.Value
		data[category] = catData
	}
	params["data"] = data

	chartResp, err := request.POST(getChartURL(endpoint), request.Option{
		Payload:     params,
		PayloadType: request.PayloadTypeJSON,
	})
	if err != nil {
		return "", err
	}
	err = chartResp.HasStatusCodeError()
	if err != nil {
		return "", err
	}

	var respMap map[string]interface{}
	err = chartResp.Response.JSON(&respMap)
	if err != nil {
		return "", err
	}

	return respMap["html_url"].(string), nil
}
