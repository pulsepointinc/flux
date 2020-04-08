package daemon

import (
	"context"
	"regexp"
	"time"

	"github.com/fluxcd/flux/pkg/metrics"
	"github.com/fluxcd/flux/pkg/resource"
	clusterResource "github.com/fluxcd/flux/pkg/cluster/kubernetes/resource"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type k8sResourceCheckFunc func(*K8sSyncBack, map[string]resource.Resource, *[]resource.ID) error

//
type K8sSyncBack struct {
	logger           log.Logger
	syncBackInterval time.Duration
	lastSyncBack     time.Time
	ignoredResources []*regexp.Regexp
	resourceManagers []k8sResourceCheckFunc
	kubeClient       *kubernetes.Clientset
}

const (
	k8sPod                = "Pod"
	k8sDeployment         = "Deployment"
	k8sStatefulSet        = "StatefulSet"
	k8sDaemonSet          = "DaemonSet"
	k8sReplicaSet         = "ReplicaSet"
	k8sCronJob            = "CronJob"
	k8sConfigMap          = "ConfigMap"
	k8sIngress            = "Ingress"
	k8sService            = "Service"
	k8sRoleBinding        = "RoleBinding"
	k8sRole               = "Role"
	k8sClusterRoleBinding = "ClusterRoleBinding"
	k8sClusterRole        = "ClusterRole"
	k8sServiceAccount     = "ServiceAccount"
	k8sNamespace          = "Namespace"
)

var (
	k8sManagers = map[string]k8sResourceCheckFunc{
		"Pod":                searchNonSyncedPods,
		"Deployment":         searchNonSyncedDeployments,
		"StatefulSet":        searchNonSyncedStatefulSets,
		"DaemonSet":          searchNonSyncedDaemonSets,
		"ReplicaSet":         searchNonSyncedReplicaSets,
		"CronJob":            searchNonSyncedCronJobs,
		"ConfigMap":          searchNonSyncedConfigMaps,
		"Ingress":            searchNonSyncedIngresses,
		"Service":            searchNonSyncedServices,
		"RoleBinding":        searchNonSyncedRoleBindings,
		"Role":               searchNonSyncedRoles,
		"ClusterRoleBinding": searchNonSyncedClusterRoleBindings,
		"ClusterRole":        searchNonSyncedClusterRoles,
		"ServiceAccount":     searchNonSyncedServiceAccounts,
		"Namespace":          searchNonSyncedNamespaces,
	}
	K8sManagerNames = []string{
		k8sPod,
		k8sDeployment,
		k8sStatefulSet,
		k8sDaemonSet,
		k8sReplicaSet,
		k8sCronJob,
		k8sConfigMap,
		k8sIngress,
		k8sService,
		k8sRoleBinding,
		k8sRole,
		k8sClusterRoleBinding,
		k8sClusterRole,
		k8sServiceAccount,
		k8sNamespace,
	}
)

func K8sSyncBackInit(logger log.Logger, kubeClient *kubernetes.Clientset, syncBackInterval time.Duration, syncBackTypes []string, syncBackIgnore []string) (*K8sSyncBack, error) {
	var err error
	ignoredResources := make([]*regexp.Regexp, len(syncBackIgnore), len(syncBackIgnore))
	for idx, ignoreStr := range syncBackIgnore {
		if ignoredResources[idx], err = regexp.Compile(ignoreStr); err != nil {
			return nil, errors.Wrapf(err, "compile regexp %v", ignoreStr)
		}
	}
	resourceManagers := make([]k8sResourceCheckFunc, len(syncBackTypes), len(syncBackTypes))
	for idx, typeStr := range syncBackTypes {
		var ok bool
		if resourceManagers[idx], ok = k8sManagers[typeStr]; !ok {
			return nil, errors.Wrapf(err, "resource not found %v", typeStr)
		}
	}

	sync := &K8sSyncBack{
		logger:           logger,
		syncBackInterval: syncBackInterval,
		ignoredResources: ignoredResources,
		resourceManagers: resourceManagers,
		kubeClient:       kubeClient,
	}
	return sync, nil
}

func (sync *K8sSyncBack) searchNonSynced(ctx context.Context, resources map[string]resource.Resource) {
	now := time.Now()
	if now.Sub(sync.lastSyncBack) < sync.syncBackInterval {
		return
	}
	sync.lastSyncBack = now

	notOwnedResources := sync.readNotOwnedResources(resources)

	for _, resourceID := range notOwnedResources {
		syncBackMetric.With(metrics.LabelName, resourceID.String()).Set(1)
	}
}

func (sync *K8sSyncBack) readNotOwnedResources(resources map[string]resource.Resource) []resource.ID {
	notOwnedResources := make([]resource.ID, 0, 0)
	for _, resourceManager := range sync.resourceManagers {
		if err := resourceManager(sync, resources, &notOwnedResources); err != nil {
			sync.logger.Log("Error fetching k8s resources", err)
		}
	}
	return notOwnedResources
}

func searchNonSyncedPods(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if podList, err := sync.kubeClient.CoreV1().Pods("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, pod := range podList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sPod, pod.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedConfigMaps(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if configMapList, err := sync.kubeClient.CoreV1().ConfigMaps("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, configMap := range configMapList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sConfigMap, configMap.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedDeployments(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if deploymentList, err := sync.kubeClient.AppsV1().Deployments("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, deployment := range deploymentList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sDeployment, deployment.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedStatefulSets(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if statefulSetList, err := sync.kubeClient.AppsV1().StatefulSets("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, statefulSet := range statefulSetList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sStatefulSet, statefulSet.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedDaemonSets(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if daemonSetList, err := sync.kubeClient.AppsV1().DaemonSets("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, daemonSet := range daemonSetList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sDaemonSet, daemonSet.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedReplicaSets(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if replicaSetList, err := sync.kubeClient.AppsV1().ReplicaSets("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, replicaSet := range replicaSetList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sReplicaSet, replicaSet.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedCronJobs(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if cronJobList, err := sync.kubeClient.BatchV1beta1().CronJobs("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, cronJob := range cronJobList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sCronJob, cronJob.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedIngresses(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if ingressList, err := sync.kubeClient.ExtensionsV1beta1().Ingresses("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, ingress := range ingressList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sIngress, ingress.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedServices(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if serviceList, err := sync.kubeClient.CoreV1().Services("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, service := range serviceList.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sService, service.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedRoleBindings(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if list, err := sync.kubeClient.RbacV1().RoleBindings("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, item := range list.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sRoleBinding, item.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedRoles(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if list, err := sync.kubeClient.RbacV1().Roles("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, item := range list.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sRole, item.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedClusterRoleBindings(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if list, err := sync.kubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, item := range list.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sClusterRoleBinding, item.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedClusterRoles(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if list, err := sync.kubeClient.RbacV1().ClusterRoles().List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, item := range list.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sClusterRole, item.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedServiceAccounts(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if list, err := sync.kubeClient.CoreV1().ServiceAccounts("").List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, item := range list.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sServiceAccount, item.ObjectMeta)
		}
	}
	return nil
}

func searchNonSyncedNamespaces(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *[]resource.ID) error {
	if list, err := sync.kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{}); err != nil {
		return err
	} else {
		for _, item := range list.Items {
			addIfNotOwned(sync, fluxResources, notOwnedResources, k8sNamespace, item.ObjectMeta)
		}
	}
	return nil
}

func addIfNotOwned(sync *K8sSyncBack, fluxResources map[string]resource.Resource, notOwnedResources *([]resource.ID), k8sType string, meta metav1.ObjectMeta) {
	ownerReferences := meta.OwnerReferences
	if len(ownerReferences) > 0 {
		return
	}
	namespace := meta.Namespace
	if len(namespace) == 0 {
		namespace = clusterResource.ClusterScope
	}
	resourceID := resource.MakeID(namespace, k8sType, meta.Name)
	resourceIdStr := resourceID.String()
	if _, ok := fluxResources[resourceIdStr]; !ok {
		for _, ignoreRegexp := range sync.ignoredResources {
			if ignoreRegexp.MatchString(resourceIdStr) {
				return
			}
		}
		*notOwnedResources = append(*notOwnedResources, resourceID)
	}
}
