package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/go-logr/logr"
	iprulerv1 "github.com/plutocholia/ipruler-controller/api/v1"
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
	NodeConfigs   map[string]*models.ConfigModel
	ClusterConfig *models.ConfigModel
	Port          int
	UpdatePath    string
	Mutex         sync.Mutex
	Namespace     string
	AppLabelKey   string
	AppLabelValue string
	Log           logr.Logger
}

var (
	globalAgentManager *AgentManager
	sharedFullConfig   *SharedFullConfig
)

// adds NodeConfig the the NodeConfigs map of AgentManager based on the NodeConfig ID
func (mgr *AgentManager) AddNodeConfig(nodeConfig *iprulerv1.NodeConfig) {
	mgr.Mutex.Lock()
	defer mgr.Mutex.Unlock()
	mgr.NodeConfigs[GetNodeConfigID(nodeConfig)] = &nodeConfig.Spec.Config
}

func (mgr *AgentManager) DeleteNodeConfig(nodeConfig *iprulerv1.NodeConfig) {
	mgr.Mutex.Lock()
	defer mgr.Mutex.Unlock()
	delete(mgr.NodeConfigs, GetNodeConfigID(nodeConfig))
}

// returns merged node selector of the node config as the ID of that config
func GetNodeConfigID(nodeConfig *iprulerv1.NodeConfig) string {
	nodeSelectorMergeLabels := ""
	for key, value := range nodeConfig.Spec.NodeSelector {
		nodeSelectorMergeLabels += fmt.Sprintf("%s:%s,", key, value)
	}
	return nodeSelectorMergeLabels
}

func (mgr *AgentManager) AddClusterConfig(clusterConfig *models.ConfigModel) {
	mgr.Mutex.Lock()
	defer mgr.Mutex.Unlock()
	mgr.ClusterConfig = clusterConfig
}

func (mgr *AgentManager) GetMergedConfigByPod(pod *corev1.Pod) *models.ConfigModel {
	mgr.Mutex.Lock()
	defer mgr.Mutex.Unlock()
	return nil
	// TODO: find the right NodeConfig ID based on the labels of the nodes which the pod is located!

}

func (mgr *AgentManager) FindNodeConfigByLabelList(nodeLabels map[string]string) *models.ConfigModel {
	mgr.Mutex.Lock()
	defer mgr.Mutex.Unlock()
	for key, value := range nodeLabels {
		if nodeConfig, exists := mgr.NodeConfigs[fmt.Sprintf("%s:%s,", key, value)]; exists {
			return nodeConfig
		}
	}
	return nil
}

func (mgr *AgentManager) InjectConfig(pod *corev1.Pod, config *models.ConfigModel) {
	mgr.Mutex.Lock()
	defer mgr.Mutex.Unlock()

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

// init for globals
func init() {

	globalAgentManager = &AgentManager{}
	// TODO: change below hard-codeds to be filled by env vars
	globalAgentManager.Port = 8080
	globalAgentManager.UpdatePath = "update"
	globalAgentManager.AppLabelKey = "app"
	globalAgentManager.AppLabelValue = "ipruler-api"
	globalAgentManager.Namespace = "kube-system"
	globalAgentManager.NodeConfigs = make(map[string]*models.ConfigModel)
	globalAgentManager.Log = ctrl.Log.WithName("AgentManager")

	sharedFullConfig = &SharedFullConfig{}
}
