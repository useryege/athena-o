package solidity

import (
	"context"
	"errors"
	"testing"

	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	soliditypkg "github.com/useryege/athena/pkg/apiclient/solidity"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

type fakeSolidityClientset struct {
	client *fakeSolidityServiceClient
	err    error
}

func (f *fakeSolidityClientset) NewSolidityServiceClient() (utilio.Closer, solidityapiclient.SolidityServiceClient, error) {
	return utilio.NopCloser, f.client, f.err
}

type fakeSolidityServiceClient struct {
	statusResp *v1alpha1.SolidityStatus
	statusErr  error
}

func (f *fakeSolidityServiceClient) GetSolidityStatus(context.Context, *solidityapiclient.GetSolidityStatusRequest, ...grpc.CallOption) (*v1alpha1.SolidityStatus, error) {
	return f.statusResp, f.statusErr
}

func (f *fakeSolidityServiceClient) GetContractSourceInfo(context.Context, *solidityapiclient.GetContractSourceInfoRequest, ...grpc.CallOption) (*v1alpha1.ContractSourceInfo, error) {
	return nil, nil
}

func (f *fakeSolidityServiceClient) ListBytecodeBlacklistEntries(context.Context, *solidityapiclient.ListBytecodeBlacklistEntriesRequest, ...grpc.CallOption) (*solidityapiclient.ListBytecodeBlacklistEntriesResponse, error) {
	return nil, nil
}

func (f *fakeSolidityServiceClient) AddBytecodeBlacklistEntry(context.Context, *solidityapiclient.AddBytecodeBlacklistEntryRequest, ...grpc.CallOption) (*solidityapiclient.AddBytecodeBlacklistEntryResponse, error) {
	return nil, nil
}

func (f *fakeSolidityServiceClient) UpdateBytecodeBlacklistNote(context.Context, *solidityapiclient.UpdateBytecodeBlacklistNoteRequest, ...grpc.CallOption) (*solidityapiclient.UpdateBytecodeBlacklistNoteResponse, error) {
	return nil, nil
}

func (f *fakeSolidityServiceClient) DeleteBytecodeBlacklist(context.Context, *solidityapiclient.DeleteBytecodeBlacklistRequest, ...grpc.CallOption) (*solidityapiclient.DeleteBytecodeBlacklistResponse, error) {
	return nil, nil
}

func TestGetSolidityStatusForwardsResponse(t *testing.T) {
	resp, err := NewServer(&fakeSolidityClientset{client: &fakeSolidityServiceClient{
		statusResp: &v1alpha1.SolidityStatus{Started: true, Status: "running"},
	}}).GetSolidityStatus(context.Background(), &soliditypkg.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestGetSolidityStatusPropagatesClientError(t *testing.T) {
	wantErr := errors.New("connect failed")
	_, err := NewServer(&fakeSolidityClientset{err: wantErr}).GetSolidityStatus(context.Background(), &soliditypkg.GetSolidityStatusRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}
