package i18n

import (
	"os"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var defaultPrinter = message.NewPrinter(language.English)

func init() {
	for _, lang := range langs {
		for _, l := range lang.List {
			message.SetString(l.Lang, lang.Key, l.Value)
		}
	}
	setDefaultLanguage()
}

func setDefaultLanguage() {
	var tag language.Tag
	switch os.Getenv("BOBO_LANG") {
	case "ja":
		tag = language.Japanese
	default:
		tag = language.English
	}
	defaultPrinter = message.NewPrinter(tag)
}

func Message(key string, args ...interface{}) string {
	return defaultPrinter.Sprintf(key, args...)
}

func CommaNumber(n int) string {
	return defaultPrinter.Sprintf("%d", n)
}

var langs = []translation{
	// AWSCost
	{Key: "Invalid date format: [%s]", List: []translationData{
		{language.Japanese, "日付の指定が不正です: [%s]"},
	}},
	{Key: "Getting costs on [%s]...", List: []translationData{
		{language.Japanese, "[%s] のコストを取得中..."},
	}},
	{Key: "[AWS Estimate Costs] %s", List: []translationData{
		{language.Japanese, "[AWS概算コスト] %s"},
	}},
	// Merge
	{Key: "No!", List: []translationData{
		{language.Japanese, "だが断る。"},
	}},
	{Key: "Set Two URLs", List: []translationData{
		{language.Japanese, "URLを2つセットしてください"},
	}},
	{Key: "Invalid URL. It must begin with [http/https]", List: []translationData{
		{language.Japanese, "[http/https] で始まるURLを指定してください"},
	}},
	{Key: "Merging...", List: []translationData{
		{language.Japanese, "マージ中..."},
	}},

	// Calendar
	{Key: "[AllDay] [%s - %s]", List: []translationData{
		{language.Japanese, "【終日】[%s - %s]"},
	}},
	{Key: "Getting events of [%s] ...", List: []translationData{
		{language.Japanese, "[%s] の予定を確認中..."},
	}},
	{Key: "Target format is invalid, use @mention or correct email address.", List: []translationData{
		{language.Japanese, "@メンション か 正しいメールアドレス を指定してください"},
	}},

	// Where
	{Key: "Getting location of [%s] ...", List: []translationData{
		{language.Japanese, "[%s] の場所を確認中..."},
	}},
	{Key: "Somewhere around there", List: []translationData{
		{language.Japanese, "そのへんにいそうです"},
	}},
	{Key: "Doing [%s] in somewhere around there (%s - %s)", List: []translationData{
		{language.Japanese, "どこかで %s してそうです (%s - %s)"},
	}},
	{Key: "[%s] | Doing [%s] at [%s] (%s - %s)", List: []translationData{
		{language.Japanese, "【%s】 | %[3]s で %[2]s してそうです (%s - %s)"},
	}},
	{Key: "prev", List: []translationData{
		{language.Japanese, "前"},
	}},
	{Key: "next", List: []translationData{
		{language.Japanese, "次"},
	}},
	{Key: "now", List: []translationData{
		{language.Japanese, "今"},
	}},
	{Key: "all-day", List: []translationData{
		{language.Japanese, "終日"},
	}},
	{Key: "[AllDay] [%s - %s]", List: []translationData{
		{language.Japanese, "【終日】[%s - %s]"},
	}},
}

type translation struct {
	Key  string
	List []translationData
}

type translationData struct {
	Lang  language.Tag
	Value string
}
