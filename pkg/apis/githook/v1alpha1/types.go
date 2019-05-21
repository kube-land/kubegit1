package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitHook is a specification for a GitHook resource
type GitHook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHookSpec   `json:"spec"`
	Status GitHookStatus `json:"status"`
}

// GitHookSpec is the spec for a GitHook resource
type GitHookSpec struct {
	Repository            string   `json:"repository"`
  Branches              []string `json:"branches"`
  Manifest              string   `json:"manifest"`

	TimestampSuffix       bool `json:"timestampSuffix"`

	ArgoWorkflow          *ArgoWorkflowSpec `json:"argoWorkflow"`

	UsernameSecret        Secret `json:"usernameSecret"`
	PasswordSecret				Secret `json:"passwordSecret"`
  SshPrivateKeySecret   Secret `json:"sshPrivateKeySecret"`

	Notification NotificationSpec `json:"notification"`
}

// GitHookStatus is the status for a GitHook resource
type GitHookStatus struct {
	LastCommit       string       `json:"lastCommit"`
	Branch           string       `json:"branch"`
	Author           string       `json:"author"`
	AppliedResource  ResourceSpec `json:"appliedResource"`
	TriggerCount     int64        `json:"triggerCount"`
	LastTrigger      metav1.Time  `json:"lastTrigger,omitempty"`
}

// ResourceSpec is the spec of a k8s resource that is used by GitHook
type ResourceSpec struct {
	APIVersion   string `json:"apiVersion"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
}

// NotificationSpec is the resource that will be used for GitHook status and where it will be notified
type NotificationSpec struct {
	Github       string `json:"github"`
	Slack        string `json:"slack"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitHookList is a list of GitHook resources
type GitHookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []GitHook `json:"items"`
}

// ArgoWorkflowSpec is the spec for an ArgoWorkflow
type ArgoWorkflowSpec struct {
	RevisionParameterName string `json:"revisionParameterName"`
	BranchParameterName   string `json:"branchParameterName"`
}

// Secret is a secret type for the repository auth of a GitHook resource
type Secret struct {
  Name string `json:"name"`
  Key  string `json:"key"`
}
