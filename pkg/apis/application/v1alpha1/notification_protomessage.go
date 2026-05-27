//go:build !kubernetes_protomessage_one_more_release
// +build !kubernetes_protomessage_one_more_release

package v1alpha1

func (*NotificationStatus) ProtoMessage()         {}
func (*NotificationDeliveryItem) ProtoMessage()   {}
func (*NotificationDeliveryDetail) ProtoMessage() {}
