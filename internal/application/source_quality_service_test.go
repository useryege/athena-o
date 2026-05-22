package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSourceQualityAnalyzer struct {
	report string
	err    error
}

func (f fakeSourceQualityAnalyzer) AnalyzeContractSource(_ context.Context, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.report, nil
}

func TestAnalyzeContractSourceQualityNotConfigured(t *testing.T) {
	service := &Service{}

	_, err := service.AnalyzeContractSourceQuality(context.Background(), "contract A {}")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want %v, err = %v", status.Code(err), codes.FailedPrecondition, err)
	}
}

func TestAnalyzeContractSourceQuality(t *testing.T) {
	service := &Service{sourceQualityAnalyzer: fakeSourceQualityAnalyzer{report: "report"}}

	report, err := service.AnalyzeContractSourceQuality(context.Background(), "contract A {}")
	if err != nil {
		t.Fatalf("AnalyzeContractSourceQuality: %v", err)
	}
	if report != "report" {
		t.Fatalf("report = %q, want report", report)
	}
}

func TestAnalyzeContractSourceQualityError(t *testing.T) {
	service := &Service{sourceQualityAnalyzer: fakeSourceQualityAnalyzer{err: errors.New("failed")}}

	_, err := service.AnalyzeContractSourceQuality(context.Background(), "contract A {}")
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("error = %v, want analyzer error", err)
	}
}
