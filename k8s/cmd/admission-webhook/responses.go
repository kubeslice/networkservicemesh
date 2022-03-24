package main

import (
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func errorReviewResponse(err error) *v1.AdmissionResponse {
	logrus.Error(err)
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func okReviewResponse() *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: true,
	}
}

func createReviewResponse(data []byte) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: true,
		Patch:   data,
		PatchType: func() *v1.PatchType {
			pt := v1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
