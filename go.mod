module github.com/giantswarm/azure-operator/v5

go 1.18

require (
	github.com/Azure/azure-sdk-for-go v65.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.15.0
	github.com/Azure/go-autorest/autorest v0.11.28
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.11
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/coreos/go-semver v0.3.0
	github.com/giantswarm/apiextensions/v6 v6.0.0
	github.com/giantswarm/badnodedetector v1.0.1
	github.com/giantswarm/certs/v4 v4.0.0
	github.com/giantswarm/conditions v0.5.0
	github.com/giantswarm/conditions-handler v0.3.0
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/exporterkit v1.0.0
	github.com/giantswarm/ipam v0.3.0
	github.com/giantswarm/k8sclient/v7 v7.0.1
	github.com/giantswarm/k8scloudconfig/v14 v14.4.1-0.20220830100205-1b8db4744274
	github.com/giantswarm/k8smetadata v0.9.3
	github.com/giantswarm/kubelock/v2 v2.0.0
	github.com/giantswarm/microendpoint v1.0.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/microkit v1.0.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/giantswarm/operatorkit/v7 v7.0.1
	github.com/giantswarm/release-operator/v3 v3.2.0
	github.com/giantswarm/tenantcluster/v6 v6.0.0
	github.com/giantswarm/to v0.4.0
	github.com/giantswarm/versionbundle v1.0.0
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.8
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.13.0
	github.com/spf13/viper v1.12.0
	golang.org/x/sync v0.0.0-20220819030929-7fc1605a5dde
	k8s.io/api v0.22.5
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	sigs.k8s.io/cluster-api v1.0.5
	sigs.k8s.io/cluster-api-provider-azure v1.0.2
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.18 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.5 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/giantswarm/backoff v1.0.0 // indirect
	github.com/giantswarm/microstorage v0.2.0 // indirect
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gobuffalo/flect v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onsi/gomega v1.17.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	go.opentelemetry.io/otel v1.3.0 // indirect
	go.opentelemetry.io/otel/trace v1.3.0 // indirect
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa // indirect
	golang.org/x/net v0.0.0-20220520000938-2e3eb7b945c2 // indirect
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/resty.v1 v1.12.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.22.5 // indirect
	k8s.io/component-base v0.22.5 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
)

replace (
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.24
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.8
	github.com/containerd/imgcrypt => github.com/containerd/imgcrypt v1.1.6
	github.com/coredns/coredns => github.com/coredns/coredns v1.9.3
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.27+incompatible
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/docker/distribution => github.com/docker/distribution v2.8.1+incompatible
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.8.1
	github.com/go-ldap/ldap/v3 => github.com/go-ldap/ldap/v3 v3.4.4
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/go-logr/stdr => github.com/go-logr/stdr v0.4.0
	github.com/gofiber/fiber/v2 => github.com/gofiber/fiber/v2 v2.36.0
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	github.com/microcosm-cc/bluemonday => github.com/microcosm-cc/bluemonday v1.0.19
	github.com/nats-io/jwt => github.com/nats-io/jwt/v2 v2.3.0
	github.com/nats-io/nats-server/v2 => github.com/nats-io/nats-server/v2 v2.8.4
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.3
	github.com/pkg/sftp => github.com/pkg/sftp v1.13.5
	github.com/valyala/fasthttp => github.com/valyala/fasthttp v1.39.0
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.10.1
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.9.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211110013926-83f114cd0513
	sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.5
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.10.3
)
