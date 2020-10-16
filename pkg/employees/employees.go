package employees

import (
	"strings"

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
