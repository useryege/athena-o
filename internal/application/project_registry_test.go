package application

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

func TestProjectRegistryListProjectContractsUsesCachedRefs(t *testing.T) {
	registry := NewProjectRegistry(nil)
	projectA := newRegistryTestProject("0x0000000000000000000000000000000000000011")
	projectB := newRegistryTestProject("0x0000000000000000000000000000000000000022")

	if err := registry.SetProject(context.Background(), projectA.Meta.ProjectID, projectA); err != nil {
		t.Fatalf("set project A: %v", err)
	}
	if err := registry.SetProject(context.Background(), projectB.Meta.ProjectID, projectB); err != nil {
		t.Fatalf("set project B: %v", err)
	}

	refs, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("refs length = %d, want 2", len(refs))
	}
	assertProjectContractRef(t, refs, projectA)
	assertProjectContractRef(t, refs, projectB)
}

func TestProjectRegistryListProjectContractsReturnsCopy(t *testing.T) {
	registry := NewProjectRegistry(nil)
	project := newRegistryTestProject("0x0000000000000000000000000000000000000033")
	if err := registry.SetProject(context.Background(), project.Meta.ProjectID, project); err != nil {
		t.Fatalf("set project: %v", err)
	}

	refs, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}
	refs[0] = ProjectContractRef{}

	refs, err = registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts again: %v", err)
	}
	assertProjectContractRef(t, refs, project)
}

func TestProjectRegistryRemoveProjectMaintainsCachedRefs(t *testing.T) {
	registry := NewProjectRegistry(nil)
	projectA := newRegistryTestProject("0x0000000000000000000000000000000000000044")
	projectB := newRegistryTestProject("0x0000000000000000000000000000000000000055")
	projectC := newRegistryTestProject("0x0000000000000000000000000000000000000066")

	for _, project := range []*Project{projectA, projectB, projectC} {
		if err := registry.SetProject(context.Background(), project.Meta.ProjectID, project); err != nil {
			t.Fatalf("set project %s: %v", project.Meta.ProjectID, err)
		}
	}
	if err := registry.RemoveProject(context.Background(), projectB.Meta.ProjectID); err != nil {
		t.Fatalf("remove project B: %v", err)
	}

	refs, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("refs length = %d, want 2", len(refs))
	}
	assertProjectContractRef(t, refs, projectA)
	assertProjectContractRef(t, refs, projectC)
	for _, ref := range refs {
		if ref.ProjectID == projectB.Meta.ProjectID {
			t.Fatal("removed project still present in contract refs")
		}
	}
}

func newRegistryTestProject(contract string) *Project {
	return &Project{
		Meta: ProjectMeta{
			ProjectID: uuid.New(),
			Contract:  common.HexToAddress(contract),
			Creator:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		},
	}
}

func assertProjectContractRef(t *testing.T, refs []ProjectContractRef, project *Project) {
	t.Helper()
	for _, ref := range refs {
		if ref.ProjectID == project.Meta.ProjectID {
			if ref.Contract != project.Meta.Contract {
				t.Fatalf("contract for %s = %s, want %s", project.Meta.ProjectID, ref.Contract, project.Meta.Contract)
			}
			if ref.Creator != project.Meta.Creator {
				t.Fatalf("creator for %s = %s, want %s", project.Meta.ProjectID, ref.Creator, project.Meta.Creator)
			}
			return
		}
	}
	t.Fatalf("project %s not found in refs", project.Meta.ProjectID)
}
