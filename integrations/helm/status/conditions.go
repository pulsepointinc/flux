package status

import (
	"encoding/json"

	"github.com/weaveworks/flux/integrations/apis/flux.weave.works/v1beta1"
	v1beta1client "github.com/weaveworks/flux/integrations/client/clientset/versioned/typed/flux.weave.works/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

// We can't rely on having UpdateStatus, or strategic merge patching
// for custom resources. So we have to create an object which
// represents the merge path or JSON patch to apply.
func UpdateConditionsPatch(oldStatus v1beta1.FluxHelmReleaseStatus, updates ...v1beta1.FluxHelmReleaseCondition) (types.PatchType, interface{}) {
	newConditions := make([]v1beta1.FluxHelmReleaseCondition, len(oldStatus.Conditions))
	for i, c := range oldStatus.Conditions {
		newConditions[i] = c
	}
updates:
	for _, up := range updates {
		for i, c := range oldStatus.Conditions {
			if c.Type == up.Type {
				newConditions[i] = up
				continue updates
			}
		}
		newConditions = append(newConditions, up)
	}
	return types.MergePatchType, map[string]interface{}{
		"status": map[string]interface{}{
			"conditions": newConditions,
		},
	}
}

func UpdateConditions(client v1beta1client.FluxHelmReleaseInterface, fhr v1beta1.FluxHelmRelease, updates ...v1beta1.FluxHelmReleaseCondition) error {
	t, obj := UpdateConditionsPatch(fhr.Status, updates...)
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = client.Patch(fhr.Name, t, bytes)
	return err
}
