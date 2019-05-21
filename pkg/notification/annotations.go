package notification

import (
	ghapi "github.com/appwavelets/kube-git/pkg/apis/githook/v1alpha1"
)

func GetNotificationAnnotations(gh *ghapi.GitHook) map[string]string {
	annotations := make(map[string]string)

	if gh.Spec.Notification.Slack != "" {
		annotations["kubegit.appwavelets.com/slack"] = gh.Spec.Notification.Slack
	}
	if gh.Spec.Notification.Github != "" {
		annotations["kubegit.appwavelets.com/github"] = gh.Spec.Notification.Github
	}

	if len(annotations) > 0 {
		annotations["kubegit.appwavelets.com/started"] = "false"
	}

	return annotations
}


func ShouldNotify(annotations map[string]string) bool {

  if annotations == nil {
    return false
  }
  if _, ok := annotations["kubegit.appwavelets.com/github"]; ok {
    return true
  }
  if _, ok := annotations["kubegit.appwavelets.com/slack"]; ok {
    return true
  }
  return false
}

func ShouldNotifyStarted(annotations map[string]string) bool {

  if annotations == nil {
    return false
  }
  if _, ok := annotations["kubegit.appwavelets.com/started"]; ok {
    return true
  }
  return false
}

type PatchAnnotations struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
}

func NewRemoveNotificationPatch() []PatchAnnotations {
  return []PatchAnnotations{
    {
      Op:    "remove",
      Path:  "/metadata/annotations/kubegit.appwavelets.com~1github",
    },
    {
      Op:    "remove",
      Path:  "/metadata/annotations/kubegit.appwavelets.com~1slack",
    },
  }
}

func NewRemoveStartedPatch() []PatchAnnotations {
  return []PatchAnnotations{
    {
      Op:    "remove",
      Path:  "/metadata/annotations/kubegit.appwavelets.com~1started",
    },
  }
}
