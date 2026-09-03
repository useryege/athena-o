package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/profile"
	profileapp "github.com/useryege/athena/internal/token/profile/application"
)

var (
	_ profileapp.BuildRepository = (*ProfileRepository)(nil)
	_ profileapp.ReadRepository  = (*ProfileRepository)(nil)
)

func (repository *ProfileRepository) ClaimProfileBuildTask(ctx context.Context, lease time.Duration) (*profile.BuildTask, error) {
	leaseSeconds, err := positiveWholeSeconds("profile build task lease", lease)
	if err != nil {
		return nil, err
	}
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.ClaimProjectProfileBuildTask(ctx, leaseSeconds)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim project profile build task: %w", err)
	}
	task, err := mapProfileBuildTask(row)
	if err != nil {
		return nil, fmt.Errorf("map claimed project profile build task: %w", err)
	}
	return &task, nil
}

func (repository *ProfileRepository) RenewProfileBuildTaskLease(
	ctx context.Context,
	task profile.BuildTask,
	lease time.Duration,
) (bool, error) {
	leaseSeconds, err := positiveWholeSeconds("profile build task lease", lease)
	if err != nil {
		return false, err
	}
	queries, err := repository.querier()
	if err != nil {
		return false, err
	}
	rows, err := queries.RenewProjectProfileBuildTaskLease(ctx, tokensqlc.RenewProjectProfileBuildTaskLeaseParams{
		LeaseSeconds: leaseSeconds, ProjectID: task.ProjectID, ClaimGeneration: task.ClaimGeneration,
	})
	if err != nil {
		return false, fmt.Errorf("renew project %d profile build lease: %w", task.ProjectID, err)
	}
	return rows == 1, nil
}

func (repository *ProfileRepository) GetProfileBuildInput(ctx context.Context, projectID int64) (profile.BuildInput, error) {
	if repository == nil || repository.pool == nil {
		return profile.BuildInput{}, fmt.Errorf("token profile repository is not configured")
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("begin project %d profile input snapshot: %w", projectID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)

	projectRow, err := queries.GetProject(ctx, projectID)
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("get project %d for profile: %w", projectID, err)
	}
	projectValue, err := mapProfileProjectSnapshot(projectRow)
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("map project %d for profile: %w", projectID, err)
	}
	walletRows, err := queries.ListProjectRelatedWalletsByProject(ctx, projectID)
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("list project %d related wallets for profile: %w", projectID, err)
	}
	projectValue.RelatedWallets = make([]profile.RelatedWallet, 0, len(walletRows))
	for _, row := range walletRows {
		projectValue.RelatedWallets = append(projectValue.RelatedWallets, profile.RelatedWallet{
			Address: bytesToAddress(row.Wallet), Role: row.Role,
		})
	}
	recipientRows, err := queries.ListProjectInitialRecipientsByProject(ctx, projectID)
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("list project %d initial recipients for profile: %w", projectID, err)
	}
	projectValue.InitialRecipients = make([]profile.InitialRecipient, 0, len(recipientRows))
	for _, row := range recipientRows {
		ratio, mapErr := int64ToUint64("initial_recipient_ratio_bps", row.RatioBps)
		if mapErr != nil {
			return profile.BuildInput{}, fmt.Errorf("map project %d initial recipient: %w", projectID, mapErr)
		}
		projectValue.InitialRecipients = append(projectValue.InitialRecipients, profile.InitialRecipient{
			Address: bytesToAddress(row.Wallet), Rank: row.RankIndex, RatioBPS: ratio,
		})
	}
	taskRows, err := queries.ListProjectDataCollectionTasksByProject(ctx, projectID)
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("list project %d collection tasks for profile: %w", projectID, err)
	}
	tasks := make([]collection.Task, 0, len(taskRows))
	for _, row := range taskRows {
		task, mapErr := mapCollectionTask(row)
		if mapErr != nil {
			return profile.BuildInput{}, fmt.Errorf("map project %d collection task %d: %w", projectID, row.ID, mapErr)
		}
		tasks = append(tasks, task)
	}
	resultRows, err := queries.ListProjectDataCollectionResultsByProject(ctx, projectID)
	if err != nil {
		return profile.BuildInput{}, fmt.Errorf("list project %d collection results for profile: %w", projectID, err)
	}
	results := make([]collection.Result, 0, len(resultRows))
	for _, row := range resultRows {
		result, mapErr := mapCollectionResult(row)
		if mapErr != nil {
			return profile.BuildInput{}, fmt.Errorf("map project %d collection result %d: %w", projectID, row.TaskID, mapErr)
		}
		results = append(results, result)
	}
	if err := tx.Commit(ctx); err != nil {
		return profile.BuildInput{}, fmt.Errorf("commit project %d profile input snapshot: %w", projectID, err)
	}
	return profile.BuildInput{Project: projectValue, Tasks: tasks, Results: results}, nil
}

func (repository *ProfileRepository) CommitProfile(
	ctx context.Context,
	command profileapp.CommitProfileCommand,
) (profileapp.CommitProfileResult, error) {
	if repository == nil || repository.pool == nil {
		return profileapp.CommitProfileResult{}, fmt.Errorf("token profile repository is not configured")
	}
	params, err := profileInsertParams(command.Profile)
	if err != nil {
		return profileapp.CommitProfileResult{}, err
	}
	if command.Task.ProjectID != command.Profile.ProjectID || command.Task.Status != profile.BuildTaskStatusRunning || command.Task.ClaimGeneration <= 0 {
		return profileapp.CommitProfileResult{}, fmt.Errorf("project %d profile build task does not have a matching active claim", command.Task.ProjectID)
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return profileapp.CommitProfileResult{}, fmt.Errorf("begin project %d profile commit: %w", command.Task.ProjectID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	locked, err := queries.LockProjectProfileBuildTaskForCompletion(ctx, tokensqlc.LockProjectProfileBuildTaskForCompletionParams{
		ProjectID: command.Task.ProjectID, ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return profileapp.CommitProfileResult{}, profile.ErrBuildTaskClaimLost
	} else if err != nil {
		return profileapp.CommitProfileResult{}, fmt.Errorf("lock project %d profile build task: %w", command.Task.ProjectID, err)
	}
	if locked.Status == string(profile.BuildTaskStatusSucceeded) {
		existing, getErr := queries.GetProjectProfile(ctx, command.Task.ProjectID)
		if getErr != nil {
			return profileapp.CommitProfileResult{}, fmt.Errorf("get succeeded project %d profile: %w", command.Task.ProjectID, getErr)
		}
		if !bytes.Equal(existing.ContentHash, command.Profile.ContentHash.Bytes()) {
			return profileapp.CommitProfileResult{}, profile.ErrContentConflict
		}
		mapped, mapErr := mapProjectProfile(existing)
		if mapErr != nil {
			return profileapp.CommitProfileResult{}, fmt.Errorf("map succeeded project %d profile: %w", command.Task.ProjectID, mapErr)
		}
		if err := tx.Commit(ctx); err != nil {
			return profileapp.CommitProfileResult{}, fmt.Errorf("commit project %d idempotent profile submission: %w", command.Task.ProjectID, err)
		}
		return profileapp.CommitProfileResult{Profile: mapped, Created: false}, nil
	}

	created := true
	var stored tokensqlc.ProjectProfile
	existing, getErr := queries.GetProjectProfile(ctx, command.Task.ProjectID)
	switch {
	case getErr == nil:
		if !bytes.Equal(existing.ContentHash, command.Profile.ContentHash.Bytes()) {
			return profileapp.CommitProfileResult{}, profile.ErrContentConflict
		}
		stored = existing
		created = false
	case errors.Is(getErr, pgx.ErrNoRows):
		stored, err = queries.InsertProjectProfile(ctx, params)
		if errors.Is(err, pgx.ErrNoRows) {
			return profileapp.CommitProfileResult{}, profile.ErrContentConflict
		}
		if err != nil {
			return profileapp.CommitProfileResult{}, fmt.Errorf("insert project %d profile: %w", command.Task.ProjectID, err)
		}
	default:
		return profileapp.CommitProfileResult{}, fmt.Errorf("get project %d existing profile: %w", command.Task.ProjectID, getErr)
	}

	if _, err := queries.MarkProjectProfileBuildTaskSucceeded(ctx, tokensqlc.MarkProjectProfileBuildTaskSucceededParams{
		FinishedAt: nullableTime(command.Profile.BuiltAt.UTC()), ProjectID: command.Task.ProjectID,
		ClaimGeneration: command.Task.ClaimGeneration,
	}); errors.Is(err, pgx.ErrNoRows) {
		return profileapp.CommitProfileResult{}, profile.ErrBuildTaskClaimLost
	} else if err != nil {
		return profileapp.CommitProfileResult{}, fmt.Errorf("mark project %d profile build succeeded: %w", command.Task.ProjectID, err)
	}
	mapped, err := mapProjectProfile(stored)
	if err != nil {
		return profileapp.CommitProfileResult{}, fmt.Errorf("map committed project %d profile: %w", command.Task.ProjectID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return profileapp.CommitProfileResult{}, fmt.Errorf("commit project %d profile: %w", command.Task.ProjectID, err)
	}
	return profileapp.CommitProfileResult{Profile: mapped, Created: created}, nil
}

func (repository *ProfileRepository) RetryProfileBuildTask(
	ctx context.Context,
	command profileapp.RetryProfileBuildTaskCommand,
) (bool, error) {
	if err := validateProfileFailureTransition(command.Task, command.FailureCount); err != nil {
		return false, err
	}
	if command.FailureCount >= 3 || command.AvailableAt.IsZero() {
		return false, fmt.Errorf("project %d profile retry requires a nonterminal failure and availability time", command.Task.ProjectID)
	}
	queries, err := repository.querier()
	if err != nil {
		return false, err
	}
	row, err := queries.RetryProjectProfileBuildTask(ctx, tokensqlc.RetryProjectProfileBuildTaskParams{
		AvailableAt: nullableTime(command.AvailableAt.UTC()), LastError: nullableText(command.LastError),
		ProjectID: command.Task.ProjectID, ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("retry project %d profile build: %w", command.Task.ProjectID, err)
	}
	if row.FailureCount != command.FailureCount {
		return false, fmt.Errorf("project %d profile retry returned failure count %d, expected %d", command.Task.ProjectID, row.FailureCount, command.FailureCount)
	}
	return true, nil
}

func (repository *ProfileRepository) FailProfileBuildTask(
	ctx context.Context,
	command profileapp.FailProfileBuildTaskCommand,
) (bool, error) {
	if err := validateProfileFailureTransition(command.Task, command.FailureCount); err != nil {
		return false, err
	}
	if command.FailureCount != 3 || command.FailedAt.IsZero() {
		return false, fmt.Errorf("project %d profile terminal failure requires failure count 3 and a failure time", command.Task.ProjectID)
	}
	queries, err := repository.querier()
	if err != nil {
		return false, err
	}
	row, err := queries.FailProjectProfileBuildTask(ctx, tokensqlc.FailProjectProfileBuildTaskParams{
		LastError: nullableText(command.LastError), FinishedAt: nullableTime(command.FailedAt.UTC()),
		ProjectID: command.Task.ProjectID, ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("fail project %d profile build: %w", command.Task.ProjectID, err)
	}
	if row.FailureCount != command.FailureCount || row.Status != string(profile.BuildTaskStatusFailed) {
		return false, fmt.Errorf("project %d profile terminal failure returned inconsistent state", command.Task.ProjectID)
	}
	return true, nil
}

func (repository *ProfileRepository) GetProjectProfile(ctx context.Context, projectID int64) (*profile.ProjectProfile, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectProfile(ctx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project %d profile: %w", projectID, err)
	}
	value, err := mapProjectProfile(row)
	if err != nil {
		return nil, fmt.Errorf("map project %d profile: %w", projectID, err)
	}
	return &value, nil
}

func mapProfileBuildTask(row tokensqlc.ProjectProfileBuildTask) (profile.BuildTask, error) {
	status := profile.BuildTaskStatus(row.Status)
	switch status {
	case profile.BuildTaskStatusPending, profile.BuildTaskStatusRunning, profile.BuildTaskStatusSucceeded, profile.BuildTaskStatusFailed:
	default:
		return profile.BuildTask{}, fmt.Errorf("unsupported profile build status %q", row.Status)
	}
	if row.FailureCount < 0 || row.FailureCount > 3 {
		return profile.BuildTask{}, fmt.Errorf("profile failure count %d is outside 0..3", row.FailureCount)
	}
	return profile.BuildTask{
		ProjectID: row.ProjectID, Status: status, FailureCount: row.FailureCount,
		AvailableAt: timeValue(row.AvailableAt), ClaimGeneration: row.ClaimGeneration,
		LockedAt: timeValue(row.LockedAt), LeaseExpiresAt: timeValue(row.LeaseExpiresAt),
		LastError: textValue(row.LastError), FinishedAt: timeValue(row.FinishedAt),
		CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt),
	}, nil
}

func mapProfileProjectSnapshot(row tokensqlc.Project) (profile.ProjectSnapshot, error) {
	blockNumber, err := int64ToUint64("deployment_block_number", row.BlockNumber)
	if err != nil {
		return profile.ProjectSnapshot{}, err
	}
	return profile.ProjectSnapshot{
		ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), CodeHash: bytesToHash(row.CodeHash),
		WrappedNativePair: bytesToAddress(row.WethPair), USDTPair: bytesToAddress(row.UsdtPair),
		DeploymentBlockNumber: blockNumber,
	}, nil
}

func mapCollectionResult(row tokensqlc.ProjectDataCollectionResult) (collection.Result, error) {
	dataType, ok := collection.ParseDataType(row.DataType)
	if !ok {
		return collection.Result{}, fmt.Errorf("unsupported collection result data type %q", row.DataType)
	}
	blockNumber, err := uint64PointerFromInt64("collection_result_block_number", row.BlockNumber)
	if err != nil {
		return collection.Result{}, err
	}
	if row.SchemaVersion <= 0 || len(row.Payload) == 0 || len(row.ContentHash) != 32 || !row.CollectedAt.Valid || !row.CreatedAt.Valid {
		return collection.Result{}, fmt.Errorf("collection result %d has incomplete persisted fields", row.TaskID)
	}
	canonical, contentHash, err := collection.NormalizeResult(row.SchemaVersion, row.Payload)
	if err != nil {
		return collection.Result{}, fmt.Errorf("normalize collection result %d: %w", row.TaskID, err)
	}
	if contentHash != bytesToHash(row.ContentHash) {
		return collection.Result{}, fmt.Errorf("collection result %d content hash does not match payload", row.TaskID)
	}
	return collection.Result{
		TaskID: row.TaskID, ProjectID: row.ProjectID, DataType: dataType, SchemaVersion: row.SchemaVersion,
		Payload: json.RawMessage(canonical), ContentHash: contentHash,
		BlockNumber: blockNumber, CollectedAt: timeValue(row.CollectedAt), CreatedAt: timeValue(row.CreatedAt),
	}, nil
}

func profileInsertParams(value profile.ProjectProfile) (tokensqlc.InsertProjectProfileParams, error) {
	if value.ProjectID <= 0 || value.SchemaVersion <= 0 || value.BuiltAt.IsZero() {
		return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project profile identity, schema version, and build time are required")
	}
	if value.Profile.ProjectID != value.ProjectID || value.Profile.SchemaVersion != value.SchemaVersion ||
		value.Profile.CompletenessStatus != value.CompletenessStatus {
		return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project %d profile identity does not match its canonical content", value.ProjectID)
	}
	if !value.Profile.BuiltAt.Equal(value.BuiltAt) {
		return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project %d profile build time does not match its canonical content", value.ProjectID)
	}
	payload, err := json.Marshal(value.Profile)
	if err != nil {
		return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("marshal project %d profile: %w", value.ProjectID, err)
	}
	digest := sha256.Sum256(payload)
	if !bytes.Equal(digest[:], value.ContentHash.Bytes()) {
		return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project %d profile content hash does not match canonical content", value.ProjectID)
	}
	failedTypes := make([]string, 0, len(value.FailedDataTypes))
	for _, dataType := range value.FailedDataTypes {
		if _, ok := collection.ParseDataType(string(dataType)); !ok {
			return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project %d profile has unsupported failed data type %q", value.ProjectID, dataType)
		}
		failedTypes = append(failedTypes, string(dataType))
	}
	if len(value.Profile.FailedDataTypes) != len(value.FailedDataTypes) {
		return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project %d profile failed data types do not match canonical content", value.ProjectID)
	}
	for index := range value.FailedDataTypes {
		if value.Profile.FailedDataTypes[index] != value.FailedDataTypes[index] {
			return tokensqlc.InsertProjectProfileParams{}, fmt.Errorf("project %d profile failed data types do not match canonical content", value.ProjectID)
		}
	}
	projection := profileProjectionFromContent(value.Profile)
	params := tokensqlc.InsertProjectProfileParams{
		ProjectID: value.ProjectID, SchemaVersion: value.SchemaVersion,
		CompletenessStatus: string(value.CompletenessStatus), FailedDataTypes: failedTypes,
		Profile: payload, ContentHash: value.ContentHash.Bytes(), LogoUrl: projection.LogoURL,
		BuiltAt: nullableTime(value.BuiltAt.UTC()),
	}
	if params.CurrentPriceUsd, err = nullableNumericFromDecimal(projection.CurrentPriceUSD); err != nil {
		return params, fmt.Errorf("map project %d current price: %w", value.ProjectID, err)
	}
	if params.MarketCapUsd, err = nullableNumericFromDecimal(projection.MarketCapUSD); err != nil {
		return params, fmt.Errorf("map project %d market cap: %w", value.ProjectID, err)
	}
	if params.FdvUsd, err = nullableNumericFromDecimal(projection.FDVUSD); err != nil {
		return params, fmt.Errorf("map project %d FDV: %w", value.ProjectID, err)
	}
	if params.TvlUsd, err = nullableNumericFromDecimal(projection.TVLUSD); err != nil {
		return params, fmt.Errorf("map project %d TVL: %w", value.ProjectID, err)
	}
	if projection.Holders != nil {
		params.Holders = pgtype.Int8{Int64: *projection.Holders, Valid: true}
	}
	if projection.ContractSourceStatus != nil {
		params.ContractSourceStatus = pgtype.Text{String: string(*projection.ContractSourceStatus), Valid: true}
	}
	if err := applyPairInsertProjection(&params, projection.WrappedNativePair, true); err != nil {
		return params, fmt.Errorf("map project %d wrapped-native pair projection: %w", value.ProjectID, err)
	}
	if err := applyPairInsertProjection(&params, projection.USDTPair, false); err != nil {
		return params, fmt.Errorf("map project %d USDT pair projection: %w", value.ProjectID, err)
	}
	return params, nil
}

func profileProjectionFromContent(value profile.ProjectProfileV1) profile.Projection {
	projection := profile.Projection{}
	if value.Market != nil {
		projection.LogoURL = value.Market.LogoURL
		projection.CurrentPriceUSD = value.Market.CurrentPriceUSD
		projection.MarketCapUSD = value.Market.MarketCapUSD
		projection.FDVUSD = value.Market.FDVUSD
		projection.TVLUSD = value.Market.TVLUSD
		holders := value.Market.Holders
		projection.Holders = &holders
	}
	if value.ContractSource != nil {
		status := value.ContractSource.VerificationStatus
		projection.ContractSourceStatus = &status
	}
	projection.WrappedNativePair = profilePairProjectionFromContent(value.Pairs.WrappedNative)
	projection.USDTPair = profilePairProjectionFromContent(value.Pairs.USDT)
	return projection
}

func profilePairProjectionFromContent(value *profile.PairProfileV1) *profile.PairProjection {
	if value == nil || value.ChainState == nil {
		return nil
	}
	state := value.ChainState
	return &profile.PairProjection{
		IsCreated: state.IsCreated, PairTokenBalanceExceedsTotalSupply: state.Signals.PairTokenBalanceExceedsTotalSupply,
		LPMinimumSupplyOnly:                state.Signals.LPMinimumSupplyOnly,
		FixedFeeAddressLPShareGte90Percent: state.Signals.FixedFeeAddressLPShareGte90Percent,
		QuoteUsdtValueInt:                  state.QuoteUsdtValueInt, ReserveUpdatedAt: state.ReserveUpdatedAt,
	}
}

func applyPairInsertProjection(params *tokensqlc.InsertProjectProfileParams, value *profile.PairProjection, wrappedNative bool) error {
	if value == nil {
		return nil
	}
	quote := nullableNumericFromBigInt(value.QuoteUsdtValueInt)
	reserve, err := uint64ToInt64("reserve_updated_at", value.ReserveUpdatedAt)
	if err != nil {
		return err
	}
	created := pgtype.Bool{Bool: value.IsCreated, Valid: true}
	balance := pgtype.Bool{Bool: value.PairTokenBalanceExceedsTotalSupply, Valid: true}
	minimum := pgtype.Bool{Bool: value.LPMinimumSupplyOnly, Valid: true}
	share := pgtype.Bool{Bool: value.FixedFeeAddressLPShareGte90Percent, Valid: true}
	updatedAt := pgtype.Int8{Int64: reserve, Valid: true}
	if wrappedNative {
		params.WethPairIsCreated = created
		params.WethPairTokenBalanceExceedsTotalSupply = balance
		params.WethPairLpMinimumSupplyOnly = minimum
		params.WethPairFixedFeeAddressLpShareGte90Percent = share
		params.WethPairQuoteUsdtValueInt = quote
		params.WethPairReserveUpdatedAt = updatedAt
		return nil
	}
	params.UsdtPairIsCreated = created
	params.UsdtPairTokenBalanceExceedsTotalSupply = balance
	params.UsdtPairLpMinimumSupplyOnly = minimum
	params.UsdtPairFixedFeeAddressLpShareGte90Percent = share
	params.UsdtPairQuoteUsdtValueInt = quote
	params.UsdtPairReserveUpdatedAt = updatedAt
	return nil
}

func mapProjectProfile(row tokensqlc.ProjectProfile) (profile.ProjectProfile, error) {
	if row.ProjectID <= 0 || row.SchemaVersion <= 0 || len(row.ContentHash) != 32 || !row.BuiltAt.Valid || !row.CreatedAt.Valid {
		return profile.ProjectProfile{}, fmt.Errorf("profile has incomplete persisted identity or timestamps")
	}
	status := profile.CompletenessStatus(row.CompletenessStatus)
	switch status {
	case profile.CompletenessStatusComplete, profile.CompletenessStatusIncomplete:
	default:
		return profile.ProjectProfile{}, fmt.Errorf("unsupported completeness status %q", row.CompletenessStatus)
	}
	var content profile.ProjectProfileV1
	if err := json.Unmarshal(row.Profile, &content); err != nil {
		return profile.ProjectProfile{}, fmt.Errorf("decode profile JSON: %w", err)
	}
	canonical, err := json.Marshal(content)
	if err != nil {
		return profile.ProjectProfile{}, fmt.Errorf("normalize profile JSON: %w", err)
	}
	digest := sha256.Sum256(canonical)
	if !bytes.Equal(digest[:], row.ContentHash) {
		return profile.ProjectProfile{}, fmt.Errorf("profile JSON content hash does not match persisted hash")
	}
	failedTypes := make([]collection.DataType, 0, len(row.FailedDataTypes))
	for _, raw := range row.FailedDataTypes {
		dataType, ok := collection.ParseDataType(raw)
		if !ok {
			return profile.ProjectProfile{}, fmt.Errorf("unsupported failed data type %q", raw)
		}
		failedTypes = append(failedTypes, dataType)
	}
	if content.ProjectID != row.ProjectID || content.SchemaVersion != row.SchemaVersion || content.CompletenessStatus != status {
		return profile.ProjectProfile{}, fmt.Errorf("profile JSON identity does not match persisted columns")
	}
	if !content.BuiltAt.Equal(timeValue(row.BuiltAt)) {
		return profile.ProjectProfile{}, fmt.Errorf("profile JSON build time does not match persisted column")
	}
	if len(content.FailedDataTypes) != len(failedTypes) {
		return profile.ProjectProfile{}, fmt.Errorf("profile JSON failed data types do not match persisted columns")
	}
	for index := range failedTypes {
		if content.FailedDataTypes[index] != failedTypes[index] {
			return profile.ProjectProfile{}, fmt.Errorf("profile JSON failed data types do not match persisted columns")
		}
	}
	projection, err := mapProfileProjection(row)
	if err != nil {
		return profile.ProjectProfile{}, err
	}
	return profile.ProjectProfile{
		ProjectID: row.ProjectID, SchemaVersion: row.SchemaVersion, CompletenessStatus: status,
		FailedDataTypes: failedTypes, ContentHash: bytesToHash(row.ContentHash), Profile: content,
		Projection: projection, BuiltAt: timeValue(row.BuiltAt), CreatedAt: timeValue(row.CreatedAt),
	}, nil
}

func mapProfileProjection(row tokensqlc.ProjectProfile) (profile.Projection, error) {
	result := profile.Projection{LogoURL: row.LogoUrl}
	var err error
	if result.CurrentPriceUSD, err = decimalFromNumeric("current_price_usd", row.CurrentPriceUsd); err != nil {
		return result, err
	}
	if result.MarketCapUSD, err = decimalFromNumeric("market_cap_usd", row.MarketCapUsd); err != nil {
		return result, err
	}
	if result.FDVUSD, err = decimalFromNumeric("fdv_usd", row.FdvUsd); err != nil {
		return result, err
	}
	if result.TVLUSD, err = decimalFromNumeric("tvl_usd", row.TvlUsd); err != nil {
		return result, err
	}
	if row.Holders.Valid {
		value := row.Holders.Int64
		result.Holders = &value
	}
	if row.ContractSourceStatus.Valid {
		value := profile.SourceVerificationStatus(row.ContractSourceStatus.String)
		switch value {
		case profile.SourceVerificationStatusVerified, profile.SourceVerificationStatusUnverified:
		default:
			return result, fmt.Errorf("unsupported contract source status %q", value)
		}
		result.ContractSourceStatus = &value
	}
	if result.WrappedNativePair, err = mapPairProjection("weth", row.WethPairIsCreated, row.WethPairTokenBalanceExceedsTotalSupply, row.WethPairLpMinimumSupplyOnly, row.WethPairFixedFeeAddressLpShareGte90Percent, row.WethPairQuoteUsdtValueInt, row.WethPairReserveUpdatedAt); err != nil {
		return result, err
	}
	if result.USDTPair, err = mapPairProjection("usdt", row.UsdtPairIsCreated, row.UsdtPairTokenBalanceExceedsTotalSupply, row.UsdtPairLpMinimumSupplyOnly, row.UsdtPairFixedFeeAddressLpShareGte90Percent, row.UsdtPairQuoteUsdtValueInt, row.UsdtPairReserveUpdatedAt); err != nil {
		return result, err
	}
	return result, nil
}

func mapPairProjection(field string, created, balance, minimum, share pgtype.Bool, quote pgtype.Numeric, updatedAt pgtype.Int8) (*profile.PairProjection, error) {
	present := created.Valid || balance.Valid || minimum.Valid || share.Valid || quote.Valid || updatedAt.Valid
	if !present {
		return nil, nil
	}
	if !created.Valid || !balance.Valid || !minimum.Valid || !share.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("%s pair projection is partially populated", field)
	}
	reserve, err := int64ToUint64(field+"_pair_reserve_updated_at", updatedAt.Int64)
	if err != nil {
		return nil, err
	}
	quoteValue, err := exactBigIntPointerFromNumeric(field+"_pair_quote_usdt_value_int", quote)
	if err != nil {
		return nil, err
	}
	return &profile.PairProjection{
		IsCreated: created.Bool, PairTokenBalanceExceedsTotalSupply: balance.Bool,
		LPMinimumSupplyOnly: minimum.Bool, FixedFeeAddressLPShareGte90Percent: share.Bool,
		QuoteUsdtValueInt: quoteValue, ReserveUpdatedAt: reserve,
	}, nil
}

func validateProfileFailureTransition(task profile.BuildTask, failureCount int32) error {
	if task.ProjectID <= 0 || task.Status != profile.BuildTaskStatusRunning || task.ClaimGeneration <= 0 {
		return fmt.Errorf("profile build task must have a matching active claim")
	}
	if failureCount != task.FailureCount+1 {
		return fmt.Errorf("project %d profile failure count must advance from %d to %d", task.ProjectID, task.FailureCount, task.FailureCount+1)
	}
	return nil
}
