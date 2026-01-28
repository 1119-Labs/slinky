package tickermetadata

import "encoding/json"

// Perpx is the Ticker.Metadata_JSON published to every Ticker in the x/marketmap module on Perpx.
type Perpx struct {
	// ReferencePrice gives a spot price for that Ticker at the point in time when the ReferencePrice was updated.
	// You should _not_ use this for up-to-date/instantaneous spot pricing data since it is updated infrequently.
	// The price is scaled by Ticker.Decimals.
	ReferencePrice uint64 `json:"reference_price"`
	// Liquidity gives a _rough_ estimate of the amount of liquidity in the Providers for a given Market.
	// It is _not_ updated in coordination with spot prices and only gives rough order of magnitude accuracy at the time
	// which the update for it is published.
	// The liquidity value stored here is USD denominated.
	Liquidity uint64 `json:"liquidity"`
	// AggregateIDs contains a list of AggregatorIDs associated with the ticker.
	// This field may not be populated if no aggregator currently indexes this Ticker.
	AggregateIDs []AggregatorID `json:"aggregate_ids"`
	// CrossLaunch is an optional bool that indicates whether this ticker should be
	// launched as a cross-margin market (instead of isolated margin).
	// If omitted, it is set to false by default.
	CrossLaunch bool `json:"cross_launch,omitempty"`
}

// NewPerpx returns a new Perpx instance.
func NewPerpx(referencePrice, liquidity uint64, aggregateIDs []AggregatorID, crossLaunch bool) Perpx {
	return Perpx{
		ReferencePrice: referencePrice,
		Liquidity:      liquidity,
		AggregateIDs:   aggregateIDs,
		CrossLaunch:    crossLaunch,
	}
}

// MarshalPerpx returns the JSON byte encoding of the Perpx.
func MarshalPerpx(m Perpx) ([]byte, error) {
	return json.Marshal(m)
}

// PerpxFromJSONString returns a Perpx instance from a JSON string.
func PerpxFromJSONString(jsonString string) (Perpx, error) {
	var elem Perpx
	err := json.Unmarshal([]byte(jsonString), &elem)
	return elem, err
}

// PerpxFromJSONBytes returns a Perpx instance from JSON bytes.
func PerpxFromJSONBytes(jsonBytes []byte) (Perpx, error) {
	var elem Perpx
	err := json.Unmarshal(jsonBytes, &elem)
	return elem, err
}
