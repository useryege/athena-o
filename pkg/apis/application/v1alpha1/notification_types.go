package v1alpha1

type SystemNotificationDeliveryItem struct {
	ID                int64  `protobuf:"varint,1,opt,name=id" json:"id"`
	Source            string `protobuf:"bytes,2,opt,name=source" json:"source"`
	Severity          string `protobuf:"bytes,3,opt,name=severity" json:"severity"`
	Title             string `protobuf:"bytes,4,opt,name=title" json:"title"`
	Body              string `protobuf:"bytes,5,opt,name=body" json:"body"`
	Link              string `protobuf:"bytes,6,opt,name=link" json:"link"`
	Channel           string `protobuf:"bytes,7,opt,name=channel" json:"channel"`
	Status            string `protobuf:"bytes,8,opt,name=status" json:"status"`
	ProviderMessageID string `protobuf:"bytes,9,opt,name=provider_message_id,json=providerMessageId" json:"providerMessageId"`
	ErrorMessage      string `protobuf:"bytes,10,opt,name=error_message,json=errorMessage" json:"errorMessage"`
	CreatedAt         string `protobuf:"bytes,11,opt,name=created_at,json=createdAt" json:"createdAt"`
	SentAt            string `protobuf:"bytes,12,opt,name=sent_at,json=sentAt" json:"sentAt"`
	TopicLabel        string `protobuf:"bytes,13,opt,name=topic_label,json=topicLabel" json:"topicLabel"`
	TelegramChat      string `protobuf:"bytes,14,opt,name=telegram_chat,json=telegramChat" json:"telegramChat"`
}

type SystemNotificationDeliveryDetail struct {
	ID                int64  `protobuf:"varint,1,opt,name=id" json:"id"`
	Source            string `protobuf:"bytes,2,opt,name=source" json:"source"`
	Severity          string `protobuf:"bytes,3,opt,name=severity" json:"severity"`
	Title             string `protobuf:"bytes,4,opt,name=title" json:"title"`
	Body              string `protobuf:"bytes,5,opt,name=body" json:"body"`
	Link              string `protobuf:"bytes,6,opt,name=link" json:"link"`
	Channel           string `protobuf:"bytes,7,opt,name=channel" json:"channel"`
	Status            string `protobuf:"bytes,8,opt,name=status" json:"status"`
	ProviderMessageID string `protobuf:"bytes,9,opt,name=provider_message_id,json=providerMessageId" json:"providerMessageId"`
	ErrorMessage      string `protobuf:"bytes,10,opt,name=error_message,json=errorMessage" json:"errorMessage"`
	CreatedAt         string `protobuf:"bytes,11,opt,name=created_at,json=createdAt" json:"createdAt"`
	SentAt            string `protobuf:"bytes,12,opt,name=sent_at,json=sentAt" json:"sentAt"`
	TopicLabel        string `protobuf:"bytes,13,opt,name=topic_label,json=topicLabel" json:"topicLabel"`
	TelegramChat      string `protobuf:"bytes,14,opt,name=telegram_chat,json=telegramChat" json:"telegramChat"`
}
