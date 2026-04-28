package argo

import (
	"sort"

	"github.com/useryege/athena/gitops-engine/pkg/utils/kube"
)

// APIResourcesToStrings converts list of API Resources list into string list
func APIResourcesToStrings(resources []kube.APIResourceInfo, includeKinds bool) []string {
	resMap := map[string]bool{}
	for _, r := range resources {
		groupVersion := r.GroupVersionResource.GroupVersion().String()
		resMap[groupVersion] = true
		if includeKinds {
			resMap[groupVersion+"/"+r.GroupKind.Kind] = true
		}
	}
	var res []string
	for k := range resMap {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	return res
}
