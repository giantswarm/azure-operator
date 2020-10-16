package employees

import (
	"strings"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
)

type SSHUserList map[string][]string

func FromDraughtsmanString(userList string) (SSHUserList, error) {
	sshUsers := SSHUserList{}

	for _, user := range strings.Split(userList, ",") {
		if user == "" {
			continue
		}

		trimmed := strings.TrimSpace(user)
		split := strings.Split(trimmed, ":")

		if len(split) != 2 {
			return nil, microerror.Maskf(parsingFailedError, "SSH user format must be <name>:<public key>")
		}

		if split[0] == "" || len(split[1]) == 0 {
			continue
		}

		sshUsers[split[0]] = append(sshUsers[split[0]], split[1])
	}

	return sshUsers, nil
}

func (s *SSHUserList) ToClusterKubernetesSSHUser() []v1alpha1.ClusterKubernetesSSHUser {
	var ret []v1alpha1.ClusterKubernetesSSHUser

	for name, keys := range *s {
		ret = append(ret, v1alpha1.ClusterKubernetesSSHUser{
			Name: name,
			// v1alpha1 type currently supports only one ssh key per user.
			PublicKey: keys[0],
		})
	}

	return ret
}
