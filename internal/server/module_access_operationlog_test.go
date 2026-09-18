package server

import (
	"strings"
	"testing"

	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
)

func TestOperationLogGatewayFlattensNullableScalars(t *testing.T) {
	value := &pb.ListOperationLogsResponse{Items: []*pb.OperationLogSummary{{
		StartedAt:        &pb.NullableString{Value: "2026-09-18T00:00:00Z"},
		ActorAccountId:   &pb.NullableString{Value: "account-1"},
		IdentityVerified: &pb.NullableBool{Value: false},
		DurationMs:       &pb.NullableString{Value: "0"},
	}}}
	raw, err := (&moduleAccessJSONMarshaler{}).Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"startedAt":"2026-09-18T00:00:00Z"`, `"actorAccountId":"account-1"`, `"identityVerified":false`, `"durationMs":"0"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("json=%s missing %s", text, want)
		}
	}

	detail := &pb.GetOperationLogResponse{Item: &pb.OperationLogDetail{
		Effect:   []string{"WALLET_UPDATED", "CACHE_PUBLISHED"},
		Protocol: &pb.OperationLogProtocolResult{GrpcCode: &pb.NullableString{Value: "OK"}},
		Source:   &pb.OperationLogSourceFacts{ProducerId: &pb.NullableString{Value: "producer-1"}},
	}}
	raw, err = (&moduleAccessJSONMarshaler{}).Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	text = string(raw)
	for _, want := range []string{`"effect":["WALLET_UPDATED","CACHE_PUBLISHED"]`, `"grpcCode":"OK"`, `"producerId":"producer-1"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("detail json=%s missing %s", text, want)
		}
	}

	resource := &pb.GetOperationLogResponse{Item: &pb.OperationLogDetail{ResourceFacts: []*pb.OperationLogResource{{Type: "wallet", Id: "wallet-1", ReferenceVerified: &pb.NullableBool{Value: false}}}}}
	raw, err = (&moduleAccessJSONMarshaler{}).Marshal(resource)
	if err != nil {
		t.Fatal(err)
	}
	if text = string(raw); !strings.Contains(text, `"referenceVerified":false`) {
		t.Fatalf("resource json=%s", text)
	}

	capture := &pb.GetOperationLogCaptureStatusResponse{Status: &pb.CaptureStatus{ObservedAt: &pb.NullableString{Value: "2026-09-18T00:00:00Z"}, InFlightEvents: &pb.NullableString{Value: "0"}}}
	raw, err = (&moduleAccessJSONMarshaler{}).Marshal(capture)
	if err != nil {
		t.Fatal(err)
	}
	text = string(raw)
	for _, want := range []string{`"observedAt":"2026-09-18T00:00:00Z"`, `"inFlightEvents":"0"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("capture json=%s missing %s", text, want)
		}
	}
}
