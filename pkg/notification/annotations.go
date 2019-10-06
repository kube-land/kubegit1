package notification

import (
	ghapi "github.com/appspero/kube-git/pkg/apis/githook/v1alpha1"
)

func GetNotificationAnnotations(gh *ghapi.GitHook) map[string]string {
	annotations := make(map[string]string)

	if gh.Spec.Notification.Slack != "" {
		annotations["kubegit.appspero.com/slack"] = gh.Spec.Notification.Slack
	}
	if gh.Spec.Notification.Github != "" {
		annotations["kubegit.appspero.com/github"] = gh.Spec.Notification.Github
	}

	if len(annotations) > 0 {
		annotations["kubegit.appspero.com/started"] = "false"
	}

	return annotations
}


func ShouldNotify(annotations map[string]string) bool {

  if annotations == nil {
    return false
  }
  if _, ok := annotations["kubegit.appspero.com/github"]; ok {
    return true
  }
  if _, ok := annotations["kubegit.appspero.com/slack"]; ok {
    return true
  }
  return false
}

func ShouldNotifyStarted(annotations map[string]string) bool {

  if annotations == nil {
    return false
  }
  if _, ok := annotations["kubegit.appspero.com/started"]; ok {
    return true
  }
  return false
}

type PatchAnnotations struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
}

func NewRemoveNotificationPatch(annotations map[string]string) []PatchAnnotations {

	var patch []PatchAnnotations

	if _, ok := annotations["kubegit.appspero.com/github"]; ok {
		patch = append(patch, PatchAnnotations{
			Op:    "remove",
	    Path:  "/metadata/annotations/kubegit.appspero.com~1github",
	   })
	}

	if _, ok := annotations["kubegit.appspero.com/slack"]; ok {
		patch = append(patch, PatchAnnotations{
	  	Op:    "remove",
	    Path:  "/metadata/annotations/kubegit.appspero.com~1slack",
	  })
	}

  return patch
}

func NewRemoveStartedPatch() []PatchAnnotations {
  return []PatchAnnotations{
    {
      Op:    "remove",
      Path:  "/metadata/annotations/kubegit.appspero.com~1started",
    },
  }
}
