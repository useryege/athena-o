package notification

import (
	"bytes"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/useryege/athena/internal/notification/apiclient"
	"strings"
	"testing"
)

func TestDeliveryStatusJSONRoundTrip(t *testing.T) {
	for _, value := range []string{"SENDING", "UNKNOWN", "CANCELLED"} {
		raw := []byte(`{"status":"NOTIFICATION_DELIVERY_STATUS_` + value + `"}`)
		var input apiclient.SendSystemNotificationResponse
		if err := jsonpb.Unmarshal(strings.NewReader(string(raw)), &input); err != nil {
			t.Fatal(err)
		}
		got := deliveryStatusString(input.Status)
		want := map[string]string{"SENDING": "sending", "UNKNOWN": "unknown", "CANCELLED": "cancelled"}[value]
		if got != want {
			t.Fatalf("mapped %q want %q", got, want)
		}
		var buffer bytes.Buffer
		err := (&jsonpb.Marshaler{}).Marshal(&buffer, &input)
		if err != nil {
			t.Fatal(err)
		}
		encoded := buffer.Bytes()
		var output apiclient.SendSystemNotificationResponse
		if err = jsonpb.Unmarshal(bytes.NewReader(encoded), &output); err != nil || output.Status != input.Status {
			t.Fatalf("round trip %s: %v", encoded, err)
		}
	}
}
