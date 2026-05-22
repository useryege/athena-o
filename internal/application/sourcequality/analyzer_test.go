package sourcequality

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/useryege/athena/util/deepseek"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeDeepSeekClient struct {
	request deepseek.ChatCompletionRequest
	content string
	err     error
	calls   int
}

func (f *fakeDeepSeekClient) Ping(context.Context) error {
	return nil
}

func (f *fakeDeepSeekClient) CreateChatCompletion(_ context.Context, request deepseek.ChatCompletionRequest) (*deepseek.ChatCompletionResponse, error) {
	f.calls++
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	return &deepseek.ChatCompletionResponse{Content: f.content}, nil
}

func TestAnalyzeContractSource(t *testing.T) {
	fake := &fakeDeepSeekClient{content: "## 质检报告"}
	temperature := 0.3
	analyzer := NewAnalyzer(fake, Options{
		Model:       "deepseek-v4-pro",
		MaxTokens:   2048,
		Temperature: &temperature,
	})

	report, err := analyzer.AnalyzeContractSource(context.Background(), "  contract A {}  ")
	if err != nil {
		t.Fatalf("AnalyzeContractSource: %v", err)
	}
	if report != "## 质检报告" {
		t.Fatalf("report = %q", report)
	}
	if fake.calls != 1 {
		t.Fatalf("calls = %d, want 1", fake.calls)
	}
	if fake.request.Model != "deepseek-v4-pro" {
		t.Fatalf("model = %q", fake.request.Model)
	}
	if fake.request.MaxTokens != 2048 {
		t.Fatalf("max tokens = %d", fake.request.MaxTokens)
	}
	if fake.request.Temperature == nil || *fake.request.Temperature != temperature {
		t.Fatalf("temperature = %v", fake.request.Temperature)
	}
	if len(fake.request.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(fake.request.Messages))
	}
	if !strings.Contains(fake.request.Messages[0].Content, "中文 Markdown") {
		t.Fatalf("system prompt missing report format: %q", fake.request.Messages[0].Content)
	}
	if !strings.Contains(fake.request.Messages[0].Content, "权限/Owner 风险") {
		t.Fatalf("system prompt missing risk coverage: %q", fake.request.Messages[0].Content)
	}
	if !strings.Contains(fake.request.Messages[1].Content, "contract A {}") {
		t.Fatalf("user prompt missing source: %q", fake.request.Messages[1].Content)
	}
	if strings.Contains(fake.request.Messages[1].Content, "  contract A {}  ") {
		t.Fatalf("user prompt contains untrimmed source: %q", fake.request.Messages[1].Content)
	}
}

func TestAnalyzeContractSourceEmptySource(t *testing.T) {
	fake := &fakeDeepSeekClient{}
	analyzer := NewAnalyzer(fake, Options{})

	_, err := analyzer.AnalyzeContractSource(context.Background(), "   ")
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want %v, err = %v", status.Code(err), codes.InvalidArgument, err)
	}
	if fake.calls != 0 {
		t.Fatalf("calls = %d, want 0", fake.calls)
	}
}

func TestAnalyzeContractSourceMissingClient(t *testing.T) {
	analyzer := NewAnalyzer(nil, Options{})

	_, err := analyzer.AnalyzeContractSource(context.Background(), "contract A {}")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want %v, err = %v", status.Code(err), codes.FailedPrecondition, err)
	}
}

func TestAnalyzeContractSourceDeepSeekError(t *testing.T) {
	fake := &fakeDeepSeekClient{err: errors.New("deepseek down")}
	analyzer := NewAnalyzer(fake, Options{})

	_, err := analyzer.AnalyzeContractSource(context.Background(), "contract A {}")
	if err == nil || !strings.Contains(err.Error(), "failed to analyze contract source quality") || !strings.Contains(err.Error(), "deepseek down") {
		t.Fatalf("error = %v, want wrapped DeepSeek error", err)
	}
}
