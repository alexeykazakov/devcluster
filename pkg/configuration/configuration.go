// Package configuration is in charge of the validation and extraction of all
// the configuration details from environment variables.
package configuration

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	// Commit current build commit set by build script.
	Commit = "0"
	// BuildTime set by build script in ISO 8601 (UTC) format:
	// YYYY-MM-DDThh:mm:ssTZD (see https://www.w3.org/TR/NOTE-datetime for
	// details).
	BuildTime = "0"
	// StartTime in ISO 8601 (UTC) format.
	StartTime = time.Now().UTC().Format("2006-01-02T15:04:05Z")
)

const (
	// EnvPrefix will be used for environment variable name prefixing.
	EnvPrefix = "DEVCLUSTER"

	// Constants for viper variable names. Will be used to set
	// default values as well as to get each value.
	varHTTPAddress = "http.address"
	// DefaultHTTPAddress is the address and port string that your service will
	// be exported to by default.
	DefaultHTTPAddress = "0.0.0.0:8080"

	varHTTPIdleTimeout = "http.idle_timeout"
	// DefaultHTTPIdleTimeout specifies the default timeout for HTTP idling.
	DefaultHTTPIdleTimeout = time.Second * 15

	varHTTPCompressResponses = "http.compress"
	// DefaultHTTPCompressResponses compresses HTTP responses for clients that
	// support it via the 'Accept-Encoding' header.
	DefaultHTTPCompressResponses = true

	// varEnvironment specifies service environment such as prod, stage, unit-tests, e2e-tests, dev, etc
	varEnvironment = "environment"
	// DefaultEnvironment is the default environment
	DefaultEnvironment          = "prod"
	UnitTestsEnvironment        = "unit-tests"
	IntegrationTestsEnvironment = "integration-tests"

	varLogLevel = "log.level"
	// DefaultLogLevel is the default log level used in your service.
	DefaultLogLevel = "info"

	varLogJSON = "log.json"
	// DefaultLogJSON is a switch to toggle on and off JSON log output.
	DefaultLogJSON = false

	varGracefulTimeout = "graceful_timeout"
	// DefaultGracefulTimeout is the duration for which the server gracefully
	// wait for existing connections to finish - e.g. 15s or 1m.
	DefaultGracefulTimeout = time.Second * 15

	varHTTPWriteTimeout = "http.write_timeout"
	// DefaultHTTPWriteTimeout specifies the default timeout for HTTP writes.
	DefaultHTTPWriteTimeout = time.Second * 15

	varHTTPReadTimeout = "http.read_timeout"
	// DefaultHTTPReadTimeout specifies the default timeout for HTTP reads.
	DefaultHTTPReadTimeout = time.Second * 15

	varAuthClientLibraryURL = "auth_client.library_url"
	// DefaultAuthClientLibraryURL is the default auth library location.
	DefaultAuthClientLibraryURL = "https://sso.prod-preview.openshift.io/auth/js/keycloak.js"

	varAuthClientConfigRaw = "auth_client.config.raw"
	// DefaultAuthClientConfigRaw specifies the auth client config.
	DefaultAuthClientConfigRaw = `{
  "realm": "devcluster-public",
  "auth-server-url": "https://sso.prod-preview.openshift.io/auth",
  "ssl-required": "none",
  "resource": "devcluster-public",
  "clientId": "devcluster-public",
  "public-client": true
}`

	varAuthClientConfigContentType = "auth_client.config.content_type"
	// DefaultAuthClientConfigContentType specifies the auth client config content type.
	DefaultAuthClientConfigContentType = "application/json; charset=utf-8"

	varAuthClientPublicKeysURL = "auth_client.public_keys_url"
	// DefaultAuthClientPublicKeysURL is the default log level used in your service.
	DefaultAuthClientPublicKeysURL = "https://sso.prod-preview.openshift.io/auth/realms/devcluster-public/protocol/openid-connect/certs"

	varNamespace = "namespace"
	// DefaultNamespace is the default k8s namespace to use.
	DefaultNamespace = "devcluster"

	// General IBM Cloud configuration
	varIBMCloudAPIKey             = "ibmcloud.apikey"
	varIBMCloudApiCallRetrySec    = "ibmcloud.api_call_retry_sec"
	DefaultBMCloudApiCallRetrySec = 30

	// Tenant cluster authN
	varIBMCloudAccountID = "ibmcloud.account_id"
	varIBMCloudTenantID  = "ibmcloud.tenant_id"

	varMongodbConnectionString = "mongodb.connection_string"
	varMongodbDatabase         = "mongodb.database"
	DefaultMongodbDatabase     = "devcluster-dummy"
)

// Config encapsulates the Viper configuration which stores the
// configuration data in-memory.
type Config struct {
	v *viper.Viper
}

// New creates a new configuration
func New() *Config {
	c := &Config{
		v: viper.New(),
	}
	c.v.SetEnvPrefix(EnvPrefix)
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()
	return c
}

// GetViperInstance returns the underlying Viper instance.
func (c *Config) GetViperInstance() *viper.Viper {
	return c.v
}

func (c *Config) setConfigDefaults() {
	c.v.SetTypeByDefaultValue(true)

	c.v.SetDefault(varHTTPAddress, DefaultHTTPAddress)
	c.v.SetDefault(varHTTPCompressResponses, DefaultHTTPCompressResponses)
	c.v.SetDefault(varHTTPWriteTimeout, DefaultHTTPWriteTimeout)
	c.v.SetDefault(varHTTPReadTimeout, DefaultHTTPReadTimeout)
	c.v.SetDefault(varHTTPIdleTimeout, DefaultHTTPIdleTimeout)
	c.v.SetDefault(varEnvironment, DefaultEnvironment)
	c.v.SetDefault(varLogLevel, DefaultLogLevel)
	c.v.SetDefault(varLogJSON, DefaultLogJSON)
	c.v.SetDefault(varGracefulTimeout, DefaultGracefulTimeout)
	c.v.SetDefault(varAuthClientLibraryURL, DefaultAuthClientLibraryURL)
	c.v.SetDefault(varAuthClientConfigRaw, DefaultAuthClientConfigRaw)
	c.v.SetDefault(varAuthClientConfigContentType, DefaultAuthClientConfigContentType)
	c.v.SetDefault(varAuthClientPublicKeysURL, DefaultAuthClientPublicKeysURL)
	c.v.SetDefault(varNamespace, DefaultNamespace)
	c.v.SetDefault(varMongodbDatabase, DefaultMongodbDatabase)
	c.v.SetDefault(varIBMCloudApiCallRetrySec, DefaultBMCloudApiCallRetrySec)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or
// environment variable) that the app-server binds to (e.g. "0.0.0.0:8080").
func (c *Config) GetHTTPAddress() string {
	return c.v.GetString(varHTTPAddress)
}

// GetHTTPCompressResponses returns true if HTTP responses should be compressed
// for clients that support it via the 'Accept-Encoding' header.
func (c *Config) GetHTTPCompressResponses() bool {
	return c.v.GetBool(varHTTPCompressResponses)
}

// GetHTTPWriteTimeout returns the duration for the write timeout.
func (c *Config) GetHTTPWriteTimeout() time.Duration {
	return c.v.GetDuration(varHTTPWriteTimeout)
}

// GetHTTPReadTimeout returns the duration for the read timeout.
func (c *Config) GetHTTPReadTimeout() time.Duration {
	return c.v.GetDuration(varHTTPReadTimeout)
}

// GetHTTPIdleTimeout returns the duration for the idle timeout.
func (c *Config) GetHTTPIdleTimeout() time.Duration {
	return c.v.GetDuration(varHTTPIdleTimeout)
}

// GetEnvironment returns the environment such as prod, stage, unit-tests, e2e-tests, dev, etc
func (c *Config) GetEnvironment() string {
	return c.v.GetString(varEnvironment)
}

// GetLogLevel returns the logging level (as set via config file or environment
// variable).
func (c *Config) GetLogLevel() string {
	return c.v.GetString(varLogLevel)
}

// IsLogJSON returns if we should log json format (as set via config file or
// environment variable).
func (c *Config) IsLogJSON() bool {
	return c.v.GetBool(varLogJSON)
}

// GetGracefulTimeout returns the duration for which the server gracefully wait
// for existing connections to finish - e.g. 15s or 1m.
func (c *Config) GetGracefulTimeout() time.Duration {
	return c.v.GetDuration(varGracefulTimeout)
}

// IsTestingMode returns if the service runs in unit-tests environment
func (c *Config) IsTestingMode() bool {
	return c.GetEnvironment() == UnitTestsEnvironment
}

// GetAuthClientLibraryURL returns the auth library location (as set via
// config file or environment variable).
func (c *Config) GetAuthClientLibraryURL() string {
	return c.v.GetString(varAuthClientLibraryURL)
}

// GetAuthClientConfigAuthContentType returns the auth config config content type (as
// set via config file or environment variable).
func (c *Config) GetAuthClientConfigAuthContentType() string {
	return c.v.GetString(varAuthClientConfigContentType)
}

// GetAuthClientConfigAuthRaw returns the auth config config (as
// set via config file or environment variable).
func (c *Config) GetAuthClientConfigAuthRaw() string {
	return c.v.GetString(varAuthClientConfigRaw)
}

// GetAuthClientPublicKeysURL returns the public keys URL (as set via config file
// or environment variable).
func (c *Config) GetAuthClientPublicKeysURL() string {
	return c.v.GetString(varAuthClientPublicKeysURL)
}

// GetIBMCloudApiCallRetrySec returns the number of seconds to wait between retrying calling IBM API
func (c *Config) GetIBMCloudApiCallRetrySec() int {
	return c.v.GetInt(varIBMCloudApiCallRetrySec)
}

// GetNamespace returns the namespace in which the devcluster service and host operator is running
func (c *Config) GetNamespace() string {
	return c.v.GetString(varNamespace)
}

// GetIBMCloudAPIKey returns the IBM Cloud API Key
func (c *Config) GetIBMCloudAPIKey() string {
	return c.v.GetString(varIBMCloudAPIKey)
}

// GetIBMCloudAccountID returns the main/parent IBM Cloud Account ID
func (c *Config) GetIBMCloudAccountID() string {
	return c.v.GetString(varIBMCloudAccountID)
}

// GetIBMCloudTenantID returns the Cloud Directory ID
func (c *Config) GetIBMCloudTenantID() string {
	return c.v.GetString(varIBMCloudTenantID)
}

func (c *Config) GetMongodbConnectionString() string {
	return c.v.GetString(varMongodbConnectionString)
}

// GetMongodbDatabase returns the mongo database name
func (c *Config) GetMongodbDatabase() string {
	return c.v.GetString(varMongodbDatabase)
}
