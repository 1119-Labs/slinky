package perpx

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/1119-Labs/slinky/cmd/constants/marketmaps"
	"github.com/1119-Labs/slinky/oracle/config"
	"github.com/1119-Labs/slinky/providers/apis/coinmarketcap"
	apihandlers "github.com/1119-Labs/slinky/providers/base/api/handlers"
	"github.com/1119-Labs/slinky/providers/base/api/metrics"
	providertypes "github.com/1119-Labs/slinky/providers/types"
	mmclient "github.com/1119-Labs/slinky/service/clients/marketmap/types"
	mmtypes "github.com/1119-Labs/slinky/x/marketmap/types"
)

var (
	_         mmclient.MarketMapFetcher = &MultiMarketMapRestAPIFetcher{}
	PERPXChain                           = mmclient.Chain{
		ChainID: ChainID,
	}
)

// NewPERPXResearchMarketMapFetcher returns a MultiMarketMapFetcher composed of perpx mainnet + research
// apiDataHandlers.
func DefaultPERPXResearchMarketMapFetcher(
	rh apihandlers.RequestHandler,
	metrics metrics.APIMetrics,
	api config.APIConfig,
	logger *zap.Logger,
) (*MultiMarketMapRestAPIFetcher, error) {
	if rh == nil {
		return nil, fmt.Errorf("request handler is nil")
	}

	if metrics == nil {
		return nil, fmt.Errorf("metrics is nil")
	}

	if !api.Enabled {
		return nil, fmt.Errorf("api is not enabled")
	}

	if err := api.ValidateBasic(); err != nil {
		return nil, err
	}

	if len(api.Endpoints) != 2 {
		return nil, fmt.Errorf("expected two endpoint, got %d", len(api.Endpoints))
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	// make a perpx research api-handler
	researchAPIDataHandler, err := NewResearchAPIHandler(logger, api)
	if err != nil {
		return nil, err
	}

	mainnetAPIDataHandler := &APIHandler{
		logger: logger,
		api:    api,
	}

	mainnetFetcher, err := apihandlers.NewRestAPIFetcher(
		rh,
		mainnetAPIDataHandler,
		metrics,
		api,
		logger,
	)
	if err != nil {
		return nil, err
	}

	researchFetcher, err := apihandlers.NewRestAPIFetcher(
		rh,
		researchAPIDataHandler,
		metrics,
		api,
		logger,
	)
	if err != nil {
		return nil, err
	}

	return NewPERPXResearchMarketMapFetcher(
		mainnetFetcher,
		researchFetcher,
		logger,
		api.Name == ResearchCMCAPIHandlerName,
	), nil
}

// MultiMarketMapRestAPIFetcher is an implementation of a RestAPIFetcher that wraps
// two underlying Fetchers for fetching the market-map according to perpx mainnet and
// the additional markets that can be added according to the perpx research json.
type MultiMarketMapRestAPIFetcher struct {
	// perpx mainnet fetcher is the api-fetcher for the perpx mainnet market-map
	perpxMainnetFetcher mmclient.MarketMapFetcher

	// perpx research fetcher is the api-fetcher for the perpx research market-map
	perpxResearchFetcher mmclient.MarketMapFetcher

	// logger is the logger for the fetcher
	logger *zap.Logger

	// isCMCOnly is a flag that indicates whether the fetcher should only return CoinMarketCap markets.
	isCMCOnly bool
}

// NewPERPXResearchMarketMapFetcher returns an aggregated market-map among the perpx mainnet and the perpx research json.
func NewPERPXResearchMarketMapFetcher(
	mainnetFetcher, researchFetcher mmclient.MarketMapFetcher,
	logger *zap.Logger,
	isCMCOnly bool,
) *MultiMarketMapRestAPIFetcher {
	return &MultiMarketMapRestAPIFetcher{
		perpxMainnetFetcher:  mainnetFetcher,
		perpxResearchFetcher: researchFetcher,
		logger:              logger.With(zap.String("module", "perpx-research-market-map-fetcher")),
		isCMCOnly:           isCMCOnly,
	}
}

// Fetch fetches the market map from the underlying fetchers and combines the results. If any of the underlying
// fetchers fetch for a chain that is different from the chain that the fetcher is initialized with, those responses
// will be ignored.
func (f *MultiMarketMapRestAPIFetcher) Fetch(ctx context.Context, chains []mmclient.Chain) mmclient.MarketMapResponse {
	// call the underlying fetchers + await their responses
	// channel to aggregate responses
	perpxMainnetResponseChan := make(chan mmclient.MarketMapResponse, 1) // buffer so that sends / receives are non-blocking
	perpxResearchResponseChan := make(chan mmclient.MarketMapResponse, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	// fetch perpx mainnet
	go func() {
		defer wg.Done()
		perpxMainnetResponseChan <- f.perpxMainnetFetcher.Fetch(ctx, chains)
		f.logger.Debug("fetched valid market-map from perpx mainnet")
	}()

	// fetch perpx research
	go func() {
		defer wg.Done()
		perpxResearchResponseChan <- f.perpxResearchFetcher.Fetch(ctx, chains)
		f.logger.Debug("fetched valid market-map from perpx research")
	}()

	// wait for both fetchers to finish
	wg.Wait()

	perpxMainnetMarketMapResponse := <-perpxMainnetResponseChan
	perpxResearchMarketMapResponse := <-perpxResearchResponseChan

	// if the perpx mainnet market-map response failed, return the perpx mainnet failed response
	if _, ok := perpxMainnetMarketMapResponse.UnResolved[PERPXChain]; ok {
		f.logger.Error("perpx mainnet market-map fetch failed", zap.Any("response", perpxMainnetMarketMapResponse))
		return perpxMainnetMarketMapResponse
	}

	// if the perpx research market-map response failed, return the perpx research failed response
	if _, ok := perpxResearchMarketMapResponse.UnResolved[PERPXChain]; ok {
		f.logger.Error("perpx research market-map fetch failed", zap.Any("response", perpxResearchMarketMapResponse))
		return perpxResearchMarketMapResponse
	}

	// otherwise, add all markets from perpx research
	perpxMainnetMarketMap := perpxMainnetMarketMapResponse.Resolved[PERPXChain].Value.MarketMap

	resolved, ok := perpxResearchMarketMapResponse.Resolved[PERPXChain]
	if ok {
		for ticker, market := range resolved.Value.MarketMap.Markets {
			// if the market is not already in the perpx mainnet market-map, add it
			if _, ok := perpxMainnetMarketMap.Markets[ticker]; !ok {
				f.logger.Debug("adding market from perpx research", zap.String("ticker", ticker))
				perpxMainnetMarketMap.Markets[ticker] = market
			}
		}
	}

	// if the fetcher is only for CoinMarketCap markets, filter out all non-CMC markets
	if f.isCMCOnly {
		for ticker, market := range perpxMainnetMarketMap.Markets {
			market.Ticker.MinProviderCount = 1
			perpxMainnetMarketMap.Markets[ticker] = market

			var (
				seenCMC     = false
				cmcProvider mmtypes.ProviderConfig
			)

			for _, provider := range market.ProviderConfigs {
				if provider.Name == coinmarketcap.Name {
					seenCMC = true
					cmcProvider = provider
				}
			}

			// if we saw a CMC provider, add it to the market
			if seenCMC {
				market.ProviderConfigs = []mmtypes.ProviderConfig{cmcProvider}
				perpxMainnetMarketMap.Markets[ticker] = market
				continue
			}

			// If we did not see a CMC provider, we can attempt to add it using the CMC marketmap
			cmcMarket, ok := marketmaps.CoinMarketCapMarketMap.Markets[ticker]
			if !ok {
				f.logger.Info("did not find CMC market for ticker", zap.String("ticker", ticker))
				delete(perpxMainnetMarketMap.Markets, ticker)
				continue
			}

			// add the CMC provider to the market
			market.ProviderConfigs = cmcMarket.ProviderConfigs
			perpxMainnetMarketMap.Markets[ticker] = market
		}
	}

	// validate the combined market-map
	if err := perpxMainnetMarketMap.ValidateBasic(); err != nil {
		f.logger.Error("combined market-map failed validation", zap.Error(err))

		return mmclient.NewMarketMapResponseWithErr(
			chains,
			providertypes.NewErrorWithCode(
				fmt.Errorf("combined market-map failed validation: %w", err),
				providertypes.ErrorUnknown,
			),
		)
	}

	perpxMainnetMarketMapResponse.Resolved[PERPXChain].Value.MarketMap = perpxMainnetMarketMap

	return perpxMainnetMarketMapResponse
}
