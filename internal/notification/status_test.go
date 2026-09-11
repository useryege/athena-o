package notification

import "testing"

func TestNormalizeDeliveryStatusFilterIncludesTerminalUnknown(t *testing.T) {
	for _, value := range []string{"pending", "sending", "sent", "failed", "unknown", "cancelled"} {
		actual, err := normalizeDeliveryStatusFilter(" " + value + " ")
		if err != nil || actual != value {
			t.Fatalf("filter %q = %q, %v", value, actual, err)
		}
	}
	if _, err := normalizeDeliveryStatusFilter("bogus"); err == nil {
		t.Fatal("invalid status accepted")
	}
}
