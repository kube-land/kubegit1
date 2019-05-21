package webhook

import (
	"fmt"
	"net/http"
	"encoding/json"
	"bytes"
	"time"

	"k8s.io/klog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ghapi "github.com/appwavelets/kube-git/pkg/apis/githook/v1alpha1"
	ghclient "github.com/appwavelets/kube-git/pkg/client/clientset/versioned"

	"github.com/appwavelets/kube-git/pkg/controller"
	"github.com/appwavelets/kube-git/pkg/notification"
	"github.com/appwavelets/kube-git/pkg/git"
	"gopkg.in/go-playground/webhooks.v5/github"


	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/restmapper"
	discocache "k8s.io/client-go/discovery/cached"
	argo "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	batch "k8s.io/api/batch/v1"

)

type WebhookHandler struct {
  controller *controller.Controller
  clientset *kubernetes.Clientset
	wfClientset *wfclient.Clientset
	ghClientset *ghclient.Clientset
	hook *github.Webhook
}


func NewWebhookHandler(controller *controller.Controller, clientset *kubernetes.Clientset, wfClientset *wfclient.Clientset, ghClientset *ghclient.Clientset, secret string) WebhookHandler {

	hook, _ := github.New(github.Options.Secret(secret))

	return WebhookHandler{
		controller: controller,
		clientset: clientset,
		wfClientset: wfClientset,
		ghClientset: ghClientset,
		hook: hook,
	}
}

func (h WebhookHandler) GithubWebhook(w http.ResponseWriter, r *http.Request) {

		payload, err := h.hook.Parse(r, github.PushEvent, github.PingEvent)
		if err != nil {
			if err == github.ErrEventNotFound {

			} else {

			}
			w.WriteHeader(400)
			fmt.Fprintf(w, "%s", err)
			return
		}

		switch payload.(type) {

		case github.PushPayload:
			push := payload.(github.PushPayload)

			sshURL := push.Repository.SSHURL
			cloneURL := push.Repository.CloneURL
			branch := push.Ref
			hash := push.HeadCommit.ID
			author := push.HeadCommit.Author.Name

			// create status annotations
			annotations := make(map[string]string)
			annotations["kubegit.appwavelets.com/branch"] = branch
			annotations["kubegit.appwavelets.com/commit"] = hash
			annotations["kubegit.appwavelets.com/author"] = author

			// get GitHooks
			ghs := h.controller.GetGitHooks()
			if len(ghs) == 0 {
				klog.Info("No GitHooks")
				return
			}

			for _, gh := range ghs {
				// if url param match repository
				if gh.Spec.Repository == sshURL || gh.Spec.Repository == cloneURL {
					// if no matched branch, continue
					if !matchBranch(branch) {
						continue
					}

					ghFullname := gh.Namespace + "/" + gh.Name
					klog.Infof("Found GitHook for bitbucket payload: %s", ghFullname)

					annotations["kubegit.appwavelets.com/githook"] = ghFullname
					annotations["kubegit.appwavelets.com/repository"] = gh.Spec.Repository

					var username []byte
					var password []byte
					var sshKey []byte

					if gh.Spec.SshPrivateKeySecret.Name != "" {
						sshPrivateKeySecret, err := h.clientset.CoreV1().Secrets(gh.Namespace).Get(gh.Spec.SshPrivateKeySecret.Name, metav1.GetOptions{})
						if err != nil {
							klog.Errorf("Error getting secret of %s GitHook: %s", ghFullname, err.Error())
							continue
						}
						sshKey = sshPrivateKeySecret.Data[gh.Spec.SshPrivateKeySecret.Key]
					}

					// getting username and password from Secrets
					if gh.Spec.UsernameSecret.Name != "" && gh.Spec.PasswordSecret.Name != "" {
						usernameSecret, err := h.clientset.CoreV1().Secrets(gh.Namespace).Get(gh.Spec.UsernameSecret.Name, metav1.GetOptions{})
						if err != nil {
							klog.Errorf("Error getting secret of %s GitHook: %s", ghFullname, err.Error())
							continue
						}
						passwordSecret, err := h.clientset.CoreV1().Secrets(gh.Namespace).Get(gh.Spec.PasswordSecret.Name, metav1.GetOptions{})
						if err != nil {
							klog.Errorf("Error getting secret of %s GitHook: %s", ghFullname, err.Error())
							continue
						}
						username = usernameSecret.Data[gh.Spec.UsernameSecret.Key]
						password = passwordSecret.Data[gh.Spec.PasswordSecret.Key]
					}

					manifest, err := git.FetchGitFile(gh.Spec.Repository, username, password, sshKey, hash, gh.Spec.Manifest)
					if err != nil {
						klog.Errorf("Error Fetch files from git repository (%s): %s", gh.Spec.Repository, err.Error())
						continue
					}
					klog.Infof("Applying GitHook of bitbucket payload: %s", ghFullname)
					// Apply Manifest
					go h.ApplyGitHook(manifest, gh, annotations)
				}
			}

		case github.PingPayload:
			w.WriteHeader(200)
			return
		}

}

func (h WebhookHandler) ApplyGitHook(manifest []byte, gh *ghapi.GitHook, annotations map[string]string) {

	ghFullname := annotations["kubegit.appwavelets.com/githook"]

	var appliedResource ghapi.ResourceSpec

	dis := h.clientset.Discovery()
	cachedDiscovery := discocache.NewMemCacheClient(dis)
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)
	restMapper.Reset()

	reader := bytes.NewReader(manifest)
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)

	ext := runtime.RawExtension{}
	if err := decoder.Decode(&ext); err != nil {
		klog.Errorf("Error decoding manifest of GitHook (%s): %s", ghFullname, err.Error())
		return
	}

	versions := &runtime.VersionedObjects{}
	_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, versions)
	if err != nil {
		klog.Errorf("Error decoding manifest of GitHook (%s): %s", ghFullname, err.Error())
		return
	}

	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		klog.Errorf("Error in RESTMapping of GitHook (%s): %s", ghFullname, err.Error())
		return
	}

	// if type is Workflow
	if (mapping.GroupVersionKind.Group == "argoproj.io" && mapping.GroupVersionKind.Version == "v1alpha1" && mapping.GroupVersionKind.Kind == "Workflow") {

		var workflow argo.Workflow
		if err := json.Unmarshal(ext.Raw, &workflow); err != nil {
			klog.Errorf("Error parsing argo workflow of GitHook (%s): %s", ghFullname, err.Error())
			return
		}

		// Set Revision Parameter
		if gh.Spec.ArgoWorkflow != nil {
			if gh.Spec.ArgoWorkflow.RevisionParameterName != "" {
				for _, p := range workflow.Spec.Arguments.Parameters {
					if p.Name == gh.Spec.ArgoWorkflow.RevisionParameterName {
						*p.Value = annotations["kubegit.appwavelets.com/commit"]
					}
					if p.Name == gh.Spec.ArgoWorkflow.BranchParameterName {
						*p.Value = annotations["kubegit.appwavelets.com/branch"]
					}
				}
			}
		}

		// set namespace
		ns := "default"
		if workflow.Namespace != "" {
			ns = workflow.Namespace
		}

		if workflow.ObjectMeta.Annotations == nil {
			workflow.ObjectMeta.Annotations = make(map[string]string)
		}
		for k, v := range notification.GetNotificationAnnotations(gh) {
			workflow.ObjectMeta.Annotations[k] = v
		}
		for k, v := range annotations {
			workflow.ObjectMeta.Annotations[k] = v
		}

		// create Workflow
		result, err := h.wfClientset.ArgoprojV1alpha1().Workflows(ns).Create(&workflow)
		if err != nil {
			klog.Errorf("Error RESTMapping of GitHook (%s): %s", ghFullname, err.Error())
			return
		}

		appliedResource = ghapi.ResourceSpec{
			APIVersion: mapping.GroupVersionKind.Group + "/" + mapping.GroupVersionKind.Version,
			Kind: mapping.GroupVersionKind.Kind,
			Name: result.Name,
			Namespace: result.Namespace,
		}

	} else if (mapping.GroupVersionKind.Group == "batch" && mapping.GroupVersionKind.Version == "v1" && mapping.GroupVersionKind.Kind == "Job") {

		var job batch.Job
		if err := json.Unmarshal(ext.Raw, &job); err != nil {
			klog.Errorf("Error parsing job of GitHook (%s): %s", ghFullname, err.Error())
			return
		}

		// set namespace
		ns := "default"
		if job.Namespace != "" {
			ns = job.Namespace
		}

		if job.ObjectMeta.Annotations == nil {
			job.ObjectMeta.Annotations = make(map[string]string)
		}
		for k, v := range notification.GetNotificationAnnotations(gh) {
			job.ObjectMeta.Annotations[k] = v
		}
		for k, v := range annotations {
			job.ObjectMeta.Annotations[k] = v
		}

		// create Workflow
		result, err := h.clientset.BatchV1().Jobs(ns).Create(&job)
		if err != nil {
			klog.Errorf("Error RESTMapping of GitHook (%s): %s", ghFullname, err.Error())
			return
		}

		appliedResource = ghapi.ResourceSpec{
			APIVersion: mapping.GroupVersionKind.Group + "/" + mapping.GroupVersionKind.Version,
			Kind: mapping.GroupVersionKind.Kind,
			Name: result.Name,
			Namespace: result.Namespace,
		}

	}

	h.UpdateGitHook(gh, annotations, appliedResource)
}

func (h WebhookHandler) UpdateGitHook(gh *ghapi.GitHook, annotations map[string]string, appliedResource ghapi.ResourceSpec) {
	ghFullname := gh.Namespace + "/" + gh.Name
	gh.Status.LastCommit = annotations["kubegit.appwavelets.com/commit"]
	gh.Status.Author = annotations["kubegit.appwavelets.com/author"]
	gh.Status.Branch = annotations["kubegit.appwavelets.com/branch"]
	gh.Status.TriggerCount = gh.Status.TriggerCount + 1
	gh.Status.AppliedResource = appliedResource
	gh.Status.LastTrigger = metav1.Time{Time: time.Now().UTC()}
	_, err := h.ghClientset.KubegitV1alpha1().GitHooks(gh.Namespace).UpdateStatus(gh)
	if err != nil {
		klog.Errorf("Error updating status of GitHook (%s): %s", ghFullname, err.Error())
	}
}

// need to be modified
func matchBranch(branch string) bool {
	return true
}

/*
eventBroadcaster := record.NewBroadcaster()
eventBroadcaster.StartLogging(klog.Infof)
eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: h.client.CoreV1().Events("")})
recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "controllerAgentName"})
recorder.Event(gh, corev1.EventTypeWarning, "ErrResourceExists", "msg")
*/
