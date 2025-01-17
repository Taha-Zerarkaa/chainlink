package configtest

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/client"
	evmclient "github.com/smartcontractkit/chainlink/v2/core/chains/evm/client"
	evmcfg "github.com/smartcontractkit/chainlink/v2/core/chains/evm/config/toml"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/services/chainlink"
	"github.com/smartcontractkit/chainlink/v2/core/store/dialects"
	"github.com/smartcontractkit/chainlink/v2/core/store/models"
	"github.com/smartcontractkit/chainlink/v2/core/utils"
)

const DefaultPeerID = "12D3KooWPjceQrSwdWXPyLLeABRXmuqt69Rg3sBYbU1Nft9HyQ6X"

// NewTestGeneralConfig returns a new chainlink.GeneralConfig with default test overrides and one chain with evmclient.NullClientChainID.
func NewTestGeneralConfig(t testing.TB) chainlink.GeneralConfig { return NewGeneralConfig(t, nil) }

// NewGeneralConfig returns a new chainlink.GeneralConfig with overrides.
// The default test overrides are applied before overrideFn, and include one chain with evmclient.NullClientChainID.
func NewGeneralConfig(t testing.TB, overrideFn func(*chainlink.Config, *chainlink.Secrets)) chainlink.GeneralConfig {
	tempDir := t.TempDir()
	g, err := chainlink.GeneralConfigOpts{
		OverrideFn: func(c *chainlink.Config, s *chainlink.Secrets) {
			overrides(c, s)
			c.RootDir = &tempDir
			if fn := overrideFn; fn != nil {
				fn(c, s)
			}
		},
	}.New()
	require.NoError(t, err)
	return g
}

// overrides applies some test config settings and adds a default chain with evmclient.NullClientChainID.
func overrides(c *chainlink.Config, s *chainlink.Secrets) {
	s.Password.Keystore = models.NewSecret("dummy-to-pass-validation")

	c.Insecure.OCRDevelopmentMode = ptr(true)
	c.InsecureFastScrypt = ptr(true)
	c.ShutdownGracePeriod = models.MustNewDuration(testutils.DefaultWaitTimeout)

	c.Database.Dialect = dialects.TransactionWrappedPostgres
	c.Database.Lock.Enabled = ptr(false)
	c.Database.MaxIdleConns = ptr[int64](20)
	c.Database.MaxOpenConns = ptr[int64](20)
	c.Database.MigrateOnStartup = ptr(false)
	c.Database.DefaultLockTimeout = models.MustNewDuration(1 * time.Minute)

	c.JobPipeline.ReaperInterval = models.MustNewDuration(0)

	c.P2P.V2.Enabled = ptr(false)

	c.WebServer.SessionTimeout = models.MustNewDuration(2 * time.Minute)
	c.WebServer.BridgeResponseURL = models.MustParseURL("http://localhost:6688")
	testIP := net.ParseIP("127.0.0.1")
	c.WebServer.ListenIP = &testIP
	c.WebServer.TLS.ListenIP = &testIP

	chainID := utils.NewBigI(evmclient.NullClientChainID)
	c.EVM = append(c.EVM, &evmcfg.EVMConfig{
		ChainID: chainID,
		Chain:   evmcfg.Defaults(chainID),
		Nodes: evmcfg.EVMNodes{
			&evmcfg.Node{
				Name:     ptr("test"),
				WSURL:    &models.URL{},
				HTTPURL:  &models.URL{},
				SendOnly: new(bool),
				Order:    ptr[int32](100),
			},
		},
	})
}

// NewGeneralConfigSimulated returns a new chainlink.GeneralConfig with overrides, including the simulated EVM chain.
// The default test overrides are applied before overrideFn.
// The simulated chain (testutils.SimulatedChainID) replaces the null chain (evmclient.NullClientChainID).
func NewGeneralConfigSimulated(t testing.TB, overrideFn func(*chainlink.Config, *chainlink.Secrets)) chainlink.GeneralConfig {
	return NewGeneralConfig(t, func(c *chainlink.Config, s *chainlink.Secrets) {
		simulated(c, s)
		if fn := overrideFn; fn != nil {
			fn(c, s)
		}
	})
}

// simulated is a config override func that appends the simulated EVM chain (testutils.SimulatedChainID),
// or replaces the null chain (client.NullClientChainID) if that is the only entry.
func simulated(c *chainlink.Config, s *chainlink.Secrets) {
	chainID := utils.NewBig(testutils.SimulatedChainID)
	enabled := true
	cfg := evmcfg.EVMConfig{
		ChainID: chainID,
		Chain:   evmcfg.Defaults(chainID),
		Enabled: &enabled,
		Nodes:   evmcfg.EVMNodes{&validTestNode},
	}
	if len(c.EVM) == 1 && c.EVM[0].ChainID.Cmp(utils.NewBigI(client.NullClientChainID)) == 0 {
		c.EVM[0] = &cfg // replace null, if only entry
	} else {
		c.EVM = append(c.EVM, &cfg)
	}
}

var validTestNode = evmcfg.Node{
	Name:     ptr("simulated-node"),
	WSURL:    models.MustParseURL("WSS://simulated-wss.com/ws"),
	HTTPURL:  models.MustParseURL("http://simulated.com"),
	SendOnly: nil,
	Order:    ptr(int32(1)),
}

func ptr[T any](v T) *T { return &v }
