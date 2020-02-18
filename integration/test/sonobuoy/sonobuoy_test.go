// +build k8srequired

package sonobuoy

import (
	"bytes"
	"context"
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
		logger:   config.Logger,
		provider: config.Provider,
	}

	return s, nil
}

func (s *Sonobuoy) Test(ctx context.Context) error {
	secret, err := s.k8sClient.CoreV1().Secrets("default").Get(fmt.Sprintf("%s-api", s.provider.clusterID), v1.GetOptions{})
	if err != nil {
		log.Fatalf("can't fetch secret: %s", err)
	}
	kubeconfig := `apiVersion: v1
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
		"CA":          secret.StringData["ca"],
		"Certificate": secret.StringData["crt"],
		"ClusterID":   s.provider.clusterID,
		"Key":         secret.StringData["key"],
	}
	t := template.Must(template.New("letter").Parse(kubeconfig))
	err = t.Execute(kubeconfigFile, values)
	if err != nil {
		log.Fatalf("templating kubeconfig file %s\n", err)
	}

	{
		cmd := exec.Command("sonobuoy", "run", "--kubeconfig", kubeconfigFilePath, "--wait", "--mode=certified-conformance")
		err := cmd.Run()
		if err != nil {
			log.Fatalf("sonobuoy failed with %s\n", err)
		}
	}

	var resultsPath string
	{
		retrieve := exec.Command("sonobuoy", "retrieve", "--kubeconfig", kubeconfigFilePath)
		var stdout bytes.Buffer
		retrieve.Stdout = &stdout
		err := retrieve.Run()
		if err != nil {
			log.Fatalf("sonobuoy failed with %s\n", err)
		}
		resultsPath = string(stdout.Bytes())
	}

	var results string
	{
		resultsCmd := exec.Command("sonobuoy", "results", resultsPath)
		var stdout bytes.Buffer
		resultsCmd.Stdout = &stdout
		err := resultsCmd.Run()
		if err != nil {
			log.Fatalf("sonobuoy failed with %s\n", err)
		}
		results = string(stdout.Bytes())
	}

	if !strings.Contains(results, "Failed: 0") {
		return microerror.Maskf(executionFailedError, "Sonobuoy tests have failed")
	}

	return nil
}
