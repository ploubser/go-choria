// generated code; DO NOT EDIT

package rpcutilclient

import (
	"fmt"
	"sync"
	"time"

	"context"

	coreclient "github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

// Stats are the statistics for a request
type Stats interface {
	Agent() string
	Action() string
	All() bool
	NoResponseFrom() []string
	UnexpectedResponseFrom() []string
	DiscoveredCount() int
	DiscoveredNodes() *[]string
	FailCount() int
	OKCount() int
	ResponsesCount() int
	PublishDuration() (time.Duration, error)
	RequestDuration() (time.Duration, error)
	DiscoveryDuration() (time.Duration, error)
	OverrideDiscoveryTime(start time.Time, end time.Time)
	UniqueRequestID() string
}

// NodeSource discovers nodes
type NodeSource interface {
	Reset()
	Discover(ctx context.Context, fw inter.Framework, filters []FilterFunc) ([]string, error)
}

// FilterFunc can generate a Choria filter
type FilterFunc func(f *protocol.Filter) error

// RenderFormat is the format used by the RenderResults helper
type RenderFormat int

const (
	// JSONFormat renders the results as a JSON document
	JSONFormat RenderFormat = iota

	// TextFormat renders the results as a Choria typical result set in line with choria req output
	TextFormat

	// TableFormat renders all successful responses in a table
	TableFormat

	// TXTFooter renders only the request summary statistics
	TXTFooter
)

// DisplayMode overrides the DDL display hints
type DisplayMode uint8

const (
	// DisplayDDL shows results based on the configuration in the DDL file
	DisplayDDL = DisplayMode(iota)
	// DisplayOK shows only passing results
	DisplayOK
	// DisplayFailed shows only failed results
	DisplayFailed
	// DisplayAll shows all results
	DisplayAll
	// DisplayNone shows no results
	DisplayNone
)

type Log interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
}

// RpcutilClient to the rpcutil agent
type RpcutilClient struct {
	fw            inter.Framework
	cfg           *config.Config
	ddl           *agent.DDL
	ns            NodeSource
	clientOpts    *initOptions
	clientRPCOpts []rpcclient.RequestOption
	filters       []FilterFunc
	targets       []string
	workers       int
	exprFilter    string
	noReplies     bool

	sync.Mutex
}

// Metadata is the agent metadata
type Metadata struct {
	License     string `json:"license"`
	Author      string `json:"author"`
	Timeout     int    `json:"timeout"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Must create a new client and panics on error
func Must(fw inter.Framework, opts ...InitializationOption) (client *RpcutilClient) {
	c, err := New(fw, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

// New creates a new client to the rpcutil agent
func New(fw inter.Framework, opts ...InitializationOption) (client *RpcutilClient, err error) {
	c := &RpcutilClient{
		fw:            fw,
		ddl:           &agent.DDL{},
		clientRPCOpts: []rpcclient.RequestOption{},
		filters: []FilterFunc{
			FilterFunc(coreclient.AgentFilter("rpcutil")),
		},
		clientOpts: &initOptions{
			cfgFile: coreclient.UserConfig(),
		},
		targets: []string{},
	}

	for _, opt := range opts {
		opt(c.clientOpts)
	}

	c.cfg = c.fw.Configuration()

	if c.clientOpts.dt > 0 {
		c.cfg.DiscoveryTimeout = int(c.clientOpts.dt.Seconds())
	}

	if c.clientOpts.ns == nil {
		switch c.cfg.DefaultDiscoveryMethod {
		case "choria":
			c.clientOpts.ns = &PuppetDBNS{}
		default:
			c.clientOpts.ns = &BroadcastNS{}
		}
	}
	c.ns = c.clientOpts.ns

	if c.clientOpts.logger == nil {
		c.clientOpts.logger = c.fw.Logger("rpcutil")
	} else {
		c.fw.SetLogger(c.clientOpts.logger.Logger)
	}

	c.ddl, err = DDL()
	if err != nil {
		return nil, fmt.Errorf("could not parse embedded DDL: %s", err)
	}

	return c, nil
}

// AgentMetadata is the agent metadata this client supports
func (p *RpcutilClient) AgentMetadata() *Metadata {
	return &Metadata{
		License:     p.ddl.Metadata.License,
		Author:      p.ddl.Metadata.Author,
		Timeout:     p.ddl.Metadata.Timeout,
		Name:        p.ddl.Metadata.Name,
		Version:     p.ddl.Metadata.Version,
		URL:         p.ddl.Metadata.URL,
		Description: p.ddl.Metadata.Description,
	}
}

// DiscoverNodes performs a discovery using the configured filter and node source
func (p *RpcutilClient) DiscoverNodes(ctx context.Context) (nodes []string, err error) {
	p.Lock()
	defer p.Unlock()

	return p.ns.Discover(ctx, p.fw, p.filters)
}

// AgentInventory performs the agent_inventory action
//
// Description: Inventory of all agents on the server including versions, licenses and more
func (p *RpcutilClient) AgentInventory() *AgentInventoryRequester {
	d := &AgentInventoryRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "agent_inventory",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// CollectiveInfo performs the collective_info action
//
// Description: Info about the main and sub collectives that the server belongs to
func (p *RpcutilClient) CollectiveInfo() *CollectiveInfoRequester {
	d := &CollectiveInfoRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "collective_info",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// DaemonStats performs the daemon_stats action
//
// Description: Get statistics from the running daemon
func (p *RpcutilClient) DaemonStats() *DaemonStatsRequester {
	d := &DaemonStatsRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "daemon_stats",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// GetConfigItem performs the get_config_item action
//
// Description: Get the active value of a specific config property
//
// Required Inputs:
//    - item (string) - The item to retrieve from the server
func (p *RpcutilClient) GetConfigItem(inputItem string) *GetConfigItemRequester {
	d := &GetConfigItemRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"item": inputItem,
			},
			action: "get_config_item",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// GetData performs the get_data action
//
// Description: Get data from a data plugin
//
// Required Inputs:
//    - source (string) - The data plugin to retrieve information from
//
// Optional Inputs:
//    - query (string) - The query argument to supply to the data plugin
func (p *RpcutilClient) GetData(inputSource string) *GetDataRequester {
	d := &GetDataRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"source": inputSource,
			},
			action: "get_data",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// GetFact performs the get_fact action
//
// Description: Retrieve a single fact from the fact store
//
// Required Inputs:
//    - fact (string) - The fact to retrieve
func (p *RpcutilClient) GetFact(inputFact string) *GetFactRequester {
	d := &GetFactRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"fact": inputFact,
			},
			action: "get_fact",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// GetFacts performs the get_facts action
//
// Description: Retrieve multiple facts from the fact store
//
// Required Inputs:
//    - facts (string) - Facts to retrieve
func (p *RpcutilClient) GetFacts(inputFacts string) *GetFactsRequester {
	d := &GetFactsRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"facts": inputFacts,
			},
			action: "get_facts",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Inventory performs the inventory action
//
// Description: System Inventory
func (p *RpcutilClient) Inventory() *InventoryRequester {
	d := &InventoryRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "inventory",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Ping performs the ping action
//
// Description: Responds to requests for PING with PONG
func (p *RpcutilClient) Ping() *PingRequester {
	d := &PingRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "ping",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}
