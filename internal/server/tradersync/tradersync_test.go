package tradersync

import (
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestPublicResolvedPreservesPresenceAndServiceEvidence(t *testing.T) {
	v := completeWire[trpc.ResolvedTarget]()
	v.Verified.Value = &trpc.BoolValue{Value: false}
	v.DisplayName.Value = &trpc.StringValue{Value: ""}
	v.PositionValue.Value = &trpc.StringValue{Value: "0"}
	v.LargestWin.Value = nil
	v.LargestWin.Evidence = &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "query_failed"}
	v.SavedNote = nil
	v.ExistingSubscription = &trpc.ExistingSubscription{Revision: math.MaxUint64}
	v.PnL = []*trpc.PnLView{completeWire[trpc.PnLView](), completeWire[trpc.PnLView](), completeWire[trpc.PnLView]()}
	v.PnL[0].Curve.Points = nil
	v.PnL[1].Curve.Points = &trpc.CurvePointList{}
	v.PnL[2].Curve.Points = &trpc.CurvePointList{Items: []*trpc.CurvePoint{{T: "9007199254740993", P: "0"}, {T: "9007199254740993", P: "1"}}}
	v.DefaultPeriod = "service-default"
	m := &responseMapper{}
	got := m.mapResolvedTarget(v)
	if m.err != nil {
		t.Fatal(m.err)
	}
	if got.Verified.Value == nil || *got.Verified.Value || got.DisplayName.Value == nil || *got.DisplayName.Value != "" || got.PositionValue.Value == nil || *got.PositionValue.Value != "0" || got.LargestWin.Value != nil || got.LargestWin.Evidence.ReasonCode != "query_failed" || got.SavedNote != nil || got.ExistingSubscription.Revision != math.MaxUint64 {
		t.Fatalf("presence or exact revision changed: %+v", got)
	}
	if got.DefaultPeriod != "service-default" || len(got.PnL) != 3 || got.PnL[0].Curve.Points != nil || got.PnL[1].Curve.Points == nil || len(got.PnL[1].Curve.Points) != 0 || got.PnL[2].Curve.Points[1].T != "9007199254740993" {
		t.Fatal("facade recomputed data or collapsed list presence", got)
	}
}

// Every scalar receives a distinct path-derived sentinel before mapping. The
// comparison uses protobuf field names, independently of the handwritten maps,
// to catch dropped or transposed fields across all 34 resource messages.
func fillSentinels(v reflect.Value, path string) {
	if v.Kind() == reflect.Pointer {
		fillSentinels(v.Elem(), path)
		return
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		sf := v.Type().Field(i)
		if !f.CanSet() || strings.HasPrefix(sf.Name, "XXX_") {
			continue
		}
		p := path + "/" + sf.Name
		switch f.Kind() {
		case reflect.String:
			f.SetString(p)
		case reflect.Bool:
			f.SetBool(true)
		case reflect.Uint64:
			f.SetUint(math.MaxUint64)
		case reflect.Int32:
			f.SetInt(int64(i + 7))
		case reflect.Pointer:
			if !f.IsNil() {
				fillSentinels(f, p)
			}
		case reflect.Slice:
			for j := 0; j < f.Len(); j++ {
				fillSentinels(f.Index(j), p)
			}
		}
	}
}
func wireName(f reflect.StructField) string {
	for _, part := range strings.Split(f.Tag.Get("protobuf"), ",") {
		if strings.HasPrefix(part, "name=") {
			return strings.TrimPrefix(part, "name=")
		}
	}
	return ""
}
func compareWireFields(t *testing.T, wire, public reflect.Value, path string) {
	t.Helper()
	if public.Kind() == reflect.Pointer {
		if public.IsNil() {
			if !wire.IsNil() {
				t.Fatalf("%s dropped presence", path)
			}
			return
		}
		public = public.Elem()
	}
	if wire.Kind() == reflect.Pointer {
		if wire.IsNil() {
			t.Fatalf("%s fabricated presence", path)
		}
		wire = wire.Elem()
	}
	if public.Kind() == reflect.Slice {
		if wire.Kind() == reflect.Struct {
			wire = wire.FieldByName("Items")
		}
		if wire.Len() != public.Len() {
			t.Fatalf("%s list count", path)
		}
		for i := 0; i < public.Len(); i++ {
			compareWireFields(t, wire.Index(i), public.Index(i), path)
		}
		return
	}
	if public.Kind() != reflect.Struct {
		if wire.Kind() == reflect.Struct {
			wire = wire.FieldByName("Value")
		}
		if !reflect.DeepEqual(wire.Interface(), public.Interface()) {
			t.Fatalf("%s got %v want %v", path, public.Interface(), wire.Interface())
		}
		return
	}
	for i := 0; i < public.NumField(); i++ {
		sf := public.Type().Field(i)
		name := wireName(sf)
		if name == "" {
			continue
		}
		found := false
		for j := 0; j < wire.NumField(); j++ {
			if wireName(wire.Type().Field(j)) == name {
				compareWireFields(t, wire.Field(j), public.Field(i), path+"/"+name)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s field %s absent", path, name)
		}
	}
}
func TestAllResponseFieldsAreCopiedExactly(t *testing.T) {
	m := &responseMapper{}
	cases := []struct {
		wire   any
		public any
	}{}
	target := completeWire[trpc.ResolvedTarget]()
	fillSentinels(reflect.ValueOf(target), "target")
	cases = append(cases, struct{ wire, public any }{target, m.mapResolvedTarget(target)})
	subscription := completeWire[trpc.Subscription]()
	fillSentinels(reflect.ValueOf(subscription), "subscription")
	cases = append(cases, struct{ wire, public any }{subscription, m.mapSubscription(subscription)})
	activity := completeWire[trpc.Activity]()
	fillSentinels(reflect.ValueOf(activity), "activity")
	cases = append(cases, struct{ wire, public any }{activity, m.mapActivity(activity)})
	history := completeWire[trpc.HistoryEntry]()
	fillSentinels(reflect.ValueOf(history), "history")
	cases = append(cases, struct{ wire, public any }{history, m.mapHistoryEntry(history)})
	batch := completeWire[trpc.SummaryBatch]()
	fillSentinels(reflect.ValueOf(batch), "batch")
	cases = append(cases, struct{ wire, public any }{batch, m.mapSummaryBatch(batch)})
	part := completeWire[trpc.SummaryPart]()
	fillSentinels(reflect.ValueOf(part), "part")
	cases = append(cases, struct{ wire, public any }{part, m.mapSummaryPart(part)})
	summary := completeWire[trpc.SubscriptionSummary]()
	fillSentinels(reflect.ValueOf(summary), "summary")
	cases = append(cases, struct{ wire, public any }{summary, m.mapSubscriptionSummary(summary)})
	runtime := completeWire[trpc.RuntimeStatus]()
	fillSentinels(reflect.ValueOf(runtime), "runtime")
	cases = append(cases, struct{ wire, public any }{runtime, m.mapRuntimeStatus(runtime)})
	if m.err != nil {
		t.Fatal(m.err)
	}
	for _, tc := range cases {
		compareWireFields(t, reflect.ValueOf(tc.wire), reflect.ValueOf(tc.public), "response")
	}
}
func TestMissingNestedRequiredObjectsRejectWholeResponse(t *testing.T) {
	for _, mutate := range []func(*trpc.Activity){func(v *trpc.Activity) { v.Metadata = nil }, func(v *trpc.Activity) { v.Metadata.Market.Evidence = nil }, func(v *trpc.Activity) { v.TargetDisplaySnapshot.DisplayName = nil }, func(v *trpc.Activity) { v.SummaryProgress.RelatedPartCounts = nil }, func(v *trpc.Activity) { v.Metadata.Legs.Items[0] = nil }, func(v *trpc.Activity) { v.PriceEvidence = nil }, func(v *trpc.Activity) { v.SourceLocation = nil }} {
		v := completeWire[trpc.Activity]()
		mutate(v)
		m := &responseMapper{}
		m.mapActivity(v)
		if status.Code(m.err) != codes.Unavailable {
			t.Fatal("invalid nested response accepted", m.err)
		}
	}
}
