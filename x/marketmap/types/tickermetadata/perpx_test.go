package tickermetadata_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/1119-Labs/slinky/x/marketmap/types/tickermetadata"
)

func Test_UnmarshalPerpx(t *testing.T) {
	t.Run("can marshal and unmarshal the same struct and values", func(t *testing.T) {
		elem := tickermetadata.NewPerpx(
			100,
			1000,
			[]tickermetadata.AggregatorID{
				tickermetadata.NewAggregatorID("coingecko", "id"),
				tickermetadata.NewAggregatorID("cmc", "id"),
			},
			false,
		)

		bz, err := tickermetadata.MarshalPerpx(elem)
		require.NoError(t, err)

		elem2, err := tickermetadata.PerpxFromJSONBytes(bz)
		require.NoError(t, err)
		require.Equal(t, elem, elem2)
	})

	t.Run("can marshal and unmarshal the same struct and values with empty AggregatorIDs", func(t *testing.T) {
		elem := tickermetadata.NewPerpx(100, 1000, nil, false)

		bz, err := tickermetadata.MarshalPerpx(elem)
		require.NoError(t, err)

		elem2, err := tickermetadata.PerpxFromJSONBytes(bz)
		require.NoError(t, err)
		require.Equal(t, elem, elem2)
	})

	t.Run("can unmarshal a JSON string into a struct", func(t *testing.T) {
		elemJSON := `{"reference_price":100,"liquidity":1000,"aggregate_ids":[{"venue":"coingecko","ID":"id"},{"venue":"cmc","ID":"id"}]}`
		elem, err := tickermetadata.PerpxFromJSONString(elemJSON)
		require.NoError(t, err)

		require.Equal(t, tickermetadata.NewPerpx(
			100,
			1000,
			[]tickermetadata.AggregatorID{
				tickermetadata.NewAggregatorID("coingecko", "id"),
				tickermetadata.NewAggregatorID("cmc", "id"),
			},
			false,
		), elem)
	})

	t.Run("can unmarshal a JSON string into a struct with empty CrossLaunch field", func(t *testing.T) {
		elemJSON := `{"reference_price":100,"liquidity":1000,"aggregate_ids":[{"venue":"coingecko","ID":"id"},{"venue":"cmc","ID":"id"}]}`
		elem, err := tickermetadata.PerpxFromJSONString(elemJSON)
		require.NoError(t, err)

		require.Equal(t, tickermetadata.NewPerpx(
			100,
			1000,
			[]tickermetadata.AggregatorID{
				tickermetadata.NewAggregatorID("coingecko", "id"),
				tickermetadata.NewAggregatorID("cmc", "id"),
			},
			false,
		), elem)
	})

	t.Run("can unmarshal a JSON string into a struct with CrossLaunch field set", func(t *testing.T) {
		elemJSON := `{"reference_price":100,"liquidity":1000,"aggregate_ids":[{"venue":"coingecko","ID":"id"},{"venue":"cmc","ID":"id"}],"cross_launch":true}`
		elem, err := tickermetadata.PerpxFromJSONString(elemJSON)
		require.NoError(t, err)

		require.Equal(t, tickermetadata.NewPerpx(
			100,
			1000,
			[]tickermetadata.AggregatorID{
				tickermetadata.NewAggregatorID("coingecko", "id"),
				tickermetadata.NewAggregatorID("cmc", "id"),
			},
			true,
		), elem)
	})
}
