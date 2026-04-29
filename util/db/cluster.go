package db

// import (
// 	"encoding/json"
// 	"fmt"
// 	"maps"
// 	"strconv"
// 	"strings"
// 	"time"

// 	log "github.com/sirupsen/logrus"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/utils/ptr"

// 	"github.com/useryege/athena/common"
// 	appv1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
// )

// // SecretToCluster converts a secret into a Cluster object
// func SecretToCluster(s *corev1.Secret) (*appv1.Cluster, error) {
// 	var config appv1.ClusterConfig
// 	if len(s.Data["config"]) > 0 {
// 		err := json.Unmarshal(s.Data["config"], &config)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal cluster config: %w", err)
// 		}
// 	}

// 	var namespaces []string
// 	for _, ns := range strings.Split(string(s.Data["namespaces"]), ",") {
// 		if ns = strings.TrimSpace(ns); ns != "" {
// 			namespaces = append(namespaces, ns)
// 		}
// 	}
// 	var refreshRequestedAt *metav1.Time
// 	if v, found := s.Annotations[appv1.AnnotationKeyRefresh]; found {
// 		requestedAt, err := time.Parse(time.RFC3339, v)
// 		if err != nil {
// 			log.Warnf("Error while parsing date in cluster secret '%s': %v", s.Name, err)
// 		} else {
// 			refreshRequestedAt = &metav1.Time{Time: requestedAt}
// 		}
// 	}
// 	var shard *int64
// 	if shardStr := s.Data["shard"]; shardStr != nil {
// 		if val, err := strconv.Atoi(string(shardStr)); err != nil {
// 			log.Warnf("Error while parsing shard in cluster secret '%s': %v", s.Name, err)
// 		} else {
// 			shard = ptr.To(int64(val))
// 		}
// 	}

// 	// copy labels and annotations excluding system ones
// 	labels := map[string]string{}
// 	if s.Labels != nil {
// 		labels = maps.Clone(s.Labels)
// 		delete(labels, common.LabelKeySecretType)
// 	}
// 	annotations := map[string]string{}
// 	if s.Annotations != nil {
// 		annotations = maps.Clone(s.Annotations)
// 		// delete system annotations
// 		delete(annotations, corev1.LastAppliedConfigAnnotation)
// 		delete(annotations, common.AnnotationKeyManagedBy)
// 	}

// 	cluster := appv1.Cluster{
// 		ID:                 string(s.UID),
// 		Server:             strings.TrimRight(string(s.Data["server"]), "/"),
// 		Name:               string(s.Data["name"]),
// 		Namespaces:         namespaces,
// 		ClusterResources:   string(s.Data["clusterResources"]) == "true",
// 		Config:             config,
// 		RefreshRequestedAt: refreshRequestedAt,
// 		Shard:              shard,
// 		Project:            string(s.Data["project"]),
// 		Labels:             labels,
// 		Annotations:        annotations,
// 	}
// 	return &cluster, nil
// }
