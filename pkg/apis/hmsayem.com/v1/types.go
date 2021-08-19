package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//_ "k8s.io/code-generator"
)

// +genclient
// +groupName=hmsayem.com
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Server describes a Server.
type Server struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServerSpec   `json:"spec"`
	Status            ServerStatus `json:"status"`
}

// ServerSpec is the spec for a Server resource
type ServerSpec struct {
	StatefulSetName string `json:"statefulSetName"`
	Replicas        *int32 `json:"replicas"`
}
type ServerStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ServerList is a list of Server resources
type ServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Server `json:"items"`
}
