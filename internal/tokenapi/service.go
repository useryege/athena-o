package tokenapi

import (
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedTokenCatalogServiceServer
	apiclient.UnimplementedTokenResearchServiceServer
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
func (s *Service) researchApplication() (ResearchApplication, error) {
	return requiredApplication(s.applications.Research, s.applications.Research != nil)
}
func (s *Service) reportingApplication() (ReportingApplication, error) {
	return requiredApplication(s.applications.Reporting, s.applications.Reporting != nil)
}
func (s *Service) selectionApplication() (SelectionApplication, error) {
	return requiredApplication(s.applications.Selection, s.applications.Selection != nil)
}
func (s *Service) policyApplication() (PolicyApplication, error) {
	return requiredApplication(s.applications.Policy, s.applications.Policy != nil)
}
func (s *Service) operationsApplication() (OperationsApplication, error) {
	return requiredApplication(s.applications.Operations, s.applications.Operations != nil)
}
