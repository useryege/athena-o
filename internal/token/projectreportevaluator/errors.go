package projectreportevaluator

import "fmt"

func errStoreRequired() error {
	return fmt.Errorf("token store is required for project-report-evaluator")
}
