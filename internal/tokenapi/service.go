package tokenapi

import (
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedTokenCatalogServiceServer
	apiclient.UnimplementedTokenCollectionServiceServer
	apiclient.UnimplementedTokenPolicyServiceServer
	apiclient.UnimplementedTokenOperationsServiceServer
	applications Applications
}

type ServiceOpts struct {
	Applications Applications
}

func NewService(opts ServiceOpts) *Service {
	return &Service{applications: opts.Applications}
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func requiredApplication[T any](application T, configured bool) (T, error) {
	if !configured {
		var zero T
		return zero, status.Error(codes.FailedPrecondition, "token application service is not configured")
	}
	return application, nil
}

func (s *Service) catalogApplication() (CatalogApplication, error) {
	return requiredApplication(s.applications.Catalog, s.applications.Catalog != nil)
}
func (s *Service) collectionApplication() (CollectionApplication, error) {
	return requiredApplication(s.applications.Collection, s.applications.Collection != nil)
}
func (s *Service) profileApplication() (ProfileApplication, error) {
	return requiredApplication(s.applications.Profile, s.applications.Profile != nil)
}
func (s *Service) projectViewApplication() (ProjectViewApplication, error) {
	return requiredApplication(s.applications.ProjectView, s.applications.ProjectView != nil)
}
func (s *Service) policyApplication() (PolicyApplication, error) {
	return requiredApplication(s.applications.Policy, s.applications.Policy != nil)
}
func (s *Service) operationsApplication() (OperationsApplication, error) {
	return requiredApplication(s.applications.Operations, s.applications.Operations != nil)
}
