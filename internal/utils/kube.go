package utils

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/metadata"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var Clientset *kubernetes.Clientset
var KubeConfig *string

type DestroyItem struct {
	YamlContent string
	Namespace   string
	TaskType    string
	Force       bool
	EndTime     time.Time
	WaitTime    time.Duration
}

/*
 * 初始化Kubernetes客户端
 * @param configContent kubeconfig配置内容
 * @return *kubernetes.Clientset 客户端实例
 * @return error 错误对象
 */
func InitK8SClient(configContent string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if configContent == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			if home := homedir.HomeDir(); home != "" {
				KubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
			} else {
				KubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
			}
			flag.Parse()
			config, err = clientcmd.BuildConfigFromFlags("", *KubeConfig)
			if err != nil {
				return nil, err
			}
		}
	} else {
		config, err = clientcmd.RESTConfigFromKubeConfig([]byte(configContent))
		if err != nil {
			return nil, err
		}
	}
	var clientset *kubernetes.Clientset

	// Use protobuf to serialize client-server communication
	config = metadata.ConfigFor(config)
	if clientset, err = kubernetes.NewForConfig(config); err != nil {
		return nil, err
	}
	return clientset, nil
}

/*
 * 部署推理任务到Kubernetes集群
 * @param namespace 命名空间
 * @param taskType 任务类型
 * @param yamlContent K8S资源配置内容
 * @param kubeConfig kubeconfig配置
 * @return error 错误对象
 */
func ApplyInference(namespace string, taskType string, yamlContent string, kubeConfig string) error {
	// Create temp file to store kubeconfig string
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return fmt.Errorf("error creating temp kubeconfig file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Ensure temp file is deleted after use

	if _, err := tmpFile.Write([]byte(kubeConfig)); err != nil {
		return fmt.Errorf("error writing to temp kubeconfig file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp kubeconfig file: %v", err)
	}

	// Execute kubectl apply with specific kubeconfig file
	cmd := exec.Command("kubectl", "apply", "--kubeconfig", tmpFile.Name(), "-f", "-")
	cmd.Stdin = bytes.NewReader([]byte(yamlContent))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running kubectl apply --kubeconfig: %v", err)
	}

	return nil
}

func Apply(namespace string, taskType string, yamlContent string) error {
	// Execute kubectl apply
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = bytes.NewReader([]byte(yamlContent))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running kubectl apply:%v", err)
	}

	return nil
}

// Condition represents a condition in the status of a Kubernetes resource
type Condition struct {
	Type string `json:"type"`
}

// Status represents the status of a Kubernetes resource
type Status struct {
	Conditions []Condition `json:"conditions"`
}

func GetPytorchJobStatus(namespace, resourceType, name string) (string, error) {
	// Define kubectl command to execute
	return GetCRDJson(namespace, resourceType, name)
}

func GetCRDJson(namespace, resourceType, name string) (string, error) {
	// Define kubectl command to execute
	kubectlCmd := exec.Command("kubectl", "get",
		resourceType,
		name,
		"-n", namespace)

	// Get kubectl command output
	var kubectlOut bytes.Buffer
	kubectlCmd.Stdout = &kubectlOut
	cmdStr := fmt.Sprintf("kubectl get %s %s -n %s", resourceType, name, namespace)
	if err := kubectlCmd.Run(); err != nil {
		Errorf("Error executing kubectl command: %v, err: %v\n", cmdStr, err)
		return "", err
	}

	// Parse output
	output := kubectlOut.String()
	lines := strings.Split(output, "\n")

	// Ensure output has multiple lines, first line as header
	if len(lines) < 2 {
		return "", fmt.Errorf("no data found")
	}

	// Get second line
	secondLine := lines[1]
	if strings.TrimSpace(secondLine) == "" {
		return "", fmt.Errorf("second line is empty")
	}

	// Split second line content
	fields := strings.Fields(secondLine)
	if len(fields) < 2 {
		return "", fmt.Errorf("unexpected line format:%v", secondLine)
	}

	// Get JOB STATUS field
	return fields[1], nil

}

func K8sDelete(yamlContent string, force bool) error {
	var cmd *exec.Cmd
	if force {
		cmd = exec.Command("kubectl", "delete", "--force", "-f", "-")
	} else {
		cmd = exec.Command("kubectl", "delete", "-f", "-")
	}

	cmd.Stdin = bytes.NewReader([]byte(yamlContent))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		Errorf("error running kubectl delete: %v", err)
		return err
	}
	return nil
}

func DeleteSync(s *DestroyItem) error {
	// Execute kubectl apply
	var cmd *exec.Cmd
	if s.Force {
		cmd = exec.Command("kubectl", "delete", "--force", "-f", "-")
	} else {
		cmd = exec.Command("kubectl", "delete", "-f", "-")
	}

	cmd.Stdin = bytes.NewReader([]byte(s.YamlContent))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running kubectl delete:%v", err)
	}

	return nil
}

func DeleteSyncInference(namespace, taskType, yamlContent string, kubeConfig string) error {
	// Create temp file to store kubeconfig string
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return fmt.Errorf("error creating temp kubeconfig file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Ensure temp file is deleted after use

	if _, err := tmpFile.Write([]byte(kubeConfig)); err != nil {
		return fmt.Errorf("error writing to temp kubeconfig file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp kubeconfig file: %v", err)
	}

	// Execute kubectl apply
	cmd := exec.Command("kubectl", "--kubeconfig", tmpFile.Name(), "delete", "-f", "-")
	cmd.Stdin = bytes.NewReader([]byte(yamlContent))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running kubectl delete --kubeconfig:%v", err)
	}

	return nil
}

func GetTaskLabelSelector(name string) v1.ListOptions {
	return v1.ListOptions{LabelSelector: "task-id=" + name}
}
