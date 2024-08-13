package controller

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	env "github.com/Netflix/go-env"
	"github.com/go-logr/logr"
	"github.com/plutocholia/ipruler-operator/internal/models"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Environment struct {
	IPRulerAgentPort        int    `env:"IPRULER_AGENT_API_PORT,default=9301"`
	IPRulerAgentNamespace   string `env:"IPRULER_AGENT_NAMESPACE,default=kube-system"`
	IPRulerAgentLabelKey    string `env:"IPRULER_AGENT_LABEL_KEY,default=app"`
	IPRulerAgentLabelValue  string `env:"IPRULER_AGENT_LABEL_VALUE,default=ipruler-agent"`
	IPRulerAgentUpdatePath  string `env:"IPRULER_AGENT_UPDATE_PATH,default=update"`
	IPRulerAgentCleanupPath string `env:"IPRULER_AGENT_CLEANUP_PATH,default=cleanup"`
}

func (e *Environment) String() string {
	return fmt.Sprintf(`
Environments:
	IPRulerAgentPort: %d
	IPRulerAgentNamespace: %s
	IPRulerAgentLabelKey: %s
	IPRulerAgentLabelValue: %s
	IPRulerAgentUpdatePath: %s
	IPRulerAgentCleanupPath: %s
`, e.IPRulerAgentPort, e.IPRulerAgentNamespace, e.IPRulerAgentLabelKey, e.IPRulerAgentLabelValue, e.IPRulerAgentUpdatePath, e.IPRulerAgentCleanupPath)
}

type SharedFullConfig struct {
	Mutex                  sync.Mutex
	ClusterConfigName      string
	ClusterConfigNamespace string
}

type AgentManager struct {
	Port          int
	UpdatePath    string
	CleanupPath   string
	Namespace     string
	AppLabelKey   string
	AppLabelValue string
	Log           logr.Logger
}

var (
	envirnment         Environment
	globalAgentManager *AgentManager
	sharedFullConfig   *SharedFullConfig
)

func (mgr *AgentManager) InjectConfig(pod *corev1.Pod, config *models.ConfigModel) {
	mgr.Log.Info("Injecting config file to", "pod", pod.Name)
	url := fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, mgr.Port, mgr.UpdatePath)

	configYaml, _ := ConvertToYAML(config)

	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(configYaml)))
	if err != nil {
		mgr.Log.Error(err, "Failed to send request", "pod", pod.Name)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mgr.Log.Error(err, "Failed to read response", "pod", pod.Name)
		resp.Body.Close()
		return
	}

	mgr.Log.Info("Injecting response from pod", "pod", pod.Name, "response", string(body))
	resp.Body.Close()
}

func (mgr *AgentManager) Cleanup(pod *corev1.Pod) {
	mgr.Log.Info("Cleaup", "pod", pod.Name)
	url := fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, mgr.Port, mgr.CleanupPath)

	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		mgr.Log.Error(err, "Failed to send cleanup request", "pod", pod.Name)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mgr.Log.Error(err, "Failed to read cleanup response", "pod", pod.Name)
		resp.Body.Close()
		return
	}

	mgr.Log.Info("Cleanup response from pod", "pod", pod.Name, "response", string(body))
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

func ConvertToYAML(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func init() {
	if _, err := env.UnmarshalFromEnviron(&envirnment); err != nil {
		log.Fatal(err)
	}

	fmt.Println(envirnment.String())

	globalAgentManager = &AgentManager{
		Port:          envirnment.IPRulerAgentPort,
		UpdatePath:    envirnment.IPRulerAgentUpdatePath,
		CleanupPath:   envirnment.IPRulerAgentCleanupPath,
		AppLabelKey:   envirnment.IPRulerAgentLabelKey,
		AppLabelValue: envirnment.IPRulerAgentLabelValue,
		Namespace:     envirnment.IPRulerAgentNamespace,
		Log:           ctrl.Log.WithName("AgentManager"),
	}
	sharedFullConfig = &SharedFullConfig{}
}
