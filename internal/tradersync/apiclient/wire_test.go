package apiclient

import (
	"math"
	"testing"

	"github.com/gogo/protobuf/proto"
)

// TestWirePreservesFalseAndMaxRevision catches loss of scalar presence and
// uint64 precision across the internal protobuf boundary.
func TestWirePreservesFalseAndMaxRevision(t *testing.T) {
	in := &Subscription{
		Revision: math.MaxUint64,
		PausedAt: &StringValue{Value: ""},
	}
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal subscription: %v", err)
	}
	var out Subscription
	if err := proto.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal subscription: %v", err)
	}
	if out.Revision != math.MaxUint64 {
		t.Fatalf("revision = %d, want %d", out.Revision, uint64(math.MaxUint64))
	}
	if out.PausedAt == nil {
		t.Fatal("paused_at was absent after round trip")
	}
	if out.PausedAt.Value != "" {
		t.Fatalf("paused_at value = %q, want explicit empty string", out.PausedAt.Value)
	}
	if out.CancelledAt != nil {
		t.Fatalf("cancelled_at = %#v, want absent", out.CancelledAt)
	}

	raw, err = proto.Marshal(&BoolField{Value: &BoolValue{Value: false}})
	if err != nil {
		t.Fatalf("marshal bool field: %v", err)
	}
	var flag BoolField
	if err := proto.Unmarshal(raw, &flag); err != nil {
		t.Fatalf("unmarshal bool field: %v", err)
	}
	if flag.Value == nil {
		t.Fatal("bool value was absent after round trip")
	}
	if flag.Value.Value {
		t.Fatal("bool value = true, want explicit false")
	}
}

// TestWirePreservesCurvePointList catches loss of the optional list wrapper,
// duplicate points, and CurvePoint.t's Unix-second string representation.
func TestWirePreservesCurvePointList(t *testing.T) {
	in := &Curve{Points: &CurvePointList{Items: []*CurvePoint{
		{T: "1722470400", P: "0"},
		{T: "1722470400", P: "0.500000000000000000"},
	}}}
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal curve: %v", err)
	}
	var out Curve
	if err := proto.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal curve: %v", err)
	}
	if out.Points == nil {
		t.Fatal("curve points wrapper was absent after round trip")
	}
	if len(out.Points.Items) != 2 {
		t.Fatalf("curve points count = %d, want 2", len(out.Points.Items))
	}
	if out.Points.Items[0].T != "1722470400" || out.Points.Items[0].P != "0" {
		t.Fatalf("first curve point = %#v, want Unix second and zero price", out.Points.Items[0])
	}
	if out.Points.Items[1].T != "1722470400" || out.Points.Items[1].P != "0.500000000000000000" {
		t.Fatalf("second curve point = %#v, want duplicate Unix second and exact decimal", out.Points.Items[1])
	}
}

// TestWirePreservesKnownEmptyLegList catches conflation of a present empty
// list with an absent optional list wrapper.
func TestWirePreservesKnownEmptyLegList(t *testing.T) {
	in := &TradeMetadata{Legs: &ComboLegList{Items: []*ComboLeg{}}}
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal trade metadata: %v", err)
	}
	var out TradeMetadata
	if err := proto.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal trade metadata: %v", err)
	}
	if out.Legs == nil {
		t.Fatal("known empty legs wrapper was absent after round trip")
	}
	if len(out.Legs.Items) != 0 {
		t.Fatalf("legs count = %d, want 0", len(out.Legs.Items))
	}
}
