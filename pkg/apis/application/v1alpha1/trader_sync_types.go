package v1alpha1

// Trader Sync public resources use exact decimal strings and explicit evidence.
// Missing optional values are omitted; available false, zero and empty values remain.

type TraderSyncFieldEvidence struct {
	Availability string `protobuf:"bytes,1,opt,name=availability" json:"availability"`
	ReasonCode   string `protobuf:"bytes,2,opt,name=reason_code,json=reasonCode" json:"reasonCode"`
	Source       string `protobuf:"bytes,3,opt,name=source" json:"source"`
	QueriedAt    string `protobuf:"bytes,4,opt,name=queried_at,json=queriedAt" json:"queriedAt"`
}

type TraderSyncStringField struct {
	Evidence TraderSyncFieldEvidence `protobuf:"bytes,1,opt,name=evidence" json:"evidence"`
	Value    *string                 `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

type TraderSyncDecimalField struct {
	Evidence TraderSyncFieldEvidence `protobuf:"bytes,1,opt,name=evidence" json:"evidence"`
	Value    *string                 `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

type TraderSyncBoolField struct {
	Evidence TraderSyncFieldEvidence `protobuf:"bytes,1,opt,name=evidence" json:"evidence"`
	Value    *bool                   `protobuf:"varint,2,opt,name=value" json:"value,omitempty"`
}

type TraderSyncTimeField struct {
	Evidence TraderSyncFieldEvidence `protobuf:"bytes,1,opt,name=evidence" json:"evidence"`
	Value    *string                 `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

type TraderSyncCurvePoint struct {
	T string `protobuf:"bytes,1,opt,name=t" json:"t"`
	P string `protobuf:"bytes,2,opt,name=p" json:"p"`
}

type TraderSyncCurve struct {
	Evidence TraderSyncFieldEvidence `protobuf:"bytes,1,opt,name=evidence" json:"evidence"`
	Points   []TraderSyncCurvePoint  `protobuf:"bytes,2,rep,name=points" json:"points"`
}

type TraderSyncPnLView struct {
	Period        string                 `protobuf:"bytes,1,opt,name=period" json:"period"`
	Amount        TraderSyncDecimalField `protobuf:"bytes,2,opt,name=amount" json:"amount"`
	Curve         TraderSyncCurve        `protobuf:"bytes,3,opt,name=curve" json:"curve"`
	Interval      string                 `protobuf:"bytes,4,opt,name=interval" json:"interval"`
	Fidelity      string                 `protobuf:"bytes,5,opt,name=fidelity" json:"fidelity"`
	ReferenceTime TraderSyncTimeField    `protobuf:"bytes,6,opt,name=reference_time,json=referenceTime" json:"referenceTime"`
	Timezone      TraderSyncStringField  `protobuf:"bytes,7,opt,name=timezone" json:"timezone"`
}

type TraderSyncResolvedTarget struct {
	Wallet               string                          `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	CanonicalProfileURL  string                          `protobuf:"bytes,2,opt,name=canonical_profile_url,json=canonicalProfileURL" json:"canonicalProfileURL"`
	Avatar               TraderSyncStringField           `protobuf:"bytes,3,opt,name=avatar" json:"avatar"`
	DisplayName          TraderSyncStringField           `protobuf:"bytes,4,opt,name=display_name,json=displayName" json:"displayName"`
	Verified             TraderSyncBoolField             `protobuf:"bytes,5,opt,name=verified" json:"verified"`
	JoinedAt             TraderSyncTimeField             `protobuf:"bytes,6,opt,name=joined_at,json=joinedAt" json:"joinedAt"`
	PositionValue        TraderSyncDecimalField          `protobuf:"bytes,7,opt,name=position_value,json=positionValue" json:"positionValue"`
	LargestWin           TraderSyncDecimalField          `protobuf:"bytes,8,opt,name=largest_win,json=largestWin" json:"largestWin"`
	Predictions          TraderSyncDecimalField          `protobuf:"bytes,9,opt,name=predictions" json:"predictions"`
	PnL                  []TraderSyncPnLView             `protobuf:"bytes,10,rep,name=pn_l,json=pnl" json:"pnl"`
	DefaultPeriod        string                          `protobuf:"bytes,11,opt,name=default_period,json=defaultPeriod" json:"defaultPeriod"`
	ConfirmationToken    string                          `protobuf:"bytes,12,opt,name=confirmation_token,json=confirmationToken" json:"confirmationToken"`
	ExpiresAt            string                          `protobuf:"bytes,13,opt,name=expires_at,json=expiresAt" json:"expiresAt"`
	UsageNotice          string                          `protobuf:"bytes,14,opt,name=usage_notice,json=usageNotice" json:"usageNotice"`
	SavedNote            *TraderSyncTargetNote           `protobuf:"bytes,15,opt,name=saved_note,json=savedNote" json:"savedNote,omitempty"`
	ExistingSubscription *TraderSyncExistingSubscription `protobuf:"bytes,16,opt,name=existing_subscription,json=existingSubscription" json:"existingSubscription,omitempty"`
	Quota                TraderSyncQuota                 `protobuf:"bytes,17,opt,name=quota" json:"quota"`
}

type TraderSyncTargetNote struct {
	Wallet   string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	Note     string `protobuf:"bytes,2,opt,name=note" json:"note"`
	Revision uint64 `protobuf:"varint,3,opt,name=revision" json:"revision,string"`
}

type TraderSyncQuota struct {
	Used  int32 `protobuf:"varint,1,opt,name=used" json:"used"`
	Limit int32 `protobuf:"varint,2,opt,name=limit" json:"limit"`
}

type TraderSyncExistingSubscription struct {
	ID       string `protobuf:"bytes,1,opt,name=id" json:"id"`
	Status   string `protobuf:"bytes,2,opt,name=status" json:"status"`
	Revision uint64 `protobuf:"varint,3,opt,name=revision" json:"revision,string"`
}

type TraderSyncTargetDisplay struct {
	DisplayName TraderSyncStringField `protobuf:"bytes,1,opt,name=display_name,json=displayName" json:"displayName"`
	Avatar      TraderSyncStringField `protobuf:"bytes,2,opt,name=avatar" json:"avatar"`
	ProfileURL  TraderSyncStringField `protobuf:"bytes,3,opt,name=profile_url,json=profileURL" json:"profileURL"`
}

type TraderSyncSubscription struct {
	ID                   string                  `protobuf:"bytes,1,opt,name=id" json:"id"`
	Wallet               string                  `protobuf:"bytes,2,opt,name=wallet" json:"wallet"`
	Status               string                  `protobuf:"bytes,3,opt,name=status" json:"status"`
	Revision             uint64                  `protobuf:"varint,4,opt,name=revision" json:"revision,string"`
	Generation           uint64                  `protobuf:"varint,5,opt,name=generation" json:"generation,string"`
	Note                 string                  `protobuf:"bytes,6,opt,name=note" json:"note"`
	NoteRevision         uint64                  `protobuf:"varint,7,opt,name=note_revision,json=noteRevision" json:"noteRevision,string"`
	CreatedAt            string                  `protobuf:"bytes,8,opt,name=created_at,json=createdAt" json:"createdAt"`
	UpdatedAt            string                  `protobuf:"bytes,9,opt,name=updated_at,json=updatedAt" json:"updatedAt"`
	PausedAt             *string                 `protobuf:"bytes,10,opt,name=paused_at,json=pausedAt" json:"pausedAt,omitempty"`
	CancelledAt          *string                 `protobuf:"bytes,11,opt,name=cancelled_at,json=cancelledAt" json:"cancelledAt,omitempty"`
	PermissionDisabledAt *string                 `protobuf:"bytes,12,opt,name=permission_disabled_at,json=permissionDisabledAt" json:"permissionDisabledAt,omitempty"`
	CurrentInterval      *TraderSyncInterval     `protobuf:"bytes,13,opt,name=current_interval,json=currentInterval" json:"currentInterval,omitempty"`
	Observation          TraderSyncObservation   `protobuf:"bytes,14,opt,name=observation" json:"observation"`
	BindingStatus        string                  `protobuf:"bytes,15,opt,name=binding_status,json=bindingStatus" json:"bindingStatus"`
	QueueNotice          string                  `protobuf:"bytes,16,opt,name=queue_notice,json=queueNotice" json:"queueNotice"`
	QueueCounts          TraderSyncStatusCounts  `protobuf:"bytes,17,opt,name=queue_counts,json=queueCounts" json:"queueCounts"`
	TargetDisplay        TraderSyncTargetDisplay `protobuf:"bytes,18,opt,name=target_display,json=targetDisplay" json:"targetDisplay"`
}

type TraderSyncInterval struct {
	EffectiveAt string  `protobuf:"bytes,1,opt,name=effective_at,json=effectiveAt" json:"effectiveAt"`
	EndedAt     *string `protobuf:"bytes,2,opt,name=ended_at,json=endedAt" json:"endedAt,omitempty"`
	Generation  uint64  `protobuf:"varint,3,opt,name=generation" json:"generation,string"`
	Epoch       uint64  `protobuf:"varint,4,opt,name=epoch" json:"epoch,string"`
}

type TraderSyncObservation struct {
	State              string                  `protobuf:"bytes,1,opt,name=state" json:"state"`
	Reason             string                  `protobuf:"bytes,2,opt,name=reason" json:"reason"`
	LastReliableAt     *string                 `protobuf:"bytes,3,opt,name=last_reliable_at,json=lastReliableAt" json:"lastReliableAt,omitempty"`
	LatestInterruption *TraderSyncInterruption `protobuf:"bytes,4,opt,name=latest_interruption,json=latestInterruption" json:"latestInterruption,omitempty"`
	InterruptionCount  string                  `protobuf:"bytes,5,opt,name=interruption_count,json=interruptionCount" json:"interruptionCount"`
}

type TraderSyncInterruption struct {
	Start           *string `protobuf:"bytes,1,opt,name=start" json:"start,omitempty"`
	End             *string `protobuf:"bytes,2,opt,name=end" json:"end,omitempty"`
	RecoveredAt     *string `protobuf:"bytes,3,opt,name=recovered_at,json=recoveredAt" json:"recoveredAt,omitempty"`
	Reason          string  `protobuf:"bytes,4,opt,name=reason" json:"reason"`
	Uncertainty     string  `protobuf:"bytes,5,opt,name=uncertainty" json:"uncertainty"`
	PossibleMissing bool    `protobuf:"varint,6,opt,name=possible_missing,json=possibleMissing" json:"possibleMissing"`
}

type TraderSyncHistoryEntry struct {
	ID           string                  `protobuf:"bytes,1,opt,name=id" json:"id"`
	Kind         string                  `protobuf:"bytes,2,opt,name=kind" json:"kind"`
	SortAt       string                  `protobuf:"bytes,3,opt,name=sort_at,json=sortAt" json:"sortAt"`
	Interval     *TraderSyncInterval     `protobuf:"bytes,4,opt,name=interval" json:"interval,omitempty"`
	Interruption *TraderSyncInterruption `protobuf:"bytes,5,opt,name=interruption" json:"interruption,omitempty"`
}

type TraderSyncActivity struct {
	ID                    string                     `protobuf:"bytes,1,opt,name=id" json:"id"`
	SubscriptionID        string                     `protobuf:"bytes,2,opt,name=subscription_id,json=subscriptionId" json:"subscriptionId"`
	SourceRecordID        string                     `protobuf:"bytes,3,opt,name=source_record_id,json=sourceRecordId" json:"sourceRecordId"`
	Wallet                string                     `protobuf:"bytes,4,opt,name=wallet" json:"wallet"`
	Side                  string                     `protobuf:"bytes,5,opt,name=side" json:"side"`
	PositionID            string                     `protobuf:"bytes,6,opt,name=position_id,json=positionId" json:"positionId"`
	CollateralRaw         string                     `protobuf:"bytes,7,opt,name=collateral_raw,json=collateralRaw" json:"collateralRaw"`
	SharesRaw             string                     `protobuf:"bytes,8,opt,name=shares_raw,json=sharesRaw" json:"sharesRaw"`
	FeeRaw                string                     `protobuf:"bytes,9,opt,name=fee_raw,json=feeRaw" json:"feeRaw"`
	CollateralSymbol      string                     `protobuf:"bytes,10,opt,name=collateral_symbol,json=collateralSymbol" json:"collateralSymbol"`
	CollateralDecimals    int32                      `protobuf:"varint,11,opt,name=collateral_decimals,json=collateralDecimals" json:"collateralDecimals"`
	SharesDecimals        int32                      `protobuf:"varint,12,opt,name=shares_decimals,json=sharesDecimals" json:"sharesDecimals"`
	PriceNumerator        string                     `protobuf:"bytes,13,opt,name=price_numerator,json=priceNumerator" json:"priceNumerator"`
	PriceDenominator      string                     `protobuf:"bytes,14,opt,name=price_denominator,json=priceDenominator" json:"priceDenominator"`
	PriceEvidence         TraderSyncFieldEvidence    `protobuf:"bytes,15,opt,name=price_evidence,json=priceEvidence" json:"priceEvidence"`
	SourceVersion         string                     `protobuf:"bytes,16,opt,name=source_version,json=sourceVersion" json:"sourceVersion"`
	SettledAt             string                     `protobuf:"bytes,17,opt,name=settled_at,json=settledAt" json:"settledAt"`
	ReceivedAt            string                     `protobuf:"bytes,18,opt,name=received_at,json=receivedAt" json:"receivedAt"`
	RecordedAt            string                     `protobuf:"bytes,19,opt,name=recorded_at,json=recordedAt" json:"recordedAt"`
	PublicTimeEvidence    TraderSyncFieldEvidence    `protobuf:"bytes,20,opt,name=public_time_evidence,json=publicTimeEvidence" json:"publicTimeEvidence"`
	Metadata              TraderSyncTradeMetadata    `protobuf:"bytes,21,opt,name=metadata" json:"metadata"`
	NoteSnapshot          string                     `protobuf:"bytes,22,opt,name=note_snapshot,json=noteSnapshot" json:"noteSnapshot"`
	NotificationMode      string                     `protobuf:"bytes,23,opt,name=notification_mode,json=notificationMode" json:"notificationMode"`
	NotificationReason    string                     `protobuf:"bytes,24,opt,name=notification_reason,json=notificationReason" json:"notificationReason"`
	Delivery              *TraderSyncDelivery        `protobuf:"bytes,25,opt,name=delivery" json:"delivery,omitempty"`
	SummaryProgress       *TraderSyncSummaryProgress `protobuf:"bytes,26,opt,name=summary_progress,json=summaryProgress" json:"summaryProgress,omitempty"`
	TargetDisplaySnapshot TraderSyncTargetDisplay    `protobuf:"bytes,27,opt,name=target_display_snapshot,json=targetDisplaySnapshot" json:"targetDisplaySnapshot"`
	FinalityAnomaly       *TraderSyncFinalityAnomaly `protobuf:"bytes,28,opt,name=finality_anomaly,json=finalityAnomaly" json:"finalityAnomaly,omitempty"`
	SourceLocation        TraderSyncSourceLocation   `protobuf:"bytes,29,opt,name=source_location,json=sourceLocation" json:"sourceLocation"`
}

type TraderSyncSourceLocation struct {
	ChainID         string `protobuf:"bytes,1,opt,name=chain_id,json=chainId" json:"chainId"`
	ExchangeAddress string `protobuf:"bytes,2,opt,name=exchange_address,json=exchangeAddress" json:"exchangeAddress"`
	TransactionHash string `protobuf:"bytes,3,opt,name=transaction_hash,json=transactionHash" json:"transactionHash"`
	BlockHash       string `protobuf:"bytes,4,opt,name=block_hash,json=blockHash" json:"blockHash"`
	BlockNumber     string `protobuf:"bytes,5,opt,name=block_number,json=blockNumber" json:"blockNumber"`
	LogIndex        string `protobuf:"bytes,6,opt,name=log_index,json=logIndex" json:"logIndex"`
}

type TraderSyncFinalityAnomaly struct {
	Reason               string  `protobuf:"bytes,1,opt,name=reason" json:"reason"`
	DetectedAt           string  `protobuf:"bytes,2,opt,name=detected_at,json=detectedAt" json:"detectedAt"`
	PublishedBlockHash   string  `protobuf:"bytes,3,opt,name=published_block_hash,json=publishedBlockHash" json:"publishedBlockHash"`
	ConflictingBlockHash *string `protobuf:"bytes,4,opt,name=conflicting_block_hash,json=conflictingBlockHash" json:"conflictingBlockHash,omitempty"`
}

type TraderSyncMarketRef struct {
	Evidence    TraderSyncFieldEvidence `protobuf:"bytes,1,opt,name=evidence" json:"evidence"`
	ID          string                  `protobuf:"bytes,2,opt,name=id" json:"id"`
	Title       string                  `protobuf:"bytes,3,opt,name=title" json:"title"`
	URL         string                  `protobuf:"bytes,4,opt,name=url" json:"url"`
	ConditionID string                  `protobuf:"bytes,5,opt,name=condition_id,json=conditionId" json:"conditionId"`
	PositionID  string                  `protobuf:"bytes,6,opt,name=position_id,json=positionId" json:"positionId"`
	Outcome     string                  `protobuf:"bytes,7,opt,name=outcome" json:"outcome"`
}

type TraderSyncComboLeg struct {
	PositionID string              `protobuf:"bytes,1,opt,name=position_id,json=positionId" json:"positionId"`
	Market     TraderSyncMarketRef `protobuf:"bytes,2,opt,name=market" json:"market"`
}

type TraderSyncTradeMetadata struct {
	Market       TraderSyncMarketRef     `protobuf:"bytes,1,opt,name=market" json:"market"`
	LegsEvidence TraderSyncFieldEvidence `protobuf:"bytes,2,opt,name=legs_evidence,json=legsEvidence" json:"legsEvidence"`
	Legs         []TraderSyncComboLeg    `protobuf:"bytes,3,rep,name=legs" json:"legs"`
	Relationship string                  `protobuf:"bytes,4,opt,name=relationship" json:"relationship"`
}

type TraderSyncDelivery struct {
	ID            string             `protobuf:"bytes,1,opt,name=id" json:"id"`
	Status        string             `protobuf:"bytes,2,opt,name=status" json:"status"`
	Reason        string             `protobuf:"bytes,3,opt,name=reason" json:"reason"`
	AuthorizedAt  *string            `protobuf:"bytes,4,opt,name=authorized_at,json=authorizedAt" json:"authorizedAt,omitempty"`
	StartedAt     *string            `protobuf:"bytes,5,opt,name=started_at,json=startedAt" json:"startedAt,omitempty"`
	ResultAt      *string            `protobuf:"bytes,6,opt,name=result_at,json=resultAt" json:"resultAt,omitempty"`
	MessageID     *string            `protobuf:"bytes,7,opt,name=message_id,json=messageId" json:"messageId,omitempty"`
	AttemptCount  string             `protobuf:"bytes,8,opt,name=attempt_count,json=attemptCount" json:"attemptCount"`
	LatestAttempt *TraderSyncAttempt `protobuf:"bytes,9,opt,name=latest_attempt,json=latestAttempt" json:"latestAttempt,omitempty"`
}

type TraderSyncAttempt struct {
	Index        string  `protobuf:"bytes,1,opt,name=index" json:"index"`
	AuthorizedAt string  `protobuf:"bytes,2,opt,name=authorized_at,json=authorizedAt" json:"authorizedAt"`
	StartedAt    *string `protobuf:"bytes,3,opt,name=started_at,json=startedAt" json:"startedAt,omitempty"`
	ResultAt     *string `protobuf:"bytes,4,opt,name=result_at,json=resultAt" json:"resultAt,omitempty"`
	Status       string  `protobuf:"bytes,5,opt,name=status" json:"status"`
	Reason       string  `protobuf:"bytes,6,opt,name=reason" json:"reason"`
}

type TraderSyncStatusCounts struct {
	Total     string `protobuf:"bytes,1,opt,name=total" json:"total"`
	Pending   string `protobuf:"bytes,2,opt,name=pending" json:"pending"`
	Sending   string `protobuf:"bytes,3,opt,name=sending" json:"sending"`
	Sent      string `protobuf:"bytes,4,opt,name=sent" json:"sent"`
	Failed    string `protobuf:"bytes,5,opt,name=failed" json:"failed"`
	Unknown   string `protobuf:"bytes,6,opt,name=unknown" json:"unknown"`
	Cancelled string `protobuf:"bytes,7,opt,name=cancelled" json:"cancelled"`
}

type TraderSyncSummaryProgress struct {
	Phase             string                 `protobuf:"bytes,1,opt,name=phase" json:"phase"`
	Reason            string                 `protobuf:"bytes,2,opt,name=reason" json:"reason"`
	BatchID           *string                `protobuf:"bytes,3,opt,name=batch_id,json=batchId" json:"batchId,omitempty"`
	RelatedPartCounts TraderSyncStatusCounts `protobuf:"bytes,4,opt,name=related_part_counts,json=relatedPartCounts" json:"relatedPartCounts"`
	BatchPartCounts   TraderSyncStatusCounts `protobuf:"bytes,5,opt,name=batch_part_counts,json=batchPartCounts" json:"batchPartCounts"`
	OldestAt          string                 `protobuf:"bytes,6,opt,name=oldest_at,json=oldestAt" json:"oldestAt"`
	FirstStartedAt    *string                `protobuf:"bytes,7,opt,name=first_started_at,json=firstStartedAt" json:"firstStartedAt,omitempty"`
}

type TraderSyncTargetCount struct {
	Wallet string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	Count  string `protobuf:"bytes,2,opt,name=count" json:"count"`
}

type TraderSyncSummaryBatch struct {
	ID             string                  `protobuf:"bytes,1,opt,name=id" json:"id"`
	OldestAt       string                  `protobuf:"bytes,2,opt,name=oldest_at,json=oldestAt" json:"oldestAt"`
	SettledFrom    string                  `protobuf:"bytes,3,opt,name=settled_from,json=settledFrom" json:"settledFrom"`
	SettledTo      string                  `protobuf:"bytes,4,opt,name=settled_to,json=settledTo" json:"settledTo"`
	RecordedFrom   string                  `protobuf:"bytes,5,opt,name=recorded_from,json=recordedFrom" json:"recordedFrom"`
	RecordedTo     string                  `protobuf:"bytes,6,opt,name=recorded_to,json=recordedTo" json:"recordedTo"`
	FirstStartedAt *string                 `protobuf:"bytes,7,opt,name=first_started_at,json=firstStartedAt" json:"firstStartedAt,omitempty"`
	ActivityCount  string                  `protobuf:"bytes,8,opt,name=activity_count,json=activityCount" json:"activityCount"`
	TargetCounts   []TraderSyncTargetCount `protobuf:"bytes,9,rep,name=target_counts,json=targetCounts" json:"targetCounts"`
	PartCounts     TraderSyncStatusCounts  `protobuf:"bytes,10,opt,name=part_counts,json=partCounts" json:"partCounts"`
	AsOf           string                  `protobuf:"bytes,11,opt,name=as_of,json=asOf" json:"asOf"`
}

type TraderSyncSummaryPart struct {
	ID                      string             `protobuf:"bytes,1,opt,name=id" json:"id"`
	Index                   int32              `protobuf:"varint,2,opt,name=index" json:"index"`
	Total                   int32              `protobuf:"varint,3,opt,name=total" json:"total"`
	Delivery                TraderSyncDelivery `protobuf:"bytes,4,opt,name=delivery" json:"delivery"`
	AssociatedActivityCount string             `protobuf:"bytes,5,opt,name=associated_activity_count,json=associatedActivityCount" json:"associatedActivityCount"`
}

type TraderSyncSubscriptionSummary struct {
	SubscriptionID           string                 `protobuf:"bytes,1,opt,name=subscription_id,json=subscriptionId" json:"subscriptionId"`
	AccountID                string                 `protobuf:"bytes,2,opt,name=account_id,json=accountId" json:"accountId"`
	Username                 string                 `protobuf:"bytes,3,opt,name=username" json:"username"`
	Email                    string                 `protobuf:"bytes,4,opt,name=email" json:"email"`
	Wallet                   string                 `protobuf:"bytes,5,opt,name=wallet" json:"wallet"`
	Status                   string                 `protobuf:"bytes,6,opt,name=status" json:"status"`
	CreatedAt                string                 `protobuf:"bytes,7,opt,name=created_at,json=createdAt" json:"createdAt"`
	UpdatedAt                string                 `protobuf:"bytes,8,opt,name=updated_at,json=updatedAt" json:"updatedAt"`
	PausedAt                 *string                `protobuf:"bytes,9,opt,name=paused_at,json=pausedAt" json:"pausedAt,omitempty"`
	CancelledAt              *string                `protobuf:"bytes,10,opt,name=cancelled_at,json=cancelledAt" json:"cancelledAt,omitempty"`
	PermissionDisabledAt     *string                `protobuf:"bytes,11,opt,name=permission_disabled_at,json=permissionDisabledAt" json:"permissionDisabledAt,omitempty"`
	Observation              TraderSyncObservation  `protobuf:"bytes,12,opt,name=observation" json:"observation"`
	ActivityCount            string                 `protobuf:"bytes,13,opt,name=activity_count,json=activityCount" json:"activityCount"`
	AssociatedDeliveryCounts TraderSyncStatusCounts `protobuf:"bytes,14,opt,name=associated_delivery_counts,json=associatedDeliveryCounts" json:"associatedDeliveryCounts"`
	AsOf                     string                 `protobuf:"bytes,15,opt,name=as_of,json=asOf" json:"asOf"`
}

type TraderSyncRuntimeMetric struct {
	Name         string  `protobuf:"bytes,1,opt,name=name" json:"name"`
	Value        string  `protobuf:"bytes,2,opt,name=value" json:"value"`
	Unit         string  `protobuf:"bytes,3,opt,name=unit" json:"unit"`
	Kind         string  `protobuf:"bytes,4,opt,name=kind" json:"kind"`
	WindowStart  *string `protobuf:"bytes,5,opt,name=window_start,json=windowStart" json:"windowStart,omitempty"`
	WindowEnd    *string `protobuf:"bytes,6,opt,name=window_end,json=windowEnd" json:"windowEnd,omitempty"`
	ServiceEpoch *string `protobuf:"bytes,7,opt,name=service_epoch,json=serviceEpoch" json:"serviceEpoch,omitempty"`
}

type TraderSyncRuntimeStatus struct {
	CollectorConnected bool                      `protobuf:"varint,1,opt,name=collector_connected,json=collectorConnected" json:"collectorConnected"`
	CollectorEpoch     string                    `protobuf:"bytes,2,opt,name=collector_epoch,json=collectorEpoch" json:"collectorEpoch"`
	FilterRevision     string                    `protobuf:"bytes,3,opt,name=filter_revision,json=filterRevision" json:"filterRevision"`
	Metrics            []TraderSyncRuntimeMetric `protobuf:"bytes,4,rep,name=metrics" json:"metrics"`
	AsOf               string                    `protobuf:"bytes,5,opt,name=as_of,json=asOf" json:"asOf"`
}
