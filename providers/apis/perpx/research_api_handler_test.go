package perpx_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/1119-Labs/slinky/providers/apis/coinmarketcap"
	perpxtypes "github.com/1119-Labs/slinky/providers/apis/perpx/types"
	"github.com/1119-Labs/slinky/providers/base/testutils"
	"github.com/1119-Labs/slinky/providers/websockets/binance"
	"github.com/1119-Labs/slinky/providers/websockets/coinbase"
	"github.com/1119-Labs/slinky/providers/websockets/gate"
	"github.com/1119-Labs/slinky/providers/websockets/kucoin"
	"github.com/1119-Labs/slinky/providers/websockets/mexc"
	"github.com/1119-Labs/slinky/providers/websockets/okx"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/1119-Labs/slinky/oracle/config"
	slinkytypes "github.com/1119-Labs/slinky/pkg/types"
	"github.com/1119-Labs/slinky/providers/apis/perpx"
	"github.com/1119-Labs/slinky/service/clients/marketmap/types"
	mmtypes "github.com/1119-Labs/slinky/x/marketmap/types"
)

func TestNewResearchAPIHandler(t *testing.T) {
	t.Run("fail if the name is incorrect", func(t *testing.T) {
		_, err := perpx.NewResearchAPIHandler(zap.NewNop(), config.APIConfig{
			Name: "incorrect",
		})
		require.Error(t, err)
	})

	t.Run("fail if the api is not enabled", func(t *testing.T) {
		_, err := perpx.NewResearchAPIHandler(zap.NewNop(), config.APIConfig{
			Name:    perpx.ResearchAPIHandlerName,
			Enabled: false,
		})
		require.Error(t, err)
	})

	t.Run("test failure of api-config validation", func(t *testing.T) {
		cfg := perpx.DefaultResearchAPIConfig
		cfg.Endpoints = []config.Endpoint{
			{
				URL: "",
			},
		}

		_, err := perpx.NewResearchAPIHandler(zap.NewNop(), cfg)
		require.Error(t, err)
	})

	t.Run("test failure if no endpoint is given", func(t *testing.T) {
		cfg := perpx.DefaultResearchAPIConfig
		cfg.Endpoints = nil

		_, err := perpx.NewResearchAPIHandler(zap.NewNop(), cfg)
		require.Error(t, err)
	})

	t.Run("test success", func(t *testing.T) {
		_, err := perpx.NewResearchAPIHandler(zap.NewNop(), perpx.DefaultResearchAPIConfig)
		require.NoError(t, err)
	})
}

// TestCreateURL tests that:
//   - If no chain in the given chains are perpx - fail
//   - If one chain in the given chains is perpx - return the first endpoint configured
func TestCreateURLResearchHandler(t *testing.T) {
	ah, err := perpx.NewResearchAPIHandler(
		zap.NewNop(),
		perpx.DefaultResearchAPIConfig,
	)
	require.NoError(t, err)

	t.Run("non-perpx chains", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: "osmosis",
			},
		}

		url, err := ah.CreateURL(chains)
		require.Error(t, err)
		require.Empty(t, url)
	})
	t.Run("multiple chains w/ a perpx chain", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: "osmosis",
			},
			{
				ChainID: perpx.ChainID,
			},
		}

		url, err := ah.CreateURL(chains)
		require.NoError(t, err)
		require.Equal(t, perpx.DefaultResearchAPIConfig.Endpoints[1].URL, url)
	})
}

func TestParseResponseResearchAPI(t *testing.T) {
	ah, err := perpx.NewResearchAPIHandler(
		zap.NewNop(),
		perpx.DefaultResearchAPIConfig,
	)
	require.NoError(t, err)

	t.Run("fail if none of the chains given are perpx", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: "osmosis",
			},
		}

		resp := ah.ParseResponse(chains, &http.Response{})
		// expect a failure response for each chain
		require.Len(t, resp.UnResolved, 1)
		require.Len(t, resp.Resolved, 0)

		require.Error(t, resp.UnResolved[chains[0]])
	})

	t.Run("failing to parse ResearchJSON response", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: perpx.ChainID,
			},
		}

		resp := ah.ParseResponse(chains, &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("")),
		})

		require.Len(t, resp.UnResolved, 1)
		require.Len(t, resp.Resolved, 0)

		require.Error(t, resp.UnResolved[chains[0]])
	})

	t.Run("failing to convert ResearchJSON response into QueryAllMarketsParams", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: perpx.ChainID,
			},
		}

		resp := ah.ParseResponse(chains, &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewBufferString(`{
				"1INCH": {
				}
			}`)),
		})

		require.Len(t, resp.UnResolved, 1)
		require.Len(t, resp.Resolved, 0)

		require.Error(t, resp.UnResolved[chains[0]])
	})

	t.Run("success", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: perpx.ChainID,
			},
		}

		researchJSON := perpxtypes.ResearchJSON{
			"1INCH": {
				ResearchJSONMarketParam: perpxtypes.ResearchJSONMarketParam{
					ID:                0,
					Pair:              "1INCH-USD",
					Exponent:          -10.0,
					MinPriceChangePpm: 4000,
					MinExchanges:      3,
					ExchangeConfigJSON: []perpxtypes.ExchangeMarketConfigJson{
						{
							ExchangeName: "Binance",
							Ticker:       "1INCHUSDT",
						},
						{
							ExchangeName: "CoinbasePro",
							Ticker:       "1INCH-USD",
						},
						{
							ExchangeName: "Gate",
							Ticker:       "1INCH_USDT",
						},
						{
							ExchangeName: "Kucoin",
							Ticker:       "1INCH-USDT",
						},
						{
							ExchangeName: "Mexc",
							Ticker:       "1INCH_USDT",
						},
						{
							ExchangeName: "Okx",
							Ticker:       "1INCH-USDT",
						},
					},
				},
			},
		}
		bz, err := json.Marshal(researchJSON)
		require.NoError(t, err)

		resp := ah.ParseResponse(chains, testutils.CreateResponseFromJSON(string(bz)))

		require.Len(t, resp.UnResolved, 0)
		require.Len(t, resp.Resolved, 1)

		mm := resp.Resolved[chains[0]].Value.MarketMap
		require.Len(t, mm.Markets, 1)

		// index by the pair
		market, ok := mm.Markets["1INCH/USD"]
		require.True(t, ok)

		// check the ticker
		expectedTicker := mmtypes.Ticker{
			CurrencyPair:     slinkytypes.NewCurrencyPair("1INCH", "USD"),
			Decimals:         10,
			MinProviderCount: 3,
			Enabled:          true,
		}
		require.Equal(t, expectedTicker, market.Ticker)

		// check each provider
		expectedProviders := map[string]mmtypes.ProviderConfig{
			binance.Name: {
				Name:           binance.Name,
				OffChainTicker: "1INCHUSDT",
			},
			coinbase.Name: {
				Name:           coinbase.Name,
				OffChainTicker: "1INCH-USD",
			},
			gate.Name: {
				Name:           gate.Name,
				OffChainTicker: "1INCH_USDT",
			},
			kucoin.Name: {
				Name:           kucoin.Name,
				OffChainTicker: "1INCH-USDT",
			},
			mexc.Name: {
				Name:           mexc.Name,
				OffChainTicker: "1INCHUSDT",
			},
			okx.Name: {
				Name:           okx.Name,
				OffChainTicker: "1INCH-USDT",
			},
		}

		for _, provider := range market.ProviderConfigs {
			expectedProvider, ok := expectedProviders[provider.Name]
			require.True(t, ok)
			require.Equal(t, expectedProvider, provider)
		}
	})
}

func TestParseResponseResearchCMCAPI(t *testing.T) {
	ah, err := perpx.NewResearchAPIHandler(
		zap.NewNop(),
		perpx.DefaultResearchCMCAPIConfig,
	)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		chains := []types.Chain{
			{
				ChainID: perpx.ChainID,
			},
		}

		researchJSON := perpxtypes.ResearchJSON{
			"1INCH": {
				ResearchJSONMarketParam: perpxtypes.ResearchJSONMarketParam{
					ID:                0,
					Pair:              "1INCH-USD",
					Exponent:          -10.0,
					MinPriceChangePpm: 4000,
					MinExchanges:      3,
					ExchangeConfigJSON: []perpxtypes.ExchangeMarketConfigJson{
						{
							ExchangeName: "Binance",
							Ticker:       "1INCHUSDT",
						},
						{
							ExchangeName: "CoinbasePro",
							Ticker:       "1INCH-USD",
						},
						{
							ExchangeName: "Gate",
							Ticker:       "1INCH_USDT",
						},
						{
							ExchangeName: "Kucoin",
							Ticker:       "1INCH-USDT",
						},
						{
							ExchangeName: "Mexc",
							Ticker:       "1INCH_USDT",
						},
						{
							ExchangeName: "Okx",
							Ticker:       "1INCH-USDT",
						},
					},
				},
				MetaData: perpxtypes.MetaData{
					CMCID: 1,
				},
			},
		}

		bz, err := json.Marshal(researchJSON)
		require.NoError(t, err)

		resp := ah.ParseResponse(chains, testutils.CreateResponseFromJSON(string(bz)))

		require.Len(t, resp.UnResolved, 0)
		require.Len(t, resp.Resolved, 1)

		mm := resp.Resolved[chains[0]].Value.MarketMap
		require.Len(t, mm.Markets, 1)

		// index by the pair
		market, ok := mm.Markets["1INCH/USD"]
		require.True(t, ok)

		// check the ticker
		expectedTicker := mmtypes.Ticker{
			CurrencyPair:     slinkytypes.NewCurrencyPair("1INCH", "USD"),
			Decimals:         10,
			MinProviderCount: 1,
			Enabled:          true,
		}
		require.Equal(t, expectedTicker, market.Ticker)

		// check each provider
		expectedProviders := map[string]mmtypes.ProviderConfig{
			coinmarketcap.Name: {
				Name:           coinmarketcap.Name,
				OffChainTicker: "1",
			},
		}

		for _, provider := range market.ProviderConfigs {
			expectedProvider, ok := expectedProviders[provider.Name]
			require.True(t, ok)
			require.Equal(t, expectedProvider, provider)
		}
	})
}
