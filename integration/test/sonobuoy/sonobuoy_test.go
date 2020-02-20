// +build k8srequired

package sonobuoy

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Test_Sonobuoy(t *testing.T) {
	err := sonobuoy.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	Logger    micrologger.Logger
	K8sClient kubernetes.Interface
	Provider  *Provider
}

type Sonobuoy struct {
	logger    micrologger.Logger
	k8sClient kubernetes.Interface
	provider  *Provider
}

func New(config Config) (*Sonobuoy, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	s := &Sonobuoy{
		logger:    config.Logger,
		k8sClient: config.K8sClient,
		provider:  config.Provider,
	}

	return s, nil
}

func (s *Sonobuoy) Test(ctx context.Context) error {
	secret, err := s.k8sClient.CoreV1().Secrets("default").Get(fmt.Sprintf("%s-api", s.provider.clusterID), v1.GetOptions{})
	if err != nil {
		log.Fatalf("can't fetch secret: %s", err)
	}
	kubeconfigTemplate := `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{.CA}}
    server: https://api.{{.ClusterID}}.k8s.godsmack.westeurope.azure.gigantic.io
  name: tenant-cluster
contexts:
- context:
    cluster: tenant-cluster
    user: ci-user
  name: tenant-cluster-context
current-context: tenant-cluster-context
kind: Config
preferences: {}
users:
- name: ci-user
  user:
    client-certificate-data: {{.Certificate}}
    client-key-data: {{.Key}}
`
	kubeconfigFilePath := "/tmp/kubeconfig"
	// Create the file
	kubeconfigFile, err := os.Create(kubeconfigFilePath)
	if err != nil {
		log.Fatalf("could not create kubeconfig file %s\n", err)
	}
	defer kubeconfigFile.Close()
	values := map[string]interface{}{
		"CA":          base64.StdEncoding.EncodeToString(secret.Data["ca"]),
		"Certificate": base64.StdEncoding.EncodeToString(secret.Data["crt"]),
		"ClusterID":   s.provider.clusterID,
		"Key":         base64.StdEncoding.EncodeToString(secret.Data["key"]),
	}
	t := template.Must(template.New("kubeconfig").Parse(kubeconfigTemplate))
	err = t.Execute(kubeconfigFile, values)
	if err != nil {
		log.Fatalf("templating kubeconfig file %s\n", err)
	}

	{
		cmd := exec.Command("go", "go", "get", "-u", "github.com/heptio/sonobuoy")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return microerror.Maskf(executionFailedError, "Error installing sonobuoy %v", err, "sonobuoyOutput", out)
		}
		s.logger.LogCtx(ctx, "output", out)
	}

	{
		cmd := exec.Command("sonobuoy", "run", "--kubeconfig", kubeconfigFilePath, "--wait", "--mode=quick")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return microerror.Maskf(executionFailedError, "Error running sonobuoy %v", err, "sonobuoyOutput", out)
		}
	}

	var resultsPath string
	{
		retrieve := exec.Command("sonobuoy", "retrieve", "--kubeconfig", kubeconfigFilePath)
		out, err := retrieve.CombinedOutput()
		if err != nil {
			return microerror.Maskf(executionFailedError, "Sonobuoy could not retrieve tests results %v", err, "sonobuoyOutput", out)
		}
		resultsPath = strings.TrimSuffix(string(out), "\n")
	}

	var results string
	{
		resultsCmd := exec.Command("sonobuoy", "results", resultsPath)
		out, err := resultsCmd.CombinedOutput()
		if err != nil {
			return microerror.Maskf(executionFailedError, "Sonobuoy could not open the results file on %s", resultsPath, "sonobuoyOutput", out)
		}
		results = string(out)
	}

	if !strings.Contains(results, "Failed: 0") {
		return microerror.Maskf(executionFailedError, "Sonobuoy tests contain failures")
	}

	return nil
}
