package sourcequality

import (
	"context"
	"fmt"
	"strings"

	"github.com/useryege/athena/util/deepseek"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Analyzer interface {
	AnalyzeContractSource(ctx context.Context, systemPrompt string, sourceCode string) (string, error)
}

type Options struct {
	Model       string
	MaxTokens   int
	Temperature *float64
}

type analyzerImpl struct {
	client  deepseek.Client
	options Options
}

func NewAnalyzer(client deepseek.Client, options Options) Analyzer {
	return &analyzerImpl{
		client:  client,
		options: options,
	}
}

func (a *analyzerImpl) AnalyzeContractSource(ctx context.Context, systemPrompt string, sourceCode string) (string, error) {
	if a == nil || a.client == nil {
		return "", status.Error(codes.FailedPrecondition, "DeepSeek analyzer is not configured")
	}
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt == "" {
		return "", status.Error(codes.InvalidArgument, "system prompt is empty")
	}
	sourceCode = strings.TrimSpace(sourceCode)
	if sourceCode == "" {
		return "", status.Error(codes.InvalidArgument, "contract source code is empty")
	}
	request := deepseek.ChatCompletionRequest{
		Model:       a.options.Model,
		MaxTokens:   a.options.MaxTokens,
		Temperature: a.options.Temperature,
		Messages: []deepseek.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildUserPrompt(sourceCode)},
		},
	}
	response, err := a.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to analyze contract source quality with DeepSeek: %w", err)
	}
	return response.Content, nil
}

func buildUserPrompt(sourceCode string) string {
	return "请分析下面的 Solidity 合同源码，并输出质检报告：\n\n```solidity\n" + sourceCode + "\n```"
}

const DefaultSystemPrompt = `你是资深智能合约安全与质检工程师。请对用户提供的 Solidity 合同源码进行静态审阅，输出中文 Markdown 纯文本质检报告。

报告必须覆盖：
1. 合约概览
2. 关键风险
3. 权限/Owner 风险
4. 交易/转账限制
5. 税费/黑名单/暂停/代理升级风险
6. 可疑函数或字段
7. 结论建议

要求：
- 不要输出 JSON。
- 不要编造源码中不存在的函数、字段或结论。
- 对无法确认的问题明确标注“需人工复核”。
- 按高、中、低风险组织发现，说明证据和影响。`
