package langchain

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"

	"github.com/eure/bobo/command"
	"github.com/evalphobia/bobo-experiment/i18n"
)

const (
	inputKey  = "input"
	outputKey = "output"
)

var _ command.CommandTemplate = &ChatGPTCommand{}

type ChatGPTCommand struct {
	Command string

	openaiOnce  sync.Once
	openaiAgent agents.Executor
}

func (t *ChatGPTCommand) GetMentionCommand() string {
	return t.Command
}

func (*ChatGPTCommand) GetHelp() string {
	return "Send text to OepnAI ChatGPT and get answer"
}

func (*ChatGPTCommand) HasHelp() bool {
	return true
}

func (*ChatGPTCommand) GetRegexp() *regexp.Regexp {
	return nil
}

func (t *ChatGPTCommand) Exec(d command.CommandData) {
	c := t.runChat(d)
	c.Exec()
}

func (t *ChatGPTCommand) runChat(d command.CommandData) command.Command {
	c := command.Command{}

	text := strings.TrimSpace(d.TextOther)
	command.NewReplyEngineTask(d.Engine, d.Channel, "...").Run()
	agent, err := t.getOrCreateClient()
	if err != nil {
		task := command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf(i18n.Message("error: %s"), err.Error()))
		c.Add(task)
		return c
	}

	resp, err := chains.Call(context.Background(), agent, map[string]any{
		inputKey: text,
	})

	answer := ""
	switch {
	case err != nil:
		errMsg := err.Error()
		if !strings.Contains(errMsg, "unable to parse agent output: ") {
			task := command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf(i18n.Message("error: %s"), err.Error()))
			c.Add(task)
			return c
		}
		answer = strings.ReplaceAll(errMsg, "unable to parse agent output: ", "")
	default:
		text, ok := resp[outputKey].(string)
		if !ok {
			task := command.NewReplyEngineTask(d.Engine, d.Channel, fmt.Sprintf(i18n.Message("Unexpected response: %+v"), resp))
			c.Add(task)
			return c
		}
		answer = text
	}
	task := command.NewReplyEngineTask(d.Engine, d.Channel, answer)
	c.Add(task)
	return c
}

func (t *ChatGPTCommand) getOrCreateClient() (agents.Executor, error) {
	var err error
	t.openaiOnce.Do(func() {
		llm, e := openai.NewChat()
		if e != nil {
			err = e
			return
		}

		chain := chains.NewLLMChain(llm, prompts.PromptTemplate{
			Template: `Assistant is a large language model trained by OpenAI.
Assistant is designed to be able to assist with a wide range of tasks, from answering simple questions to providing in-depth explanations and discussions on a wide range of topics. As a language model, Assistant is able to generate human-like text based on the input it receives, allowing it to engage in natural-sounding conversations and provide responses that are coherent and relevant to the topic at hand.
Assistant is constantly learning and improving, and its capabilities are constantly evolving. It is able to process and understand large amounts of text, and can use this knowledge to provide accurate and informative responses to a wide range of questions. Additionally, Assistant is able to generate its own text based on the input it receives, allowing it to engage in discussions and provide explanations and descriptions on a wide range of topics.
Overall, Assistant is a powerful tool that can help with a wide range of tasks and provide valuable insights and information on a wide range of topics. Whether you need help with a specific question or just want to have a conversation about a particular topic, Assistant is here to assist.

When you have a response to say to the Human, you MUST use the format:
AI: [your response here]
` + i18n.Message(`Begin!
{{ if .history }}
Previous conversation history:
{{.history}}
{{ end }}

New input: {{.input}}

{{ if .agent_scratchpad }}
Thought:{{.agent_scratchpad}}
{{ end }}`),
			TemplateFormat: prompts.TemplateFormatGoTemplate,
			InputVariables: []string{inputKey},
			PartialVariables: map[string]any{
				"agent_scratchpad": "",
				"history":          "",
			},
		})
		chain.Memory = &memory.Buffer{
			ChatHistory:    memory.NewChatMessageHistory(),
			ReturnMessages: false,
			InputKey:       inputKey,
			OutputKey:      "",
			HumanPrefix:    "Human",
			AIPrefix:       "",
			MemoryKey:      "history",
		}

		a := agents.ConversationalAgent{
			Chain:     *chain,
			OutputKey: outputKey,
		}
		t.openaiAgent = agents.NewExecutor(&a, nil)
	})
	return t.openaiAgent, err
}
