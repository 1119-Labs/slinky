package perpx_test

import (
	"fmt"
	"net/http"
	"testing"

	perpx "github.com/1119-Labs/slinky/providers/apis/perpx"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/1119-Labs/slinky/providers/base/testutils"
	providertypes "github.com/1119-Labs/slinky/providers/types"
	"github.com/1119-Labs/slinky/service/clients/marketmap/types"
)

var chains = []types.Chain{
	{
		ChainID: "Perpx",
	},
	{
		ChainID: "osmosis",
	},
}

func TestCreateURL(t *testing.T) {
	handler, err := perpx.NewAPIHandler(zap.NewNop(), perpx.DefaultAPIConfig)
	require.NoError(t, err)

	t.Run("multiple chains", func(t *testing.T) {
		_, err := handler.CreateURL(chains)
		require.Error(t, err)
	})

	t.Run("single chain", func(t *testing.T) {
		url, err := handler.CreateURL(chains[:1])
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf(perpx.Endpoint, perpx.DefaultAPIConfig.Endpoints[0].URL), url)
	})
}

func TestParseResponse(t *testing.T) {
	testCases := []struct {
		name     string
		chains   []types.Chain
		resp     func() *http.Response
		expected types.MarketMapResponse
	}{
		{
			name:   "multiple chains",
			chains: chains,
			resp:   func() *http.Response { return nil },
			expected: types.MarketMapResponse{
				UnResolved: types.UnResolvedMarketMap{
					chains[0]: providertypes.UnresolvedResult{
						ErrorWithCode: providertypes.NewErrorWithCode(fmt.Errorf("expected one chain, got 2"), providertypes.ErrorAPIGeneral),
					},
					chains[1]: providertypes.UnresolvedResult{
						ErrorWithCode: providertypes.NewErrorWithCode(fmt.Errorf("expected one chain, got 2"), providertypes.ErrorAPIGeneral),
					},
				},
			},
		},
		{
			name:   "nil response",
			chains: chains[:1],
			resp:   func() *http.Response { return nil },
			expected: types.MarketMapResponse{
				UnResolved: types.UnResolvedMarketMap{
					chains[0]: providertypes.UnresolvedResult{
						ErrorWithCode: providertypes.NewErrorWithCode(fmt.Errorf("nil response"), providertypes.ErrorAPIGeneral),
					},
				},
			},
		},
		{
			name:   "errors when the response body cannot be parsed",
			chains: chains[:1],
			resp: func() *http.Response {
				return testutils.CreateResponseFromJSON("invalid json")
			},
			expected: types.MarketMapResponse{
				UnResolved: types.UnResolvedMarketMap{
					chains[0]: providertypes.UnresolvedResult{
						ErrorWithCode: providertypes.NewErrorWithCode(fmt.Errorf("failed to parse market map response"), providertypes.ErrorAPIGeneral),
					},
				},
			},
		},
		{
			name:   "errors when the response body cannot be converted to a market map",
			chains: chains[:1],
			resp: func() *http.Response {
				return testutils.CreateResponseFromJSON(PerpxResponseInvalid)
			},
			expected: types.MarketMapResponse{
				UnResolved: types.UnResolvedMarketMap{
					chains[0]: providertypes.UnresolvedResult{
						ErrorWithCode: providertypes.NewErrorWithCode(fmt.Errorf("failed to convert market map response"), providertypes.ErrorUnknown),
					},
				},
			},
		},
		{
			name:   "successful response",
			chains: chains[:1],
			resp: func() *http.Response {
				return testutils.CreateResponseFromJSON(PerpxResponseValid)
			},
			expected: types.MarketMapResponse{
				Resolved: types.ResolvedMarketMap{
					chains[0]: types.MarketMapResult{
						Value: &convertedResponse,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := perpx.NewAPIHandler(zap.NewNop(), perpx.DefaultAPIConfig)
			require.NoError(t, err)

			resp := handler.ParseResponse(tc.chains, tc.resp())
			require.Len(t, resp.Resolved, len(tc.expected.Resolved))
			require.Len(t, resp.UnResolved, len(tc.expected.UnResolved))

			for cp, result := range tc.expected.Resolved {
				require.Contains(t, resp.Resolved, cp)
				r := resp.Resolved[cp]
				require.Equal(t, result.Value, r.Value)
			}

			for cp := range tc.expected.UnResolved {
				require.Contains(t, resp.UnResolved, cp)
				require.Error(t, resp.UnResolved[cp])
			}
		})
	}
}
