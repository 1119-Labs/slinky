package perpx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	slinkytypes "github.com/1119-Labs/slinky/pkg/types"
	"github.com/1119-Labs/slinky/providers/apis/perpx"
	apihandlermocks "github.com/1119-Labs/slinky/providers/base/api/handlers/mocks"
	providertypes "github.com/1119-Labs/slinky/providers/types"
	mmclient "github.com/1119-Labs/slinky/service/clients/marketmap/types"
	mmtypes "github.com/1119-Labs/slinky/x/marketmap/types"
)

func TestPERPXMultiMarketMapFetcher(t *testing.T) {
	perpxMainnetMMFetcher := apihandlermocks.NewAPIFetcher[mmclient.Chain, *mmtypes.MarketMapResponse](t)
	perpxResearchMMFetcher := apihandlermocks.NewAPIFetcher[mmclient.Chain, *mmtypes.MarketMapResponse](t)

	fetcher := perpx.NewPERPXResearchMarketMapFetcher(perpxMainnetMMFetcher, perpxResearchMMFetcher, zap.NewExample(), false)

	t.Run("test that if the mainnet api-price fetcher response is unresolved, we return it", func(t *testing.T) {
		ctx := context.Background()
		perpxMainnetMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			UnResolved: map[mmclient.Chain]providertypes.UnresolvedResult{
				perpx.PERPXChain: {
					ErrorWithCode: providertypes.NewErrorWithCode(fmt.Errorf("error"), providertypes.ErrorAPIGeneral),
				},
			},
		}, nil).Once()
		perpxResearchMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{}, nil).Once()

		response := fetcher.Fetch(ctx, []mmclient.Chain{perpx.PERPXChain})
		require.Len(t, response.UnResolved, 1)
	})

	t.Run("test that if the perpx-research response is unresolved, we return that", func(t *testing.T) {
		ctx := context.Background()
		perpxMainnetMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			Resolved: map[mmclient.Chain]providertypes.ResolvedResult[*mmtypes.MarketMapResponse]{
				perpx.PERPXChain: providertypes.NewResult(&mmtypes.MarketMapResponse{}, time.Now()),
			},
		}, nil).Once()
		perpxResearchMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			UnResolved: map[mmclient.Chain]providertypes.UnresolvedResult{
				perpx.PERPXChain: {},
			},
		}, nil).Once()

		response := fetcher.Fetch(ctx, []mmclient.Chain{perpx.PERPXChain})
		require.Len(t, response.UnResolved, 1)
	})

	t.Run("test if both responses are resolved, the tickers are appended to each other + validation fails", func(t *testing.T) {
		ctx := context.Background()
		perpxMainnetMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			Resolved: map[mmclient.Chain]providertypes.ResolvedResult[*mmtypes.MarketMapResponse]{
				perpx.PERPXChain: providertypes.NewResult(&mmtypes.MarketMapResponse{
					MarketMap: mmtypes.MarketMap{
						Markets: map[string]mmtypes.Market{
							"BTC/USD": {},
						},
					},
				}, time.Now()),
			},
		}, nil).Once()
		perpxResearchMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			Resolved: map[mmclient.Chain]providertypes.ResolvedResult[*mmtypes.MarketMapResponse]{
				perpx.PERPXChain: providertypes.NewResult(&mmtypes.MarketMapResponse{
					MarketMap: mmtypes.MarketMap{
						Markets: map[string]mmtypes.Market{
							"ETH/USD": {},
						},
					},
				}, time.Now()),
			},
		}, nil).Once()

		response := fetcher.Fetch(ctx, []mmclient.Chain{perpx.PERPXChain})
		require.Len(t, response.UnResolved, 1)
	})

	t.Run("test that if both responses are resolved, the responses are aggregated + validation passes", func(t *testing.T) {
		ctx := context.Background()
		perpxMainnetMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			Resolved: map[mmclient.Chain]providertypes.ResolvedResult[*mmtypes.MarketMapResponse]{
				perpx.PERPXChain: providertypes.NewResult(&mmtypes.MarketMapResponse{
					MarketMap: mmtypes.MarketMap{
						Markets: map[string]mmtypes.Market{
							"BTC/USD": {
								Ticker: mmtypes.Ticker{
									CurrencyPair:     slinkytypes.NewCurrencyPair("BTC", "USD"),
									Decimals:         8,
									MinProviderCount: 1,
									Enabled:          true,
								},
								ProviderConfigs: []mmtypes.ProviderConfig{
									{
										Name:           "perpx",
										OffChainTicker: "BTC/USD",
									},
								},
							},
						},
					},
				}, time.Now()),
			},
		}, nil).Once()
		perpxResearchMMFetcher.On("Fetch", ctx, []mmclient.Chain{perpx.PERPXChain}).Return(mmclient.MarketMapResponse{
			Resolved: map[mmclient.Chain]providertypes.ResolvedResult[*mmtypes.MarketMapResponse]{
				perpx.PERPXChain: providertypes.NewResult(&mmtypes.MarketMapResponse{
					MarketMap: mmtypes.MarketMap{
						Markets: map[string]mmtypes.Market{
							"ETH/USD": {
								Ticker: mmtypes.Ticker{
									CurrencyPair:     slinkytypes.NewCurrencyPair("ETH", "USD"),
									Decimals:         8,
									MinProviderCount: 1,
								},
								ProviderConfigs: []mmtypes.ProviderConfig{
									{
										Name:           "perpx",
										OffChainTicker: "BTC/USD",
									},
								},
							},
						},
					},
				}, time.Now()),
			},
		}, nil).Once()

		response := fetcher.Fetch(ctx, []mmclient.Chain{perpx.PERPXChain})
		require.Len(t, response.Resolved, 1)

		marketMap := response.Resolved[perpx.PERPXChain].Value.MarketMap

		require.Len(t, marketMap.Markets, 2)
		require.Contains(t, marketMap.Markets, "BTC/USD")
		require.Contains(t, marketMap.Markets, "ETH/USD")
	})
}
