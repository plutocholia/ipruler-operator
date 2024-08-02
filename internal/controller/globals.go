package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/go-logr/logr"
	"github.com/plutocholia/ipruler-controller/internal/models"
	"github.com/plutocholia/ipruler-controller/internal/utils"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SharedFullConfig struct {
	Mutex                  sync.Mutex
	ClusterConfigName      string
	ClusterConfigNamespace string
}

type AgentManager struct {
	Port          int
	UpdatePath    string
	Namespace     string
	AppLabelKey   string
	AppLabelValue string
	Log           logr.Logger
}

var (
	globalAgentManager *AgentManager
	sharedFullConfig   *SharedFullConfig
)

func (mgr *AgentManager) InjectConfig(pod *corev1.Pod, config *models.ConfigModel) {
	mgr.Log.Info("Injecting config file to", "pod", pod.Name)
	url := fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, mgr.Port, mgr.UpdatePath)

	configYaml, _ := utils.ConvertToYAML(config)

	// TODO: don't send any requests if the config is empty
	// fmt.Println("the config yaml", configYaml)

	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(configYaml)))
	if err != nil {
		mgr.Log.Error(err, "msg", "Failed to send request", "pod", pod.Name)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mgr.Log.Error(err, "msg", "Failed to read response", "pod", pod.Name)
		resp.Body.Close()
		return
	}

	mgr.Log.Info("Injecting response from pod", "pod", pod.Name, "response", string(body))
	resp.Body.Close()
}

func PodIsReady(pod *corev1.Pod) bool {
	// check if the pod is running and has an ip address and is not going to be deleted!
	if pod.Status.Phase == corev1.PodRunning &&
		pod.Status.PodIP != "" &&
		pod.DeletionTimestamp == nil {
		// check if the pod is ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady &&
				condition.Status != corev1.ConditionTrue {
				return false
			}
		}
		return true
	}
	return false
}

func init() {
	globalAgentManager = &AgentManager{
		Port:          8080,
		UpdatePath:    "update",
		AppLabelKey:   "app",
		AppLabelValue: "ipruler",
		Namespace:     "kube-system",
		Log:           ctrl.Log.WithName("AgentManager"),
	}
	sharedFullConfig = &SharedFullConfig{}
}
