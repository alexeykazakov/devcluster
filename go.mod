module github.com/codeready-toolchain/devcluster

go 1.14

require (
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/codeready-toolchain/toolchain-common v0.0.0-20200827095351-b2cb9e622a9a
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/gzip v0.0.1
	github.com/gin-gonic/gin v1.5.0
	github.com/go-logr/logr v0.1.0
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/matryer/resync v0.0.0-20161211202428-d39c09a11215
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	go.mongodb.org/mongo-driver v1.1.2
	gopkg.in/h2non/gock.v1 v1.0.14
	gopkg.in/square/go-jose.v2 v2.3.1
	k8s.io/apimachinery v0.18.3
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.3 // Required by prometheus-operator
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6 // avoids case-insensitive import collision: "github.com/googleapis/gnostic/openapiv2" and "github.com/googleapis/gnostic/OpenAPIv2"
)
