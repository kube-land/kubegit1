package main

import (
	"net/http"
	"flag"
	"os"
	"fmt"
	"k8s.io/klog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ghclient "github.com/appwavelets/kube-git/pkg/client/clientset/versioned"
	wfclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/appwavelets/kube-git/pkg/webhook"
	"github.com/appwavelets/kube-git/pkg/controller"
	"github.com/appwavelets/kube-git/pkg/notification"
)

var (
	masterURL   string
	kubeconfig  string

	webhookPort             = flag.Int("webhook-port", 8080, "Service port of the webhook server.")
	githubWebhookSecret     = flag.String("github-webhook-secret", "", "Secret of the GitHub to be used with webhook server.")
	notificationConfigFile  = flag.String("notification-config-file", "/etc/kube-git/notification.yaml", "File containing the metadata configuration.")
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func main() {

	//klog.InitFlags(nil)
	//flag.Set("logtostderr", "true")
 	flag.Set("alsologtostderr", "true")
	flag.Set("stderrthreshold", "info")
	flag.Set("v", "2")

	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			f2.Value.Set(value)
		}
	})

	githubWebhookSecretEnv := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if githubWebhookSecretEnv != "" {
		*githubWebhookSecret = githubWebhookSecretEnv
	}

	notificationConfig, err := notification.LoadConfig(*notificationConfigFile)
	if err != nil {
		klog.Fatalf("Filed to load configuration: %v", err)
	}

  cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	wfClientset, err := wfclient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building Argo clientset: %s", err.Error())
	}

	ghClientset, err := ghclient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building GitHook clientset: %s", err.Error())
	}


  stopCh := make(chan struct{})
  controller := controller.NewController(clientset, wfClientset, ghClientset, notificationConfig)
	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

	handler := webhook.NewWebhookHandler(controller, clientset, wfClientset, ghClientset, *githubWebhookSecret)

	http.HandleFunc("/github", handler.GithubWebhook)

	port := fmt.Sprintf(":%d", *webhookPort)

	klog.Infof("Starting kube-git webhook at port: %d", *webhookPort)
	klog.Fatal(http.ListenAndServe(port, nil))
}
