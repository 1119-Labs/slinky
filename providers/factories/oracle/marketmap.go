package oracle

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/1119-Labs/slinky/oracle/config"
	perpx "github.com/1119-Labs/slinky/providers/apis/perpx"
	"github.com/1119-Labs/slinky/providers/apis/marketmap"
	"github.com/1119-Labs/slinky/providers/base"
	apihandlers "github.com/1119-Labs/slinky/providers/base/api/handlers"
	apimetrics "github.com/1119-Labs/slinky/providers/base/api/metrics"
	providermetrics "github.com/1119-Labs/slinky/providers/base/metrics"
	"github.com/1119-Labs/slinky/service/clients/marketmap/types"
	mmtypes "github.com/1119-Labs/slinky/x/marketmap/types"
)

// MarketMapProviderFactory returns a sample implementation of the market map provider. This provider
// is responsible for fetching updates to the canonical market map on the given chain.
func MarketMapProviderFactory(
	logger *zap.Logger,
	providerMetrics providermetrics.ProviderMetrics,
	apiMetrics apimetrics.APIMetrics,
	cfg config.ProviderConfig,
) (*types.MarketMapProvider, error) {
	// Validate the provider config.
	err := cfg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: cfg.API.MaxQueries,
			Proxy:           http.ProxyFromEnvironment,
		},
		Timeout: cfg.API.Timeout,
	}

	var (
		apiDataHandler   types.MarketMapAPIDataHandler
		ids              []types.Chain
		marketMapFetcher types.MarketMapFetcher
	)

	requestHandler, err := apihandlers.NewRequestHandlerImpl(client)
	if err != nil {
		return nil, err
	}

	switch cfg.Name {
	case perpx.Name:
		apiDataHandler, err = perpx.NewAPIHandler(logger, cfg.API)
		ids = []types.Chain{{ChainID: perpx.ChainID}}
	case perpx.SwitchOverAPIHandlerName:
		marketMapFetcher, err = perpx.NewDefaultSwitchOverMarketMapFetcher(
			logger,
			cfg.API,
			requestHandler,
			apiMetrics,
		)
		ids = []types.Chain{{ChainID: perpx.ChainID}}
	case perpx.ResearchAPIHandlerName, perpx.ResearchCMCAPIHandlerName:
		marketMapFetcher, err = perpx.DefaultPERPXResearchMarketMapFetcher(
			requestHandler,
			apiMetrics,
			cfg.API,
			logger,
		)
		ids = []types.Chain{{ChainID: perpx.ChainID}}
	default:
		marketMapFetcher, err = marketmap.NewMarketMapFetcher(
			logger,
			cfg.API,
			apiMetrics,
		)
		ids = []types.Chain{{ChainID: "local-node"}}
	}
	if err != nil {
		return nil, err
	}

	if marketMapFetcher == nil {
		marketMapFetcher, err = apihandlers.NewRestAPIFetcher(
			requestHandler,
			apiDataHandler,
			apiMetrics,
			cfg.API,
			logger,
		)
		if err != nil {
			return nil, err
		}
	}

	queryHandler, err := types.NewMarketMapAPIQueryHandlerWithMarketMapFetcher(
		logger,
		cfg.API,
		marketMapFetcher,
		apiMetrics,
	)
	if err != nil {
		return nil, err
	}

	return types.NewMarketMapProvider(
		base.WithName[types.Chain, *mmtypes.MarketMapResponse](cfg.Name),
		base.WithLogger[types.Chain, *mmtypes.MarketMapResponse](logger),
		base.WithAPIQueryHandler(queryHandler),
		base.WithAPIConfig[types.Chain, *mmtypes.MarketMapResponse](cfg.API),
		base.WithMetrics[types.Chain, *mmtypes.MarketMapResponse](providerMetrics),
		base.WithIDs[types.Chain, *mmtypes.MarketMapResponse](ids),
	)
}
