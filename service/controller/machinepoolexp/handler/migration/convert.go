package migration

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func cloneObjectMeta(objectMeta metav1.ObjectMeta) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            objectMeta.Name,
		Namespace:       objectMeta.Namespace,
		Labels:          objectMeta.Labels,
		Annotations:     objectMeta.Annotations,
		OwnerReferences: objectMeta.OwnerReferences,
		Finalizers:      objectMeta.Finalizers,
		ClusterName:     objectMeta.ClusterName,
	}
}

func convertSpec(oldSpec interface{}, newSpec interface{}) error {
	oldJson, err := json.Marshal(oldSpec)
	if err != nil {
		return err
	}

	err = json.Unmarshal(oldJson, newSpec)
	if err != nil {
		return err
	}

	return nil
}
