package tradersync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
)

// Fixed public field directory from the approved 34-resource contract.
// Do not derive these keys from the mapper or public DTO tags under test.
var gatewayFieldDirectory = map[string]string{
	"FieldEvidence":                      "availability reasonCode source queriedAt",
	"StringField":                        "evidence value",
	"DecimalField":                       "evidence value",
	"BoolField":                          "evidence value",
	"TimeField":                          "evidence value",
	"CurvePoint":                         "t p",
	"Curve":                              "evidence points",
	"PnLView":                            "period amount curve interval fidelity referenceTime timezone",
	"ResolvedTarget":                     "wallet canonicalProfileURL avatar displayName verified joinedAt positionValue largestWin predictions pnl defaultPeriod confirmationToken expiresAt usageNotice savedNote existingSubscription quota",
	"TargetNote":                         "wallet note revision",
	"Quota":                              "used limit",
	"ExistingSubscription":               "id status revision",
	"TargetDisplay":                      "displayName avatar profileURL",
	"Subscription":                       "id wallet status revision generation note noteRevision createdAt updatedAt pausedAt cancelledAt permissionDisabledAt currentInterval observation bindingStatus queueNotice queueCounts targetDisplay",
	"Interval":                           "effectiveAt endedAt generation epoch",
	"Observation":                        "state reason lastReliableAt latestInterruption interruptionCount",
	"Interruption":                       "start end recoveredAt reason uncertainty possibleMissing",
	"HistoryEntry":                       "id kind sortAt interval interruption",
	"Activity":                           "id subscriptionId sourceRecordId wallet side positionId collateralRaw sharesRaw feeRaw collateralSymbol collateralDecimals sharesDecimals priceNumerator priceDenominator priceEvidence sourceVersion settledAt receivedAt recordedAt publicTimeEvidence metadata noteSnapshot notificationMode notificationReason delivery summaryProgress targetDisplaySnapshot finalityAnomaly sourceLocation",
	"SourceLocation":                     "chainId exchangeAddress transactionHash blockHash blockNumber logIndex",
	"FinalityAnomaly":                    "reason detectedAt publishedBlockHash conflictingBlockHash",
	"MarketRef":                          "evidence id title url conditionId positionId outcome",
	"ComboLeg":                           "positionId market",
	"TradeMetadata":                      "market legsEvidence legs relationship",
	"Delivery":                           "id status reason authorizedAt startedAt resultAt messageId attemptCount latestAttempt",
	"Attempt":                            "index authorizedAt startedAt resultAt status reason",
	"StatusCounts":                       "total pending sending sent failed unknown cancelled",
	"SummaryProgress":                    "phase reason batchId relatedPartCounts batchPartCounts oldestAt firstStartedAt",
	"TargetCount":                        "wallet count",
	"SummaryBatch":                       "id oldestAt settledFrom settledTo recordedFrom recordedTo firstStartedAt activityCount targetCounts partCounts asOf",
	"SummaryPart":                        "id index total delivery associatedActivityCount",
	"SubscriptionSummary":                "subscriptionId accountId username email wallet status createdAt updatedAt pausedAt cancelledAt permissionDisabledAt observation activityCount associatedDeliveryCounts asOf",
	"RuntimeMetric":                      "name value unit kind windowStart windowEnd serviceEpoch",
	"RuntimeStatus":                      "collectorConnected collectorEpoch filterRevision metrics asOf",
	"PageInfo":                           "nextCursor",
	"ActivityPageInfo":                   "nextCursor refreshCursor snapshot asOf hasNewer",
	"ResolveTargetResponse":              "target",
	"CreateSubscriptionResponse":         "subscription",
	"ListSubscriptionsResponse":          "subscriptions page quota asOf",
	"GetSubscriptionResponse":            "subscription",
	"PauseSubscriptionResponse":          "subscription",
	"ResumeSubscriptionResponse":         "subscription",
	"CancelSubscriptionResponse":         "subscription",
	"UpdateTargetNoteResponse":           "note",
	"ListActivitiesResponse":             "activities page",
	"GetActivityResponse":                "activity",
	"ListSubscriptionHistoryResponse":    "entries page asOf",
	"GetSummaryBatchResponse":            "batch",
	"ListSummaryPartsResponse":           "parts page asOf",
	"ListSubscriptionSummariesResponse":  "summaries page asOf",
	"GetSubscriptionSummaryResponse":     "summary",
	"GetTraderSyncRuntimeStatusResponse": "status",
}

// Expectations come from the internal wire fixture, without calling a facade
// mapper or serializing a public DTO. Complete fixtures exercise every field;
// comparing whole JSON objects also catches unexpected public fields.
func wireJSON(t *testing.T, v reflect.Value, seen map[string]bool) any {
	t.Helper()
	if v.Kind() == reflect.Pointer {
		require.False(t, v.IsNil())
		return wireJSON(t, v.Elem(), seen)
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Int32:
		return float64(v.Int())
	case reflect.Slice:
		a := make([]any, v.Len())
		for i := range a {
			a[i] = wireJSON(t, v.Index(i), seen)
		}
		return a
	case reflect.Struct:
		name := v.Type().Name()
		switch name {
		case "StringValue", "BoolValue":
			return wireJSON(t, v.FieldByName("Value"), seen)
		case "CurvePointList", "ComboLegList":
			return wireJSON(t, v.FieldByName("Items"), seen)
		}
		seen[name] = true
		result := map[string]any{}
		for i := 0; i < v.NumField(); i++ {
			sf := v.Type().Field(i)
			key := wireName(sf)
			if key == "" {
				continue
			}
			switch key {
			case "canonical_profile_url":
				key = "canonicalProfileURL"
			case "profile_url":
				key = "profileURL"
			case "pn_l":
				key = "pnl"
			default:
				words := strings.Split(key, "_")
				key = words[0]
				for _, word := range words[1:] {
					key += strings.ToUpper(word[:1]) + word[1:]
				}
			}
			result[key] = wireJSON(t, v.Field(i), seen)
		}
		names, ok := gatewayFieldDirectory[name]
		require.True(t, ok, "missing field directory for %s", name)
		keys := make([]string, 0, len(result))
		for key := range result {
			keys = append(keys, key)
		}
		require.ElementsMatch(t, strings.Fields(names), keys, "wire/public field directory changed for %s", name)
		return result
	}
	t.Fatalf("unhandled wire value %s", v.Type())
	return nil
}

func gatewayJSON(t *testing.T, remote *recordingInternal, method, path, body string, wantStatus int) map[string]any {
	t.Helper()
	h := contractGateway(t, New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil }))
	request, err := http.NewRequest(method, h.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := h.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, wantStatus, response.StatusCode, string(raw))
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	return got
}

func TestGatewayAll16RPCsCompareEveryResponseField(t *testing.T) {
	const base = "/api/v1/trader-sync"
	cases := []struct {
		name, method, path, body string
		response                 any
		request                  any
	}{
		{"ResolveTarget", "POST", base + "/targets:resolve", `{"input":" unchanged "}`, completeWire[trpc.ResolveTargetResponse](), &trpc.ResolveTargetRequest{Actor: facadeActor, Input: " unchanged "}},
		{"CreateSubscription", "POST", base + "/subscriptions", `{"confirmationToken":"opaque","requestId":"create","note":{"value":""}}`, completeWire[trpc.CreateSubscriptionResponse](), &trpc.CreateSubscriptionRequest{Actor: facadeActor, ConfirmationToken: "opaque", RequestId: "create", Note: &trpc.NoteInput{Value: ""}}},
		{"ListSubscriptions", "GET", base + "/subscriptions?page.page_size=17&page.cursor=opaque%2B%2F%3D&view=history&state=paused", "", completeWire[trpc.ListSubscriptionsResponse](), &trpc.ListSubscriptionsRequest{Actor: facadeActor, Page: &trpc.PageInput{PageSize: 17, Cursor: "opaque+/="}, View: "history", State: "paused"}},
		{"GetSubscription", "GET", base + "/subscriptions/sub", "", completeWire[trpc.GetSubscriptionResponse](), &trpc.GetSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub"}},
		{"PauseSubscription", "POST", base + "/subscriptions/sub:pause", `{"expectedRevision":"18446744073709551615","requestId":"pause"}`, completeWire[trpc.PauseSubscriptionResponse](), &trpc.PauseSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub", ExpectedRevision: ^uint64(0), RequestId: "pause"}},
		{"ResumeSubscription", "POST", base + "/subscriptions/sub:resume", `{"expectedRevision":"9007199254740993","requestId":"resume"}`, completeWire[trpc.ResumeSubscriptionResponse](), &trpc.ResumeSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub", ExpectedRevision: 9007199254740993, RequestId: "resume"}},
		{"CancelSubscription", "POST", base + "/subscriptions/sub:cancel", `{"expectedRevision":"9007199254740995","requestId":"cancel"}`, completeWire[trpc.CancelSubscriptionResponse](), &trpc.CancelSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub", ExpectedRevision: 9007199254740995, RequestId: "cancel"}},
		{"UpdateTargetNote", "PATCH", base + "/targets/wallet/note", `{"note":"更新","expectedRevision":"9007199254740997","requestId":"note"}`, completeWire[trpc.UpdateTargetNoteResponse](), &trpc.UpdateTargetNoteRequest{Actor: facadeActor, Wallet: "wallet", Note: "更新", ExpectedRevision: 9007199254740997, RequestId: "note"}},
		{"ListActivities", "GET", base + "/activities?page.page_size=19&page.cursor=next&subscription_id=sub&from=from&to=to&summary_batch_id=batch&refresh_cursor=refresh", "", completeWire[trpc.ListActivitiesResponse](), &trpc.ListActivitiesRequest{Actor: facadeActor, Page: &trpc.PageInput{PageSize: 19, Cursor: "next"}, SubscriptionId: "sub", From: "from", To: "to", SummaryBatchId: "batch", RefreshCursor: "refresh"}},
		{"GetActivity", "GET", base + "/activities/9007199254740993", "", completeWire[trpc.GetActivityResponse](), &trpc.GetActivityRequest{Actor: facadeActor, ActivityId: "9007199254740993"}},
		{"ListSubscriptionHistory", "GET", base + "/subscriptions/sub/history?page.page_size=23&page.cursor=history", "", completeWire[trpc.ListSubscriptionHistoryResponse](), &trpc.ListSubscriptionHistoryRequest{Actor: facadeActor, SubscriptionId: "sub", Page: &trpc.PageInput{PageSize: 23, Cursor: "history"}}},
		{"GetSummaryBatch", "GET", base + "/summaries/9007199254740999", "", completeWire[trpc.GetSummaryBatchResponse](), &trpc.GetSummaryBatchRequest{Actor: facadeActor, BatchId: "9007199254740999"}},
		{"ListSummaryParts", "GET", base + "/summaries/batch/parts?activity_id=activity&page.page_size=29&page.cursor=parts", "", completeWire[trpc.ListSummaryPartsResponse](), &trpc.ListSummaryPartsRequest{Actor: facadeActor, BatchId: "batch", ActivityId: "activity", Page: &trpc.PageInput{PageSize: 29, Cursor: "parts"}}},
		{"ListSubscriptionSummaries", "GET", "/api/v1/admin/trader-sync/subscriptions?page.page_size=31&page.cursor=admin&account_id=filter-owner&state=cancelled&wallet=wallet&include_cancelled=true", "", completeWire[trpc.ListSubscriptionSummariesResponse](), &trpc.ListSubscriptionSummariesRequest{Actor: facadeActor, Page: &trpc.PageInput{PageSize: 31, Cursor: "admin"}, AccountId: "filter-owner", State: "cancelled", Wallet: "wallet", IncludeCancelled: true}},
		{"GetSubscriptionSummary", "GET", "/api/v1/admin/trader-sync/subscriptions/sub", "", completeWire[trpc.GetSubscriptionSummaryResponse](), &trpc.GetSubscriptionSummaryRequest{Actor: facadeActor, SubscriptionId: "sub"}},
		{"GetTraderSyncRuntimeStatus", "GET", "/api/v1/admin/trader-sync/status", "", completeWire[trpc.GetTraderSyncRuntimeStatusResponse](), &trpc.GetTraderSyncRuntimeStatusRequest{Actor: facadeActor}},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fillSentinels(reflect.ValueOf(tc.response), tc.name)
			want := wireJSON(t, reflect.ValueOf(tc.response), seen)
			remote := &recordingInternal{response: tc.response}
			got := gatewayJSON(t, remote, tc.method, tc.path, tc.body, http.StatusOK)
			require.Equal(t, want, got, "final JSON changed after internal and public protobuf hops")
			require.Equal(t, int32(1), remote.calls.Load(), "exactly one internal handler must run")
			require.Equal(t, tc.request, remote.request, "actual internal server must receive the complete trusted request")

			// Every RPC has at least one required response object. Exercise the
			// consumer-visible 503 instead of stopping at a direct mapper assertion.
			missing := &recordingInternal{response: reflect.New(reflect.TypeOf(tc.response).Elem()).Interface()}
			gatewayJSON(t, missing, tc.method, tc.path, tc.body, http.StatusServiceUnavailable)
			require.Equal(t, int32(1), missing.calls.Load())
		})
	}
	// This explicit directory prevents quietly dropping an entire resource fixture.
	for _, name := range strings.Fields("FieldEvidence StringField DecimalField BoolField TimeField CurvePoint Curve PnLView ResolvedTarget TargetNote Quota ExistingSubscription TargetDisplay Subscription Interval Observation Interruption HistoryEntry Activity SourceLocation FinalityAnomaly MarketRef ComboLeg TradeMetadata Delivery Attempt StatusCounts SummaryProgress TargetCount SummaryBatch SummaryPart SubscriptionSummary RuntimeMetric RuntimeStatus PageInfo ActivityPageInfo") {
		require.True(t, seen[name], "resource %s absent from final JSON coverage", name)
	}
	require.Len(t, cases, 16)
	t.Logf("compared 16 actual internal RPC requests and complete JSON for %d wire message types", len(seen))
}

func TestGatewayBoundaryValuesAfterBothProtobufHops(t *testing.T) {
	t.Run("resolved false and six ordered PnL periods", func(t *testing.T) {
		target := completeWire[trpc.ResolvedTarget]()
		target.Verified.Value = &trpc.BoolValue{Value: false}
		target.JoinedAt = &trpc.TimeField{Evidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "invalid_time"}}
		target.PositionValue.Value = &trpc.StringValue{Value: "90071992547409931234567890.000000000000000001"}
		target.PnL = nil
		for _, period := range []string{"1D", "1W", "1M", "1Y", "YTD", "ALL"} {
			p := completeWire[trpc.PnLView]()
			p.Period = period
			p.Curve.Points = &trpc.CurvePointList{Items: []*trpc.CurvePoint{{T: "1725148800", P: "0"}, {T: "1725148800", P: "1.25"}}}
			target.PnL = append(target.PnL, p)
		}
		remote := &recordingInternal{response: &trpc.ResolveTargetResponse{Target: target}}
		got := gatewayJSON(t, remote, "POST", "/api/v1/trader-sync/targets:resolve", `{"input":"wallet"}`, 200)["target"].(map[string]any)
		require.Equal(t, false, got["verified"].(map[string]any)["value"])
		require.NotContains(t, got["joinedAt"], "value")
		require.Equal(t, "invalid_time", got["joinedAt"].(map[string]any)["evidence"].(map[string]any)["reasonCode"])
		require.Equal(t, "90071992547409931234567890.000000000000000001", got["positionValue"].(map[string]any)["value"])
		periods := got["pnl"].([]any)
		require.Len(t, periods, 6)
		for i, period := range []string{"1D", "1W", "1M", "1Y", "YTD", "ALL"} {
			p := periods[i].(map[string]any)
			require.Equal(t, period, p["period"])
			require.Equal(t, []any{map[string]any{"t": "1725148800", "p": "0"}, map[string]any{"t": "1725148800", "p": "1.25"}}, p["curve"].(map[string]any)["points"])
		}
		target.PnL[0].Curve = &trpc.Curve{Evidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "query_failed"}, Points: nil}
		target.PnL[1].Curve = &trpc.Curve{Evidence: &trpc.FieldEvidence{Availability: "available"}, Points: &trpc.CurvePointList{}}
		target.Verified = &trpc.BoolField{Evidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "invalid_boolean"}}
		got = gatewayJSON(t, remote, "POST", "/api/v1/trader-sync/targets:resolve", `{"input":"wallet"}`, 200)["target"].(map[string]any)
		require.NotContains(t, got["verified"], "value")
		require.Equal(t, "invalid_boolean", got["verified"].(map[string]any)["evidence"].(map[string]any)["reasonCode"])
		periods = got["pnl"].([]any)
		for i, wantAvailability := range []string{"unavailable", "available"} {
			curve := periods[i].(map[string]any)["curve"].(map[string]any)
			require.Contains(t, curve, "points")
			require.Nil(t, curve["points"])
			require.Equal(t, wantAvailability, curve["evidence"].(map[string]any)["availability"])
		}
	})
	t.Run("max revision", func(t *testing.T) {
		sub := completeWire[trpc.Subscription]()
		sub.Revision = ^uint64(0)
		got := gatewayJSON(t, &recordingInternal{response: &trpc.GetSubscriptionResponse{Subscription: sub}}, "GET", "/api/v1/trader-sync/subscriptions/sub", "", 200)
		require.Equal(t, "18446744073709551615", got["subscription"].(map[string]any)["revision"])
	})
	for _, shape := range []string{"unknown", "empty", "partial"} {
		t.Run("activity legs "+shape, func(t *testing.T) {
			activity := completeWire[trpc.Activity]()
			activity.Id = "9007199254740993"
			activity.PriceNumerator = "0"
			activity.CollateralRaw = "90071992547409931234567890"
			activity.SettledAt = ""
			activity.Delivery = &trpc.Delivery{Status: "unknown", MessageId: nil, AuthorizedAt: nil, StartedAt: nil, ResultAt: nil}
			switch shape {
			case "unknown":
				activity.Metadata.Legs = nil
				activity.Metadata.LegsEvidence = &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "query_failed"}
			case "empty":
				activity.Metadata.Legs = &trpc.ComboLegList{}
				activity.Metadata.LegsEvidence = &trpc.FieldEvidence{Availability: "available"}
			case "partial":
				activity.Metadata.Legs = &trpc.ComboLegList{Items: []*trpc.ComboLeg{{PositionId: "9007199254740993123", Market: &trpc.MarketRef{Evidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "not_found"}}}}}
			}
			cursor := "original+/=signed.cursor"
			got := gatewayJSON(t, &recordingInternal{response: &trpc.ListActivitiesResponse{Activities: []*trpc.Activity{activity}, Page: &trpc.ActivityPageInfo{NextCursor: cursor}}}, "GET", "/api/v1/trader-sync/activities", "", 200)
			a := got["activities"].([]any)[0].(map[string]any)
			require.Equal(t, "9007199254740993", a["id"])
			require.Equal(t, "0", a["priceNumerator"])
			require.Equal(t, "90071992547409931234567890", a["collateralRaw"])
			require.Equal(t, "", a["settledAt"])
			require.Equal(t, cursor, got["page"].(map[string]any)["nextCursor"])
			for _, key := range []string{"authorizedAt", "startedAt", "resultAt", "messageId"} {
				require.NotContains(t, a["delivery"], key)
			}
			metadata := a["metadata"].(map[string]any)
			if shape == "partial" {
				legs := metadata["legs"].([]any)
				require.Len(t, legs, 1)
				require.Equal(t, "9007199254740993123", legs[0].(map[string]any)["positionId"])
				require.Equal(t, "not_found", legs[0].(map[string]any)["market"].(map[string]any)["evidence"].(map[string]any)["reasonCode"])
			} else {
				require.Contains(t, metadata, "legs")
				require.Nil(t, metadata["legs"], "public protobuf encodes both empty repeated forms as null")
				require.Equal(t, activity.Metadata.LegsEvidence.Availability, metadata["legsEvidence"].(map[string]any)["availability"])
			}
		})
	}
	t.Run("empty repeated and required page", func(t *testing.T) {
		response := &trpc.ListActivitiesResponse{Activities: []*trpc.Activity{}, Page: &trpc.ActivityPageInfo{}}
		got := gatewayJSON(t, &recordingInternal{response: response}, "GET", "/api/v1/trader-sync/activities", "", 200)
		require.NotContains(t, got, "activities")
		require.Equal(t, map[string]any{"hasNewer": false}, got["page"])
		response.Page = nil
		gatewayJSON(t, &recordingInternal{response: response}, "GET", "/api/v1/trader-sync/activities", "", 503)
	})
}
