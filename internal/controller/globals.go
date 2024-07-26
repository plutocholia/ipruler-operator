package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/plutocholia/ipruler-controller/internal/models"
	corev1 "k8s.io/api/core/v1"
)

// type AgentPodStatus int

// const (
// 	Sync AgentPodStatus = iota + 1
// 	OutOfSync
// )

// type AgentPod struct {
// 	Pod    *corev1.Pod
// 	Status AgentPodStatus
// }

// func NewAgentPod(pod *corev1.Pod) *AgentPod {
// 	return &AgentPod{
// 		Pod:    pod,
// 		Status: OutOfSync,
// 	}
// }

type AgentManager struct {
	Config        *models.ConfigModel
	Port          int
	UpdatePath    string
	Mutex         sync.Mutex
	Namespace     string
	AppLabelKey   string
	AppLabelValue string
}

var (
	globalAgentManager *AgentManager
)

func (mgr *AgentManager) InjectConfig(pod *corev1.Pod) {

	setupLog.Info("Injecting config file to", "pod", pod.Name)
	url := fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, mgr.Port, mgr.UpdatePath)

	configYaml, _ := ConvertToYAML(mgr.Config)

	// TODO: don't send any requests if the config is empty
	// fmt.Println("the config yaml", configYaml)

	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(configYaml)))
	if err != nil {
		setupLog.Error(err, "msg", "Failed to send request", "pod", pod.Name)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		setupLog.Error(err, "msg", "Failed to read response", "pod", pod.Name)
		resp.Body.Close()
		return
	}

	setupLog.Info("Injecting response from pod", "pod", pod.Name, "response", string(body))
	resp.Body.Close()
}

func PodIsReadyForConfigInjection(pod *corev1.Pod) bool {
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
	globalAgentManager = &AgentManager{}
	// TODO: change below hard-codeds to be filled by env vars
	globalAgentManager.Port = 8080
	globalAgentManager.UpdatePath = "update"
	globalAgentManager.AppLabelKey = "app"
	globalAgentManager.AppLabelValue = "ipruler-api"
	globalAgentManager.Namespace = "kube-system"
}
