module github.com/keel-hq/keel

go 1.22.4

toolchain go1.22.5

replace (
	k8s.io/api => k8s.io/api v0.26.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.26.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.26.3
	k8s.io/apiserver => k8s.io/apiserver v0.26.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.26.3
	k8s.io/client-go => k8s.io/client-go v0.26.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.26.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.26.3
	k8s.io/code-generator => k8s.io/code-generator v0.26.3
	k8s.io/component-base => k8s.io/component-base v0.26.3
	k8s.io/cri-api => k8s.io/cri-api v0.26.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.26.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.26.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.26.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.26.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.26.3
	k8s.io/kubectl => k8s.io/kubectl v0.26.3
	k8s.io/kubelet => k8s.io/kubelet v0.26.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.26.3
	k8s.io/metrics => k8s.io/metrics v0.26.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.26.3
)

replace helm.sh/helm/v3 => helm.sh/helm/v3 v3.11.2

replace k8s.io/kubernetes => k8s.io/kubernetes v1.26.3

replace github.com/docker/distribution => github.com/docker/distribution v2.8.1+incompatible

require (
	cloud.google.com/go/pubsub v1.37.0
	github.com/Masterminds/semver v1.5.0
	github.com/aws/aws-sdk-go v1.44.240
	github.com/daneharrigan/hipchat v0.0.0-20170512185232-835dc879394a
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.8.3+incompatible
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jinzhu/gorm v1.9.16
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0
	github.com/prometheus/client_golang v1.17.0
	github.com/rusenask/cron v1.1.0
	github.com/rusenask/docker-registry-client v0.0.0-20200210164146-049272422097
	github.com/ryanuber/go-glob v1.0.0
	github.com/sirupsen/logrus v1.9.3
	github.com/slack-go/slack v0.12.2
	github.com/stretchr/testify v1.9.0
	github.com/tbruyelle/hipchat-go v0.0.0-20170717082847-35aebc99209a
	github.com/urfave/negroni v1.0.0
	golang.org/x/net v0.26.0
	google.golang.org/api v0.170.0
	google.golang.org/grpc v1.62.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	helm.sh/helm/v3 v3.11.2
	k8s.io/api v0.26.3
	k8s.io/apimachinery v0.26.3
	k8s.io/cli-runtime v0.26.3
	k8s.io/client-go v0.26.3
	sigs.k8s.io/yaml v1.3.0
)

require (
	cloud.google.com/go/storage v1.40.0
	golang.org/x/oauth2 v0.18.0
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
)

require (
	cloud.google.com/go v0.112.1 // indirect
	cloud.google.com/go/compute v1.24.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.7 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/Masterminds/squirrel v1.5.3 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/containerd/containerd v1.7.20 // indirect
	github.com/containerd/errdefs v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v27.1.0+incompatible // indirect
	github.com/docker/docker v27.1.2+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/emicklei/go-restful/v3 v3.10.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-gorp/gorp/v3 v3.0.5 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.3 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.10.7 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.15 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rubenv/sql-migrate v1.3.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240314234333-6e1732d8331c // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240311132316-a219d84964c2 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.26.0 // indirect
	k8s.io/apiserver v0.26.3 // indirect
	k8s.io/component-base v0.26.3 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/kube-openapi v0.0.0-20221012153701-172d655c2280 // indirect
	k8s.io/kubectl v0.26.0 // indirect
	k8s.io/utils v0.0.0-20230220204549-a5ecb0141aa5 // indirect
	oras.land/oras-go v1.2.6 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/kustomize/api v0.12.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
