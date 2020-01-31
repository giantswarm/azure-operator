package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"

	"github.com/giantswarm/e2e-harness/pkg/harness"
	"github.com/giantswarm/e2e-harness/pkg/runner"
	"github.com/giantswarm/e2e-harness/pkg/wait"
)

const (
	DefaultDirectory = "integration"
)

type E2e struct {
	Version string `yaml:"version"`
	Test    Test   `yaml:"test"`
}

type Step struct {
	Run     string   `yaml:"run"`
	WaitFor WaitStep `yaml:"waitFor"`
}

type WaitStep struct {
	Run     string        `yaml:"run"`
	Match   string        `yaml:"match"`
	Timeout time.Duration `yaml:"timeout"`
	Step    time.Duration `yaml:"step"`
}

type Test struct {
	Env []string `yaml:"env"`
}

type Config struct {
	Dir        string
	K8sContext string
	Name       string
	Tag        string
}

type Dependencies struct {
	Logger micrologger.Logger
	Runner runner.Runner
	Wait   *wait.Wait
	Fs     afero.Fs
}

type Project struct {
	logger micrologger.Logger
	runner runner.Runner
	wait   *wait.Wait
	fs     afero.Fs
	cfg    *Config
}

func New(deps *Dependencies, cfg *Config) *Project {
	p := &Project{
		logger: deps.Logger,
		runner: deps.Runner,
		wait:   deps.Wait,
		fs:     deps.Fs,

		cfg: cfg,
	}

	return p
}

func (p *Project) CommonSetupSteps(ctx context.Context) error {
	p.logger.Log("level", "info", "message", "executing common setup steps")

	steps := []Step{
		{
			Run: "kubectl config use-context " + p.cfg.K8sContext,
		},
		// Fix kube-dns RBAC issues.
		// Allow kube-dns and other kube-system services full access to the API.
		// See:
		// * https://github.com/kubernetes/minikube/issues/1734
		// * https://github.com/kubernetes/minikube/issues/1722
		{
			Run: "kubectl create clusterrolebinding cluster-admin:kube-system --clusterrole=cluster-admin --serviceaccount=kube-system:default",
		},
		{
			Run: "kubectl -n kube-system create sa tiller",
		},
		{
			Run: "kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller",
		},
		{
			Run: "helm init --service-account tiller",
			WaitFor: WaitStep{
				Run:   "kubectl get pod -n kube-system",
				Match: `tiller-deploy.*1/1\s*Running`,
			},
		}}

	for _, s := range steps {
		o := func() error {
			err := p.RunStep(s)
			if err != nil {
				return microerror.Mask(err)
			}
			return nil
		}
		b := backoff.NewExponential(backoff.MediumMaxWait, backoff.LongMaxInterval)
		n := backoff.NewNotifier(p.logger, context.Background())
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	p.logger.Log("level", "info", "message", "finished common setup steps")

	return nil
}

func (p *Project) CommonTearDownSteps(ctx context.Context) error {
	p.logger.Log("level", "info", "message", "starting common teardown steps")
	steps := []Step{
		{
			Run: "helm reset --force",
		},
		{
			Run: "kubectl -n kube-system delete sa tiller",
		},
		{
			Run: "kubectl delete clusterrolebinding tiller",
		}}

	for _, s := range steps {
		if err := p.RunStep(s); err != nil {
			return microerror.Mask(err)
		}
	}
	p.logger.Log("level", "info", "message", "finished common teardown steps")
	return nil
}

func (p *Project) Test(ctx context.Context) error {
	p.logger.Log("level", "info", "message", "started tests")

	// --test-dir is mounted in /e2e in the test container, and the binary with
	// the e2e tests is named <project_name>-e2e so the final location in the
	// test container is /e2e/<project_name>-e2e
	name := harness.GetProjectName()
	binaryPath := fmt.Sprintf("/e2e/%s-e2e", name)

	e2e, err := p.readProjectFile()
	if err != nil {
		return microerror.Mask(err)
	}

	if err := p.runner.RunPortForward(os.Stdout, binaryPath+" -test.v", e2e.Test.Env...); err != nil {
		return microerror.Mask(err)
	}

	p.logger.Log("level", "info", "message", "finished tests")

	return nil
}

func (p *Project) RunStep(step Step) error {
	// expand env vars
	sEnv := os.ExpandEnv(step.Run)

	//if err := p.runner.RunPortForward(ioutil.Discard, sEnv); err != nil {
	if err := p.runner.RunPortForward(os.Stdout, sEnv); err != nil {
		return microerror.Mask(err)
	}

	if step.WaitFor.Run != "" {
		md := &wait.MatchDef{
			Run:      step.WaitFor.Run,
			Match:    step.WaitFor.Match,
			Deadline: step.WaitFor.Timeout,
		}
		if err := p.wait.For(md); err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}

func (p *Project) readProjectFile() (*E2e, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	projectFile := filepath.Join(dir, p.cfg.Dir, "project.yaml")
	if _, err := os.Stat(projectFile); os.IsNotExist(err) {
		msg := fmt.Sprintf("optional %s not found, continue build", projectFile)
		p.logger.Log("level", "info", "message", msg)
		return &E2e{}, nil
	}

	afs := &afero.Afero{Fs: p.fs}
	content, err := afs.ReadFile(projectFile)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	e2e := &E2e{}

	if err := yaml.Unmarshal(content, e2e); err != nil {
		return nil, microerror.Mask(err)
	}
	return e2e, nil
}
