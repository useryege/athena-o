package application

type ProjectSimulator interface {
}

var _ ProjectSimulator = &projectSimulatorImpl{}

type projectSimulatorImpl struct {
}

func NewProjectSimulator() ProjectSimulator {
	return &projectSimulatorImpl{}
}
