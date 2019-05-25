package controller

import (
  "fmt"
  "time"

  "k8s.io/klog"
  "k8s.io/client-go/tools/cache"
  "k8s.io/apimachinery/pkg/fields"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

  batch "k8s.io/api/batch/v1"

  argo "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

  ghapi "github.com/appwavelets/kube-git/pkg/apis/githook/v1alpha1"
  ghclient "github.com/appwavelets/kube-git/pkg/client/clientset/versioned"
	wfclient "github.com/argoproj/argo/pkg/client/clientset/versioned"

  "k8s.io/client-go/util/workqueue"
  "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
  "k8s.io/client-go/kubernetes"

  "github.com/appwavelets/kube-git/pkg/notification"

  "encoding/json"
  types "k8s.io/apimachinery/pkg/types"

)

const (
	resyncPeriod = 15 * time.Minute
	maxRetries   = 5
)

type Controller struct {
  clientset   kubernetes.Interface
  jobInformer cache.SharedIndexInformer

  wfClientset wfclient.Interface
  wfInformer  cache.SharedIndexInformer

  ghClientset ghclient.Interface
  ghInformer  cache.SharedIndexInformer

  queue workqueue.RateLimitingInterface

  notification *notification.Config
}

type Task struct {
	Key    string
  Action string
  Type   string
}

func NewController(clientset kubernetes.Interface, wfClientset wfclient.Interface, ghClientset ghclient.Interface, notificationConfig *notification.Config) *Controller {

    queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

    jobListwatch := cache.NewListWatchFromClient(clientset.BatchV1().RESTClient(), "jobs", metav1.NamespaceAll, fields.Everything())
    jobInformer := cache.NewSharedIndexInformer(
  		jobListwatch,
  		&batch.Job{},
  		resyncPeriod,
  		cache.Indexers{},
  	)
    jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
      AddFunc: func(obj interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(obj)
  			if err == nil {
  				job := obj.(*batch.Job)
  				if notification.ShouldNotifyStarted(job.ObjectMeta.Annotations) {
  					queue.AddRateLimited(Task{
  						Key:  key,
              Action: "CREATE",
  						Type: "Job",
  					})
  				}
  			} else {
  				runtime.HandleError(err)
  				return
  			}
      },
      UpdateFunc: func(old, new interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(new)
  			if err == nil {
  				job := new.(*batch.Job)
  				if notification.ShouldNotify(job.ObjectMeta.Annotations) {
  					queue.AddRateLimited(Task{
  						Key:  key,
              Action: "UPDATE",
  						Type: "Job",
  					})
  				}
  			} else {
  				runtime.HandleError(err)
  				return
  			}
  		},
	  })

    wfListwatch := cache.NewListWatchFromClient(wfClientset.ArgoprojV1alpha1().RESTClient(), "workflows", metav1.NamespaceAll, fields.Everything())
    wfInformer := cache.NewSharedIndexInformer(
  		wfListwatch,
  		&argo.Workflow{},
  		resyncPeriod,
  		cache.Indexers{},
  	)
    wfInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
      AddFunc: func(obj interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(obj)
  			if err == nil {
  				wf := obj.(*argo.Workflow)
  				if notification.ShouldNotifyStarted(wf.ObjectMeta.Annotations) {
  					queue.AddRateLimited(Task{
  						Key:  key,
              Action: "CREATE",
  						Type: "Workflow",
  					})
  				}
  			} else {
  				runtime.HandleError(err)
  				return
  			}
      },
      UpdateFunc: func(old, new interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(new)
  			if err == nil {
  				wf := new.(*argo.Workflow)
  				if notification.ShouldNotify(wf.ObjectMeta.Annotations) {
  					queue.AddRateLimited(Task{
  						Key:  key,
              Action: "UPDATE",
  						Type: "Workflow",
  					})
  				}
  			} else {
  				runtime.HandleError(err)
  				return
  			}
  		},
	  })

    ghListwatch := cache.NewListWatchFromClient(ghClientset.KubegitV1alpha1().RESTClient(), "githooks", metav1.NamespaceAll, fields.Everything())
    ghInformer := cache.NewSharedIndexInformer(
  		ghListwatch,
  		&ghapi.GitHook{},
  		resyncPeriod,
  		cache.Indexers{},
  	)

  	return &Controller{
      clientset: clientset,
      jobInformer: jobInformer,
      wfClientset: wfClientset,
      wfInformer: wfInformer,
  		ghClientset: ghClientset,
      ghInformer: ghInformer,
      queue: queue,
      notification: notificationConfig,
  	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) error {

  defer runtime.HandleCrash()
	//defer c.queue.ShutDown()

  klog.Info("Starting kube-git controller")

  go c.jobInformer.Run(stopCh)
  go c.wfInformer.Run(stopCh)
  go c.ghInformer.Run(stopCh)

	klog.Info("Waiting for caches to sync")
  if ok := cache.WaitForCacheSync(stopCh, c.jobInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for jobs caches to sync")
	}
	if ok := cache.WaitForCacheSync(stopCh, c.wfInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for workflows caches to sync")
	}
  if ok := cache.WaitForCacheSync(stopCh, c.ghInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for githooks caches to sync")
	}

  klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

  return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNext() {
	}
}

// processNext will read a single work item off the workqueue and
// attempt to process it, by calling the process.
func (c *Controller) processNext() bool {
	key, quit := c.queue.Get()

	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.process(key.(Task))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < maxRetries {
		klog.Infof("Error processing %s (will retry): %v", key, err)
		c.queue.AddRateLimited(key)
	} else {
		// err != nil and too many retries
		klog.Errorf("Error processing %s (giving up): %v", key, err)
		c.queue.Forget(key)
		runtime.HandleError(err)
	}

	return true
}

func (c *Controller) process(task Task) error {

  if task.Type == "Job" {

    obj, exists, err := c.jobInformer.GetIndexer().GetByKey(task.Key)
  	if err != nil {
  		return fmt.Errorf("failed to retrieve job by key %q: %v", task.Key, err)
  	}
    if exists {
      job := obj.(*batch.Job)
      if task.Action == "CREATE" {
        c.notification.Notify("STARTED", "Job", job.ObjectMeta.Namespace, job.ObjectMeta.Name, job.ObjectMeta.Annotations)
        return c.jobRemoveStarted(job.ObjectMeta.Namespace, job.ObjectMeta.Name)
      } else {
        for _, condition := range job.Status.Conditions {
          if condition.Type == "Failed" {
            c.notification.Notify("FAILED", "Job", job.ObjectMeta.Namespace, job.ObjectMeta.Name, job.ObjectMeta.Annotations)
            return c.jobRemoveNotification(job.ObjectMeta.Namespace, job.ObjectMeta.Name, job.ObjectMeta.Annotations)
          } else if condition.Type == "Complete" {
            c.notification.Notify("SUCCEEDED", "Job", job.ObjectMeta.Namespace, job.ObjectMeta.Name, job.ObjectMeta.Annotations)
            return c.jobRemoveNotification(job.ObjectMeta.Namespace, job.ObjectMeta.Name, job.ObjectMeta.Annotations)
          }
        }
      }
    }

  } else if task.Type == "Workflow" {

    obj, exists, err := c.wfInformer.GetIndexer().GetByKey(task.Key)
  	if err != nil {
  		return fmt.Errorf("failed to retrieve workflow by key %q: %v", task.Key, err)
  	}
    if exists {
      wf := obj.(*argo.Workflow)
      if task.Action == "CREATE" {
        c.notification.Notify("STARTED", "Argo Workflow", wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, wf.ObjectMeta.Annotations)
        return c.wfRemoveStarted(wf.ObjectMeta.Namespace, wf.ObjectMeta.Name)
      } else {
        if wf.Status.Phase == "Failed" {
          c.notification.Notify("FAILED", "Argo Workflow", wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, wf.ObjectMeta.Annotations)
          return c.wfRemoveNotification(wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, wf.ObjectMeta.Annotations)
        } else if wf.Status.Phase == "Succeeded" {
          c.notification.Notify("SUCCEEDED", "Argo Workflow", wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, wf.ObjectMeta.Annotations)
          return c.wfRemoveNotification(wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, wf.ObjectMeta.Annotations)
        }
      }
    }

  }

	return nil

}

func (c *Controller) GetGitHooks() []*ghapi.GitHook {
  ghObjects := c.ghInformer.GetIndexer().List()
  var ghs []*ghapi.GitHook
  for _, gh := range ghObjects {
    ghs = append(ghs, gh.(*ghapi.GitHook))
  }
	return ghs
}

func (c *Controller) jobRemoveNotification(ns string, job string, annotations map[string]string) error {
	payloadBytes, _ := json.Marshal(notification.NewRemoveNotificationPatch(annotations))
	_, err := c.clientset.BatchV1().Jobs(ns).Patch(job, types.JSONPatchType, payloadBytes)
	return err
}

func (c *Controller) wfRemoveNotification(ns string, wf string, annotations map[string]string) error {
	payloadBytes, _ := json.Marshal(notification.NewRemoveNotificationPatch(annotations))
	_, err := c.wfClientset.ArgoprojV1alpha1().Workflows(ns).Patch(wf, types.JSONPatchType, payloadBytes)
	return err
}

func (c *Controller) jobRemoveStarted(ns string, job string) error {
	payloadBytes, _ := json.Marshal(notification.NewRemoveStartedPatch())
	_, err := c.clientset.BatchV1().Jobs(ns).Patch(job, types.JSONPatchType, payloadBytes)
	return err
}

func (c *Controller) wfRemoveStarted(ns string, wf string) error {
	payloadBytes, _ := json.Marshal(notification.NewRemoveStartedPatch())
	_, err := c.wfClientset.ArgoprojV1alpha1().Workflows(ns).Patch(wf, types.JSONPatchType, payloadBytes)
	return err
}
