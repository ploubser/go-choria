package choria

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/provtarget"
	"github.com/choria-io/go-choria/submission"
	"github.com/fatih/color"
	"github.com/golang-jwt/jwt"
	"github.com/nats-io/jsm.go/kv"
	"golang.org/x/term"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	certmanagersec "github.com/choria-io/go-choria/providers/security/certmanager"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/security"
	"github.com/choria-io/go-choria/providers/security/filesec"
	"github.com/choria-io/go-choria/providers/security/puppetsec"
	"github.com/choria-io/go-choria/puppet"
	"github.com/choria-io/go-choria/srvcache"
	log "github.com/sirupsen/logrus"
)

// Framework is a utility encompassing choria config and various utilities
type Framework struct {
	Config *config.Config

	security security.Provider
	log      *log.Logger

	bi       *build.Info
	srvcache *srvcache.Cache
	puppet   *puppet.Wrapper
	mu       *sync.Mutex
}

// New sets up a Choria with all its config loaded and so forth
func New(path string) (*Framework, error) {
	conf, err := config.NewConfig(path)
	if err != nil {
		return nil, err
	}

	conf.ApplyBuildSettings(BuildInfo())

	return NewWithConfig(conf)
}

// NewWithConfig creates a new instance of the framework with the supplied config instance
func NewWithConfig(cfg *config.Config) (*Framework, error) {
	c := Framework{
		Config: cfg,
		mu:     &sync.Mutex{},
		bi:     BuildInfo(),
	}

	rand.Seed(time.Now().UnixNano())

	err := c.SetupLogging(false)
	if err != nil {
		return &c, fmt.Errorf("could not set up logging: %s", err)
	}

	config.Mutate(cfg, c.Logger("config"))

	c.srvcache = srvcache.New(cfg.Identity, 5*time.Second, net.LookupSRV, c.Logger("srvcache"))
	c.puppet = puppet.New()

	err = c.setupSecurity()
	if err != nil {
		return &c, fmt.Errorf("could not set up security framework: %s", err)
	}

	return &c, nil
}

// BuildInfo retrieves build information
func (fw *Framework) BuildInfo() *build.Info {
	return BuildInfo()
}

func (fw *Framework) setupSecurity() error {
	var err error

	switch fw.Config.Choria.SecurityProvider {
	case "puppet":
		fw.security, err = puppetsec.New(
			puppetsec.WithResolver(fw),
			puppetsec.WithChoriaConfig(fw.BuildInfo(), fw.Config),
			puppetsec.WithLog(fw.Logger("security")))

	case "file":
		fw.security, err = filesec.New(
			filesec.WithChoriaConfig(fw.BuildInfo(), fw.Config),
			filesec.WithLog(fw.Logger("security")))

	case "pkcs11":
		err = fw.setupPKCS11()

	case "certmanager":
		fw.security, err = certmanagersec.New(
			certmanagersec.WithChoriaConfig(fw.Config),
			certmanagersec.WithLog(fw.Logger("security")),
			certmanagersec.WithContext(context.Background()))

	default:
		err = fmt.Errorf("unknown security provider %s", fw.Config.Choria.SecurityProvider)
	}

	if err != nil {
		return err
	}

	if !(fw.Config.DisableSecurityProviderVerify || fw.Config.DisableTLS) && protocol.IsSecure() {
		errors, ok := fw.security.Validate()
		if !ok {
			return fmt.Errorf("security setup is not valid, %d errors encountered: %s", len(errors), strings.Join(errors, ", "))
		}
	}

	return nil
}

// ProvisionMode determines if this instance is in provisioning mode
// if the setting `plugin.choria.server.provision` is set at all then
// the value of that is returned, else it the build time property
// ProvisionDefault is consulted
func (fw *Framework) ProvisionMode() bool {
	if !fw.Config.InitiatedByServer || (fw.bi.ProvisionBrokerURLs() == "" && fw.bi.ProvisionJWTFile() == "" && fw.bi.ProvisionToken() == "") {
		return false
	}

	if fw.Config.HasOption("plugin.choria.server.provision") {
		return fw.Config.Choria.Provision
	}

	return fw.bi.ProvisionDefault()
}

// PrometheusTextFileDir is the configured directory where to write prometheus text file stats
func (fw *Framework) PrometheusTextFileDir() string {
	return fw.Config.Choria.PrometheusTextFileDir
}

// SupportsProvisioning determines if a node can auto provision
func (fw *Framework) SupportsProvisioning() bool {
	if fw.ProvisionMode() {
		return true
	}

	return fw.bi.SupportsProvisioning()
}

// ConfigureProvisioning adjusts the active configuration to match the
// provisioning profile
func (fw *Framework) ConfigureProvisioning() {
	provtarget.Configure(fw.Config, fw.Logger("provtarget"))

	if !fw.ProvisionMode() {
		return
	}

	fw.Config.RPCAuthorization = false
	fw.Config.Choria.FederationCollectives = []string{}
	fw.Config.Collectives = []string{"provisioning"}
	fw.Config.MainCollective = "provisioning"
	fw.Config.Registration = []string{}
	fw.Config.FactSourceFile = fw.bi.ProvisionFacts()
	fw.Config.Choria.NatsUser = fw.bi.ProvisioningBrokerUsername()
	fw.Config.Choria.NatsPass = fw.bi.ProvisioningBrokerPassword()
	fw.Config.Choria.SecurityAlwaysOverwriteCache = true
	fw.Config.Choria.SSLDir = filepath.Join(filepath.Dir(fw.Config.ConfigFile), "ssl")
	fw.Config.Choria.SecurityProvider = "file"

	if fw.bi.ProvisionStatusFile() != "" {
		fw.Config.Choria.StatusFilePath = fw.bi.ProvisionStatusFile()
		fw.Config.Choria.StatusUpdateSeconds = 10
	}

	if fw.bi.ProvisionRegistrationData() != "" {
		fw.Config.RegistrationCollective = "provisioning"
		fw.Config.Registration = []string{"file_content"}
		fw.Config.RegisterInterval = 120
		fw.Config.RegistrationSplay = false
		fw.Config.Choria.FileContentRegistrationTarget = "provisioning.registration.data"
		fw.Config.Choria.FileContentRegistrationData = fw.bi.ProvisionRegistrationData()
	}

	if !fw.bi.ProvisionSecurity() {
		protocol.Secure = "false"
	}

	if fw.bi.ProvisionBrokerSRVDomain() != "" {
		fw.Config.Choria.UseSRVRecords = true
		fw.Config.Choria.SRVDomain = fw.bi.ProvisionBrokerSRVDomain()
	}
}

// IsFederated determines if the configuration is setting up any Federation collectives
func (fw *Framework) IsFederated() (result bool) {
	return len(fw.FederationCollectives()) != 0
}

// Logger creates a new logrus entry
func (fw *Framework) Logger(component string) *log.Entry {
	return fw.log.WithFields(log.Fields{"component": component})
}

// FederationCollectives determines the known Federation Member
// Collectives based on the CHORIA_FED_COLLECTIVE environment
// variable or the choria.federation.collectives config item
func (fw *Framework) FederationCollectives() (collectives []string) {
	var found []string

	env := os.Getenv("CHORIA_FED_COLLECTIVE")

	if env != "" {
		found = strings.Split(env, ",")
	}

	if len(found) == 0 {
		found = fw.Config.Choria.FederationCollectives
	}

	for _, collective := range found {
		collectives = append(collectives, strings.TrimSpace(collective))
	}

	return
}

// FederationMiddlewareServers determines the correct Federation Middleware Servers
//
// It does this by:
//
//    * looking for choria.federation_middleware_hosts configuration
//	  * Doing SRV lookups of  _mcollective-federation_server._tcp and _x-puppet-mcollective_federation._tcp
func (fw *Framework) FederationMiddlewareServers() (servers srvcache.Servers, err error) {
	configured := fw.Config.Choria.FederationMiddlewareHosts
	servers = srvcache.NewServers()

	if len(configured) > 0 {
		servers, err = srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured Federation Middleware: %s", err)
		}
	}

	if servers.Count() == 0 {
		servers, err = fw.QuerySrvRecords([]string{"_mcollective-federation_server._tcp", "_x-puppet-mcollective_federation._tcp"})
		if err != nil {
			return servers, fmt.Errorf("could not resolve Federation Middleware Server SRV records: %s", err)
		}
	}

	servers.Each(func(s srvcache.Server) {
		if s.Scheme() == "" {
			s.SetScheme("nats")
		}
	})

	return servers, err
}

// PuppetDBServers resolves the PuppetDB server based on configuration of _x-puppet-db._tcp
func (fw *Framework) PuppetDBServers() (servers srvcache.Servers, err error) {
	if fw.Config.Choria.PuppetDBHost != "" {
		configured := fmt.Sprintf("%s:%d", fw.Config.Choria.PuppetDBHost, fw.Config.Choria.PuppetDBPort)

		servers, err = srvcache.StringHostsToServers([]string{configured}, "https")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured PuppetDB host: %s", err)
		}

		return servers, nil
	}

	if fw.Config.Choria.UseSRVRecords {
		servers, err = fw.QuerySrvRecords([]string{"_x-puppet-db._tcp"})
		if err != nil {
			return servers, fmt.Errorf("could not resolve PuppetDB Server SRV records: %s", err)
		}

		if servers.Count() == 0 {
			servers, err = fw.QuerySrvRecords([]string{"_x-puppet._tcp"})
			if err != nil {
				return servers, fmt.Errorf("could not resolve Puppet Server SRV records: %s", err)
			}

			// In the case where we take _x-puppet._tcp SRV records we unfortunately have
			// to force the port else it uses the one from Puppet which will 404
			servers.Each(func(s srvcache.Server) {
				s.SetPort(fw.Config.Choria.PuppetDBPort)
			})
		}

		servers.Each(func(s srvcache.Server) {
			if s.Scheme() == "" {
				s.SetScheme("https")
			}
		})
	}

	if servers == nil || servers.Count() == 0 {
		configured := fmt.Sprintf("%s:%d", "puppet", fw.Config.Choria.PuppetDBPort)

		servers, err = srvcache.StringHostsToServers([]string{configured}, "https")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured PuppetDB host: %s", err)
		}
	}

	return servers, nil
}

// ProvisioningServers determines the build time provisioning servers
// when it's unset or results in an empty server list this will return
// an error
func (fw *Framework) ProvisioningServers(ctx context.Context) (srvcache.Servers, error) {
	return provtarget.Targets(ctx, fw.Logger("provtarget"))
}

// ShouldUseNGS determined is we are configured to use NGS
func (fw *Framework) ShouldUseNGS() bool {
	return fw.Config.Choria.NatsNGS && fw.Config.Choria.NatsCredentials != ""
}

// MiddlewareServers determines the correct Middleware Servers
//
// It does this by:
//
//    * if ngs is configured and credentials are set and middleware_hosts are empty, use ngs
//    * looking for choria.federation_middleware_hosts configuration
//	  * Doing SRV lookups of _mcollective-server._tcp and __x-puppet-mcollective._tcp
//    * Defaulting to puppet:4222
func (fw *Framework) MiddlewareServers() (servers srvcache.Servers, err error) {
	configured := fw.Config.Choria.MiddlewareHosts

	if fw.ShouldUseNGS() && len(configured) == 0 {
		return srvcache.NewServers(srvcache.NewServer("connect.ngs.global", 4222, "nats")), nil
	}

	if fw.IsFederated() {
		return fw.FederationMiddlewareServers()
	}

	servers = srvcache.NewServers()

	if len(configured) > 0 {
		servers, err = srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured Middleware: %s", err)
		}
	}

	if servers.Count() == 0 && fw.Config.Choria.UseSRVRecords {
		servers, err = fw.QuerySrvRecords([]string{"_mcollective-server._tcp", "_x-puppet-mcollective._tcp"})
		if err != nil {
			log.Warnf("Could not resolve Middleware Server SRV records: %s", err)
		}
	}

	if servers.Count() == 0 {
		servers = srvcache.NewServers(srvcache.NewServer("puppet", 4222, "nats"))
	}

	servers.Each(func(s srvcache.Server) {
		if s.Scheme() == "" {
			s.SetScheme("nats")
		}
	})

	return servers, nil
}

func (fw *Framework) SetLogWriter(out io.Writer) {
	if fw.log != nil {
		fw.log.SetOutput(out)
	}
}

func (fw *Framework) commonLogOpener() error {
	switch {
	case fw.Config.LogFile == "discard":
		fw.log.SetOutput(io.Discard)

	case fw.Config.LogFile != "":
		fw.log.Formatter = &log.JSONFormatter{}

		file, err := os.OpenFile(fw.Config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("could not set up logging: %s", err)
		}

		fw.log.SetOutput(file)
	}

	return nil
}

// SetLogger sets the logger to use
func (fw *Framework) SetLogger(logger *log.Logger) {
	fw.log = logger
}

// SetupLogging configures logging based on choria config directives
// currently only file and console behaviors are supported
func (fw *Framework) SetupLogging(debug bool) (err error) {
	fw.log = log.New()
	fw.log.SetOutput(os.Stdout)

	err = fw.openLogfile()
	if err != nil {
		return err
	}

	switch fw.Config.LogLevel {
	case "debug":
		fw.log.SetLevel(log.DebugLevel)
	case "info":
		fw.log.SetLevel(log.InfoLevel)
	case "warn":
		fw.log.SetLevel(log.WarnLevel)
	case "error":
		fw.log.SetLevel(log.ErrorLevel)
	case "fatal":
		fw.log.SetLevel(log.FatalLevel)
	default:
		fw.log.SetLevel(log.WarnLevel)
	}

	if debug {
		fw.log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(fw.log.Formatter)
	log.SetLevel(fw.log.Level)
	log.SetOutput(fw.log.Out)

	return
}

// TrySrvLookup will attempt to lookup a series of names returning the first found
// if SRV lookups are disabled or nothing is found the default will be returned
func (fw *Framework) TrySrvLookup(names []string, defaultSrv srvcache.Server) (srvcache.Server, error) {
	if !fw.Config.Choria.UseSRVRecords {
		return defaultSrv, nil
	}

	for _, q := range names {
		a, err := fw.QuerySrvRecords([]string{q})
		if err == nil && a.Count() > 0 {
			found := a.Servers()[0]
			log.Infof("Found %s:%d from %s SRV lookups", found.Host(), found.Port(), strings.Join(names, ", "))

			return found, nil
		}
	}

	log.Debugf("Could not find SRV records for %s, returning defaults %s:%d", strings.Join(names, ", "), defaultSrv.Host(), defaultSrv.Port())

	return defaultSrv, nil
}

// QuerySrvRecords looks for SRV records within the right domain either
// thanks to facter domain or the configured domain.
//
// If the config disables SRV then a error is returned.
func (fw *Framework) QuerySrvRecords(records []string) (srvcache.Servers, error) {
	servers := srvcache.NewServers()

	if !fw.Config.Choria.UseSRVRecords {
		return servers, errors.New("SRV lookups are disabled in the configuration file")
	}

	domain := fw.Config.Choria.SRVDomain
	var err error

	if fw.Config.Choria.SRVDomain == "" {
		domain, err = fw.FacterDomain()
		if err != nil {
			return servers, err
		}

		// cache the result to speed things up
		fw.Config.Choria.SRVDomain = domain
	}

	for _, q := range records {
		record := q + "." + domain
		log.Debugf("Attempting SRV lookup for %s", record)

		servers, err = fw.srvcache.LookupSrvServers("", "", record, "")
		if err != nil {
			log.Debugf("Failed to resolve %s: %s", record, err)
			continue
		}

		log.Debugf("Found %d SRV records for %s", servers.Count(), record)
		break
	}

	return servers, nil
}

// NetworkBrokerPeers are peers in the broker cluster resolved from
// _mcollective-broker._tcp or from the plugin config
func (fw *Framework) NetworkBrokerPeers() (servers srvcache.Servers, err error) {
	servers, err = fw.QuerySrvRecords([]string{"_mcollective-broker._tcp"})
	if err != nil {
		log.Errorf("SRV lookup for _mcollective-broker._tcp failed: %s", err)
		err = nil
	}

	if servers.Count() == 0 {
		servers, err = srvcache.StringHostsToServers(fw.Config.Choria.NetworkPeers, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse network peers: %s", err)
		}
	}

	servers.Each(func(f srvcache.Server) {
		if f.Scheme() == "" {
			f.SetScheme("nats")
		}
	})

	return servers, nil
}

// DiscoveryServer is the server configured as a discovery proxy
func (fw *Framework) DiscoveryServer() (srvcache.Server, error) {
	dflt := srvcache.NewServer(fw.Config.Choria.DiscoveryHost, fw.Config.Choria.DiscoveryPort, "")

	if !fw.ProxiedDiscovery() {
		return dflt, errors.New("proxy discovery is not enabled")
	}

	return fw.TrySrvLookup([]string{"_mcollective-discovery._tcp"}, dflt)
}

// ProxiedDiscovery determines if a client is configured for proxied discover
func (fw *Framework) ProxiedDiscovery() bool {
	if fw.Config.HasOption("plugin.choria.discovery_host") || fw.Config.HasOption("plugin.choria.discovery_port") {
		return true
	}

	return fw.Config.Choria.DiscoveryProxy
}

// Getuid returns the numeric user id of the caller
func (fw *Framework) Getuid() int {
	return os.Getuid()
}

// PuppetSetting retrieves a config setting by shelling out to puppet apply --configprint
func (fw *Framework) PuppetSetting(setting string) (string, error) {
	return fw.puppet.Setting(setting)
}

// FacterStringFact looks up a facter fact, returns "" when unknown
func (fw *Framework) FacterStringFact(fact string) (string, error) {
	return fw.puppet.FacterStringFact(fact)
}

// FacterFQDN determines the machines fqdn by querying facter.  Returns "" when unknown
func (fw *Framework) FacterFQDN() (string, error) {
	return fw.puppet.FacterStringFact("networking.fqdn")
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func (fw *Framework) FacterDomain() (string, error) {
	return fw.puppet.FacterStringFact("networking.domain")
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
func (fw *Framework) FacterCmd() string {
	return fw.puppet.AIOCmd("facter", "")
}

// PuppetAIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
func (fw *Framework) PuppetAIOCmd(command string, def string) string {
	return fw.puppet.AIOCmd(command, def)
}

// NewRequestID Creates a new RequestID
func (fw *Framework) NewRequestID() (string, error) {
	return NewRequestID()
}

// UniqueID creates a new unique ID, usually a v4 uuid, if that fails a random string based ID is made
func (fw *Framework) UniqueID() string {
	return util.UniqueID()
}

// CallerID determines the cert based callerid
func (fw *Framework) CallerID() string {
	caller, _, _, err := fw.UniqueIDFromUnverifiedToken()
	if err == nil {
		return caller
	}

	return fmt.Sprintf("choria=%s", fw.Certname())
}

// HasCollective determines if a collective is known in the configuration
func (fw *Framework) HasCollective(collective string) bool {
	for _, c := range fw.Config.Collectives {
		if c == collective {
			return true
		}
	}

	return false
}

// OverrideCertname indicates if the user wish to force a specific certname, empty when not
func (fw *Framework) OverrideCertname() string {
	return fw.Config.OverrideCertname
}

// DisableTLSVerify indicates if the user whish to disable TLS verification
func (fw *Framework) DisableTLSVerify() bool {
	return fw.Config.DisableTLSVerify
}

// Configuration returns the active configuration
func (fw *Framework) Configuration() *config.Config {
	return fw.Config
}

func (fw *Framework) UniqueIDFromUnverifiedToken() (caller string, id string, token string, err error) {
	t, claims, err := fw.ParseSignerTokenUnverified()
	if err != nil {
		return "", "", "", err
	}

	caller, ok := claims["callerid"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("invalid callerid in token claims")
	}

	return caller, fmt.Sprintf("%x", md5.Sum([]byte(caller))), t.Raw, nil
}

func (fw *Framework) ParseSignerTokenUnverified() (token *jwt.Token, claims jwt.MapClaims, err error) {
	ts, err := fw.SignerToken()
	if err != nil {
		return nil, nil, err
	}

	claims = jwt.MapClaims{}
	token, _, err = new(jwt.Parser).ParseUnverified(ts, &claims)
	if err != nil {
		return nil, nil, err
	}

	err = claims.Valid()
	if err != nil {
		return nil, nil, err
	}

	return token, claims, nil
}

// Retrieves the AAA token used for signing requests
func (fw *Framework) SignerToken() (token string, err error) {
	if fw.Config.Choria.RemoteSignerTokenFile == "" && fw.Config.Choria.RemoteSignerTokenEnvironment == "" {
		return "", fmt.Errorf("no token file or environment variable is defined")
	}

	if fw.Config.Choria.RemoteSignerTokenFile != "" {
		tb, err := os.ReadFile(fw.Config.Choria.RemoteSignerTokenFile)
		if err != nil {
			return "", fmt.Errorf("could not read token file: %v", err)
		}

		return strings.TrimSpace(string(tb)), err
	}

	token = os.Getenv(fw.Config.Choria.RemoteSignerTokenEnvironment)
	if token == "" {
		return "", fmt.Errorf("did not find a token in environment variable %s", fw.Config.Choria.RemoteSignerTokenEnvironment)
	}

	return strings.TrimSpace(token), nil
}

// HTTPClient creates a *http.Client prepared by the security provider with certificates and more set
func (fw *Framework) HTTPClient(secure bool) (*http.Client, error) {
	return fw.security.HTTPClient(secure)
}

func (fw *Framework) PQLQuery(query string) ([]byte, error) {
	q := url.Values{}
	q.Set("query", query)
	path := fmt.Sprintf("/pdb/query/v4?%s", q.Encode())

	pdb, err := fw.PuppetDBServers()
	if err != nil {
		return nil, err
	}
	pdbhost := pdb.Strings()[0]

	fw.log.Debugf("Performing PQL query against %s: %s", pdbhost, query)

	client, err := fw.HTTPClient(true)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", fmt.Sprintf("%s%s", pdbhost, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid PuppetDB response: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (fw *Framework) PQLQueryCertNames(query string) ([]string, error) {
	body, err := fw.PQLQuery(query)
	if err != nil {
		return nil, err
	}

	var res []struct {
		Certname    string `json:"certname"`
		Deactivated bool   `json:"deactivated"`
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, r := range res {
		if !r.Deactivated {
			nodes = append(nodes, r.Certname)
		}
	}

	return nodes, nil
}

// Colorize returns a string of either 'red', 'green' or 'yellow'. If the 'color' configuration
// is set to false then the string will have no color hints
func (fw *Framework) Colorize(c string, format string, a ...interface{}) string {
	if !fw.Config.Color {
		return fmt.Sprintf(format, a...)
	}

	switch c {
	case "red":
		return color.RedString(fmt.Sprintf(format, a...))
	case "green":
		return color.GreenString(fmt.Sprintf(format, a...))
	case "yellow":
		return color.YellowString(fmt.Sprintf(format, a...))
	default:
		return fmt.Sprintf(format, a...)
	}
}

// ProgressWidth determines the width of the progress bar, when -1 there is not enough space for a progress bar
func (fw *Framework) ProgressWidth() int {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}

	if width < 35 {
		return -1
	}

	width -= 30
	if width > 80 {
		width = 80
	}

	return width
}

// GovernorSubject the subject to use for choria managed Governors within a collective
func GovernorSubject(name string, collective string) string {
	return fmt.Sprintf("%s.governor.%s", collective, name)
}

// GovernorSubject the subject to use for choria managed Governors
func (fw *Framework) GovernorSubject(name string) string {
	return GovernorSubject(name, fw.Config.MainCollective)
}

func (fw *Framework) KV(ctx context.Context, conn inter.Connector, bucket string, create bool, opts ...kv.Option) (kv.KV, error) {
	kv, _, err := fw.KVWithConn(ctx, conn, bucket, create, opts...)
	return kv, err
}

// KVWithConn creates a connection to a key-value store and gives access to the connector
func (fw *Framework) KVWithConn(ctx context.Context, conn inter.Connector, bucket string, create bool, opts ...kv.Option) (kv.KV, inter.Connector, error) {
	logger := fw.Logger("kv")

	var err error
	if conn == nil {
		conn, err = fw.NewConnector(ctx, fw.MiddlewareServers, fmt.Sprintf("kv %s", fw.CallerID()), logger)
		if err != nil {
			return nil, nil, err
		}
	}

	opts = append(opts, kv.WithLogger(logger))

	var store kv.KV

	if create {
		store, err = kv.NewBucket(conn.Nats(), bucket, opts...)
	} else {
		store, err = kv.NewClient(conn.Nats(), bucket, opts...)
	}
	if err != nil {
		return nil, conn, err
	}

	return store, conn, nil
}

// DirectorySubmitter gives access to submit data into the Choria Submission system if enabled
func (fw *Framework) DirectorySubmitter() (submission.Submitter, error) {
	if fw.Config.Choria.SubmissionSpool == "" {
		return nil, fmt.Errorf("submission not enabled")
	}

	return submission.NewFromChoria(fw, submission.Directory)
}
