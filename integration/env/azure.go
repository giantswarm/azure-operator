package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/giantswarm/azure-operator/integration/network"
)

const (
	EnvVarAzureAZs              = "AZURE_AZS"
	EnvVarAzureCIDR             = "AZURE_CIDR"
	EnvVarAzureCalicoSubnetCIDR = "AZURE_CALICO_SUBNET_CIDR"
	EnvVarAzureMasterSubnetCIDR = "AZURE_MASTER_SUBNET_CIDR"
	EnvVarAzureVPNSubnetCIDR    = "AZURE_VPN_SUBNET_CIDR"
	EnvVarAzureWorkerSubnetCIDR = "AZURE_WORKER_SUBNET_CIDR"

	EnvVarAzureClientID       = "AZURE_CLIENTID"
	EnvVarAzureClientSecret   = "AZURE_CLIENTSECRET"
	EnvVarAzureLocation       = "AZURE_LOCATION"
	EnvVarAzureSubscriptionID = "AZURE_SUBSCRIPTIONID"
	EnvVarAzureTenantID       = "AZURE_TENANTID"

	EnvVarCommonDomainResourceGroup = "COMMON_DOMAIN_RESOURCE_GROUP"
	EnvVarCircleBuildNumber         = "CIRCLE_BUILD_NUM"
)

var (
	azureClientID       string
	azureClientSecret   string
	azureLocation       string
	azureSubscriptionID string
	azureTenantID       string

	azureCIDR             string
	azureCalicoSubnetCIDR string
	azureMasterSubnetCIDR string
	azureVPNSubnetCIDR    string
	azureWorkerSubnetCIDR string

	commonDomainResourceGroup string
	sshPublicKey              string
)

func init() {
	azureClientID = os.Getenv(EnvVarAzureClientID)
	if azureClientID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureClientID))
	}

	azureClientSecret = os.Getenv(EnvVarAzureClientSecret)
	if azureClientSecret == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureClientSecret))
	}

	azureLocation = os.Getenv(EnvVarAzureLocation)
	if azureLocation == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureLocation))
	}

	azureSubscriptionID = os.Getenv(EnvVarAzureSubscriptionID)
	if azureSubscriptionID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureSubscriptionID))
	}

	azureTenantID = os.Getenv(EnvVarAzureTenantID)
	if azureTenantID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureTenantID))
	}

	commonDomainResourceGroup = os.Getenv(EnvVarCommonDomainResourceGroup)
	if commonDomainResourceGroup == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCommonDomainResourceGroup))
	}

	sshPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDBSSJCLkZWhOvs6blotU+fWbrTmC7fOwOm0+w01Ww/YN3j3j1vCrvji1A4Yonr89ePQEQKfZsYcYFodQI/D3Uzu9rOFy0dCMQfvL/J6N8LkNtmooh3J2p061829MurAdD+TVsNGrD2FZGm5Ab4NiyDXIGAYCaHL6BHP16ipBglYjLQt6jVyzdTbYspkRi1QrsNFN3gIv9V47qQSvoNEsC97gvumKzCSQ/EwJzFoIlqVkZZHZTXvGwnZrAVXB69t9Y8OJ5zA6cYFAKR0O7lEiMpebdLNGkZgMA6t2PADxfT78PHkYXLR/4tchVuOSopssJqgSs7JgIktEE14xKyNyoLKIyBBo3xwywnDySsL8R2zG4Ytw1luo79pnSpIzTvfwrNhd7Cg//OYzyDCty+XUEUQx2JfOBx5Qb1OFw71WA+zYqjbworOsy2ZZ9UAy8ryjiaeT8L2ZRGuhdicD6kkL3Lxg5UeNIxS2FLNwgepZ4D8Vo6Yxe+VOZl524ffoOJSHQ0Gz8uE76hXMNEcn4t8HVkbR4sCMgLn2YbwJ2dJcROj4w80O4qgtN1vsL16r4gt9o6euml8LbmnJz6MtGdMczSO7kHRxirtEHMTtYbT1wNgUAzimbScRggBpUz5gbz+NRE1Xgnf4A5yNMRy+JOWtLVUozJlcGSiQkVcexzdb27yQ==
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC7tf8jCm827wKhbBUh0xT/2D954cO54sOJ5/vn5sZSDIkxErMUCKH5WZSEjh3iAaKeq8wAn6XpXYvCwRu62csO1vu5l3Wh/kLnYo+1ALLoL8jM4VdKUiv4jOaM2ZL/UR5j1rt5L0kK3//kjtCXMlwyjpBxH9crJPA1lnmUdADDN+XBZ1x4EmpWwR8eV2CiYLU7sylF9V0R1bObUptpvOeYb/B3T1H9GSFgpVSQzvtI/OEZmoSzBz7VdJiIfGTwUKEcEr+9WBpVD5quLmG0LdwQ68dBeTjIaj4A5PYfu9iiNTKNiqDEIWtIkoVLo7PxZJblrYPQPYFycnUJeLHngZYmX12TBPcl3xQPdxyPeTGz4KBa0jfeWdHi7JkaOHtrmQvF0wcj3REEZYMJKz/8tMA4tqP5AnvTudZgNGHXtO9kiGhG5rn3dWTr6R+crRuWszQVVasx4IEKMOwdxc8sgmx1W0mPetKDUh6siFF3TRu0KcJ9BDrHGciWMkfXQgP4txIRgvPHGJmoywRQ3zoN0hWzjI6bEaUvRVEyk0u0dreTmTiG6JFcSaSMJWZvuhvKCKTbp1ysITzH7EIJwQ2nfSz88j4tVRfXA/BSxOc4aR6l3j1zApSfV7mVag9TSPfMVdXWEoOlpdiQH/V0Mm5ummkQ1JloDGBRKR0AuKUrGtkKfw==
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCu2V1p0cZ2N4ucug9LQMB+YMg9AQ5+aZQTdDTZ7oBuEcBuGtdnSSbcxj1lHoMYvhz6ugFVolkusRnZSakZY/XPVlwIHC56TWWrJ0hJ4sQEzCqVSHx0ZBHaMZepxCz7KSh/4KjtZFyaBC9SFwUo7kGgBYoFdClhxZsmfMsk0RneY8FjWme/cwXSaEGdaaTyOA52UOCg6Ax3nnE/gAJBsL8HgI17bFjj8og6TdPoP+33wujGHFORy8HF/m6p1I2Nm9Mp+gkG6PzdkWbF7UFci5uYHXy5IEu6uGzEPQiB5BjgfVIvZyH3VfKxmG1T2yyp4/qDQOmkjlIahpPyI00Y3SWAab7MdQXJ2hTgWFo/NP+AEdd45+PrSvTMy2k5bVl9GMntP+z+9oAhwH8OStSCJ0GBGlVG89fd0vFV1XVmLPwS8XhuhAoU1KRt6/Hc8cs7uSUiKOTY8Xn6VNUozxK137QpHBb81jU7OCcmopF9dlqoV6m18iZK1NjP4+FFxUyi5O4HI6aFrZXf7Cw5G9C8EXML3qLIMxd2pIJsu8QTw/5kC7sBtmFY/5RqW0TZ5hWuyGSuFcRan5E08Qct5rGAQ6QjJ9rZqQUPeJFcN6gEvGUam0XdeziZD6lPFUDkte9y653lIrPqBoSbsJuk/FJU/+RTSYEl+VCmaac3ru6jYV6M8w== tuomas@giantswarm.io
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCqr+y4/jMUAPvecuTUQbHLjIxFqex+boGkuPFd2H6fshCEQSNaU6Ou3ROmjaU1kfIJM/zTNI/YsDjdp7o9+w5YzeJr9MxSSPhSZ+ynFyh6SqVd4qNlnkSz7rhHIkN8cfIf28R8vAaEw5Qxn+xYBhd0m+KXOMsHx7rqtleEKrlZUnxaFLKM/3+d71EuDLdYIL011ZhzTl5iH66WIwgyNiLlKrrdvstkiTWf6fnhXpmG/SSPQtLeZfhRnJ26sKfRmnxlNYGAk4k9nUZLr2ee4o6+fteknFsJ2VJj9wfs0PjBeS+iASyKc0Vusesv3daROFTM6D2o7Artv3RrdCeMowtv tobiasz@giantswarm.io
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC4eK4FveEEsyhDZQty3zRtN6QOw7zRi3+ns5Fm+nZR6dpQZ1Q4A8XF0YDgvxrVlffE5Diwn7quSPInEUu0i+BKR1D+uXLyej+PWWd4kiSw7v4bl0jZ8FMEjoei2fUFZIFV4Uslug5ifW1yhzbLQZ2nkjY0mFlKHzcUSX6jJM/YauasV3gjrZ/YtUqm7keue5jXJZZernIFj7A8BuIu6K/KrjJXhKOphQO4oAm8nXaqWCpRUx9p8yWwZ2a8NxRQz7+MRWMCpdABCaPXpB9f8QWoB5TbBs3xKowQbV6rGiPEbk0Fv8tnY0ERGx8iFMVVGZnhApz79LUzby+cLoZvP5obPs/H9sG92Bpz6bPFkoIls9Ra2R7/udhDcQS0q/OAy/klPmbnABXPoi0cVx+iMUoI426yV8aYH3gO4IdT2s1an2klTlD+VQ/pWnWX172dZiAarebtRdCsvdeFKQkHRDH/v89NQvNFcLQDSOxb7RIPd8JQilrIt9jDv9N837RQ+MNDwVHFa2sNqV7lFu4dKxPgfTxJg/NZA2XHa2fN7YItUqwwxtpqQuVLPt2p08n8WSTYkzZJnRnwZdw3RzflQyR/G3BNdaQidFMPfBjtibyUulH4LsuCNSQ8DV2lgIrXiCcdh4uCPAObmGbAz8xPfPmEY/NskD1n+JeXzSE8OVnLPw== nikola@giantswarm.io`

	// azureCDIR must be provided along with other CIDRs,
	// otherwise we compute CIDRs base on EnvVarCircleBuildNumber value.
	azureCDIR := os.Getenv(EnvVarAzureCIDR)
	if azureCDIR == "" {
		buildNumber, err := strconv.ParseUint(os.Getenv(EnvVarCircleBuildNumber), 10, 32)
		if err != nil {
			panic(err)
		}

		subnets, err := network.ComputeSubnets(uint(buildNumber))
		if err != nil {
			panic(err)
		}

		azureCIDR = subnets.Parent.String()
		azureCalicoSubnetCIDR = subnets.Calico.String()
		azureMasterSubnetCIDR = subnets.Master.String()
		azureVPNSubnetCIDR = subnets.VPN.String()
		azureWorkerSubnetCIDR = subnets.Worker.String()
	}
}

func AzureAvailabilityZones() []int {
	azureAvailabilityZones := os.Getenv(EnvVarAzureAZs)
	if azureAvailabilityZones == "" {
		return []int{}
	}

	azs := strings.Split(strings.TrimSpace(azureAvailabilityZones), " ")
	zones := make([]int, len(azs))

	for i, s := range azs {
		zone, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("AvailabilityZones valid numbers are 1, "+
				"2, 3. Your '%s' env var contains %s",
				EnvVarAzureAZs, azureAvailabilityZones))
		}
		if zone < 1 || zone > 3 {
			panic(fmt.Sprintf("AvailabilityZones valid numbers are 1, "+
				"2, 3. Your '%s' env var contains %s",
				EnvVarAzureAZs, azureAvailabilityZones))
		}
		zones[i] = zone
	}
	return zones
}

func AzureCalicoSubnetCIDR() string {
	return azureCalicoSubnetCIDR
}

func AzureClientID() string {
	return azureClientID
}

func AzureClientSecret() string {
	return azureClientSecret
}

func AzureCIDR() string {
	return azureCIDR
}

func AzureLocation() string {
	return azureLocation
}

func AzureMasterSubnetCIDR() string {
	return azureMasterSubnetCIDR
}

func AzureSubscriptionID() string {
	return azureSubscriptionID
}

func AzureTenantID() string {
	return azureTenantID
}

func AzureVPNSubnetCIDR() string {
	return azureVPNSubnetCIDR
}

func AzureWorkerSubnetCIDR() string {
	return azureWorkerSubnetCIDR
}

func CommonDomainResourceGroup() string {
	return commonDomainResourceGroup
}

func SSHPublicKey() string {
	return sshPublicKey
}
