package main

import (
	"encoding/json"
	"fmt"

	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mutate(request v1beta1.AdmissionRequest) (v1beta1.AdmissionResponse, error) {
	response := v1beta1.AdmissionResponse{}

	// Default response
	response.Allowed = true
	response.UID = request.UID

	// Decode the pod object
	var err error
	pod := v1.Pod{}
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		return response, fmt.Errorf("unable to decode Pod %w", err)
	}

	// Check for a GPU
	hasGPU := false
	for _, container := range pod.Spec.Containers {
		// if container.Resources.Requests.
		if limit, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
			if !limit.IsZero() {
				hasGPU = true
				break
			}
		}
	}

	if hasGPU {
		patch := v1beta1.PatchTypeJSONPatch
		response.PatchType = &patch

		response.AuditAnnotations = map[string]string{
			"gpu-admission-controller": "Added dedicated=gpu toleration",
		}

		toleration := v1.Toleration{
			Key:      "dedicated",
			Value:    "gpu",
			Operator: v1.TolerationOpEqual,
			Effect:   v1.TaintEffectNoSchedule,
		}

		patches := []map[string]interface{}{
			{
				"op":    "add",
				"path":  "/spec/tolerations/-",
				"value": toleration,
			},
		}
		response.Patch, err = json.Marshal(patches)
		if err != nil {
			return response, err
		}

		response.Result = &metav1.Status{
			Status: metav1.StatusSuccess,
		}
	}

	return response, nil
}
