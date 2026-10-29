package deribitws_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gen-asyncapi/asyncapirt"
	ws "github.com/psyb0t/go-deribit/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	indexMessage    = ws.MessageDeribitPriceIndexIndexNameSubscriptionMessage
	rawOrderMessage = ws.MessageUserOrdersInstrumentNameRawSubscriptionMessage
	fixturePrice    = 123.456789
	requestID       = 42
	publicMethod    = "public/subscribe"
	privateMethod   = "private/subscribe"
)

func TestDecodeSubscriptionFixtures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		fixture string
		name    string
	}{
		{
			fixture: "testdata/platform-state.json",
			name:    ws.MessagePlatformStateSubscriptionMessage,
		},
		{
			fixture: "testdata/price-index.json",
			name:    indexMessage,
		},
		{
			fixture: "testdata/quote.json",
			name:    ws.MessageQuoteInstrumentNameSubscriptionMessage,
		},
		{
			fixture: "testdata/raw-order.json",
			name:    rawOrderMessage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fixture, func(t *testing.T) {
			t.Parallel()

			frame, err := os.ReadFile(tc.fixture)
			require.NoError(t, err)

			message, err := ws.DecodeReceivedMessage(frame)
			require.NoError(t, err)
			assert.Equal(t, tc.name, message.Name)
			assertFixturePayload(t, message.Payload)
		})
	}
}

func assertFixturePayload(t *testing.T, value any) {
	t.Helper()

	switch payload := value.(type) {
	case ws.PlatformStateSubscriptionMessagePayload:
		require.NotNil(t, payload.Params.Data.Maintenance)
		assert.True(t, *payload.Params.Data.Maintenance)
	case ws.DeribitPriceIndexIndexNameSubscriptionMessagePayload:
		assert.InDelta(t, fixturePrice, payload.Params.Data.Price, 1e-10)
		assert.Equal(t, "btc_usd", string(payload.Params.Data.IndexName))
	case ws.QuoteInstrumentNameSubscriptionMessagePayload:
		assert.Nil(t, payload.Params.Data.BestBidPrice)
		assert.Nil(t, payload.Params.Data.BestAskPrice)
		require.NotNil(t, payload.Params.Data.BestAskAmount)
		assert.Zero(t, *payload.Params.Data.BestAskAmount)
	case ws.UserOrdersInstrumentNameRawSubscriptionMessagePayload:
		assert.Equal(t, "example-order", payload.Params.Data.OrderID)
		assert.Equal(t, "open", string(payload.Params.Data.OrderState))
	default:
		t.Fatalf("unexpected payload type %T", value)
	}
}

func TestDecodeRejectsInvalidFrames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		frame   string
		wantErr error
	}{
		{
			name:    "malformed JSON",
			frame:   "{",
			wantErr: commerr.ErrCouldNotDecode,
		},
		{
			name: "unknown channel",
			frame: `{"jsonrpc":"2.0","method":"subscription",
				"params":{"channel":"unknown","data":{}}}`,
			wantErr: asyncapirt.ErrUnknownMessage,
		},
		{
			name: "missing notification data",
			frame: `{"jsonrpc":"2.0","method":"subscription",
				"params":{"channel":"platform_state"}}`,
			wantErr: asyncapirt.ErrUnknownMessage,
		},
		{
			name: "invalid JSON-RPC version",
			frame: `{"jsonrpc":"1.0","method":"subscription",
				"params":{"channel":"platform_state","data":{}}}`,
			wantErr: asyncapirt.ErrUnknownMessage,
		},
		{
			name: "invalid method",
			frame: `{"jsonrpc":"2.0","method":"other",
				"params":{"channel":"platform_state","data":{}}}`,
			wantErr: asyncapirt.ErrUnknownMessage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ws.DecodeReceivedMessage([]byte(tc.frame))
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

type recordingTransport struct {
	sent [][]byte
}

func (transport *recordingTransport) WriteMessage(
	_ context.Context,
	data []byte,
) error {
	transport.sent = append(transport.sent, data)

	return nil
}

func (transport *recordingTransport) ReadMessage(
	context.Context,
) ([]byte, error) {
	return nil, ctxerrors.Wrap(context.Canceled, "read test transport")
}

func (transport *recordingTransport) Close() error {
	return nil
}

func TestClientSendsSubscriptionEnvelopes(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	client := ws.NewClient(transport)

	err := client.SendSubscribePlatformState(
		t.Context(),
		ws.PlatformStateSubscribeRequestPayload{
			ID: requestID,
			Params: ws.PlatformStateSubscribeRequestPayloadParams{
				Channels: []string{ws.PlatformStateAddressTemplate},
			},
		},
	)
	require.NoError(t, err)

	err = client.SendSubscribeUserLock(
		t.Context(),
		ws.UserLockSubscribeRequestPayload{
			ID: requestID,
			Params: ws.UserLockSubscribeRequestPayloadParams{
				Channels: []string{ws.UserLockAddressTemplate},
			},
		},
	)
	require.NoError(t, err)

	blockChannel := ws.BlockRfqTradesCurrencyAddress("BTC")

	var params ws.BlockRfqTradesCurrencySubscribeRequestPayloadParams

	params.Channels = []string{blockChannel}
	err = client.SendSubscribeBlockRfqTradesCurrency(
		t.Context(),
		ws.BlockRfqTradesCurrencySubscribeRequestPayload{
			ID:     requestID,
			Params: params,
		},
	)
	require.NoError(t, err)
	require.Len(t, transport.sent, 3)
	assertSentFrame(t, transport.sent[0], publicMethod,
		ws.PlatformStateAddressTemplate)
	assertSentFrame(t, transport.sent[1], privateMethod,
		ws.UserLockAddressTemplate)
	assertSentFrame(t, transport.sent[2], publicMethod, blockChannel)
	require.NoError(t, client.Close())
}

func assertSentFrame(
	t *testing.T,
	data []byte,
	method, channel string,
) {
	t.Helper()

	var frame struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Method  string `json:"method"`
		Params  struct {
			Channels []string `json:"channels"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal(data, &frame))
	assert.Equal(t, "2.0", frame.Jsonrpc)
	assert.Equal(t, int64(requestID), frame.ID)
	assert.Equal(t, method, frame.Method)
	assert.Equal(t, []string{channel}, frame.Params.Channels)
}

func TestClientRejectsInvalidChannelBeforeSending(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		channels []string
	}{
		{name: "empty channels"},
		{name: "wrong channel", channels: []string{"other"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			transport := &recordingTransport{}
			client := ws.NewClient(transport)

			var payload ws.PlatformStateSubscribeRequestPayload

			payload.ID = requestID
			payload.Params.Channels = tc.channels

			err := client.SendSubscribePlatformState(t.Context(), payload)
			require.ErrorIs(t, err, commerr.ErrValidationFailed)
			assert.Empty(t, transport.sent)
		})
	}
}

func TestChannelAddressReplacesParameters(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "deribit_price_index.btc_usd",
		ws.DeribitPriceIndexIndexNameAddress("btc_usd"))
}

func TestDecodeCorrectedWireShapes(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"testdata/trades.json",
		"testdata/block-confirmation.json",
		"testdata/rfq-taker.json",
		"testdata/user-trades.json",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()

			frame, err := os.ReadFile(tc)
			require.NoError(t, err)

			message, err := ws.DecodeReceivedMessage(frame)
			require.NoError(t, err)
			assertCorrectedWireShape(t, message.Payload)
		})
	}
}

func assertCorrectedWireShape(t *testing.T, value any) {
	t.Helper()

	switch payload := value.(type) {
	case ws.TradesInstrumentNameIntervalSubscriptionMessagePayload:
		require.Len(t, payload.Params.Data, 1)
		assert.Equal(t, "example-trade", payload.Params.Data[0].TradeID)
	case ws.BlockTradeConfirmationsSubscriptionMessagePayload:
		require.Len(t, payload.Params.Data.Trades, 1)
		assert.InDelta(t, fixturePrice,
			payload.Params.Data.Trades[0].Price, 1e-10)
	case ws.BlockRfqTakerCurrencySubscriptionMessagePayload:
		assert.Nil(t, payload.Params.Data.ComboID)
		assert.Equal(t, []string{"example-maker"}, payload.Params.Data.Makers)
		assert.Empty(t, payload.Params.Data.Legs)
	case ws.UserTradesInstrumentNameIntervalSubscriptionMessagePayload:
		require.Len(t, payload.Params.Data, 1)
		trade := payload.Params.Data[0]
		assert.Nil(t, trade.MatchingID)
		require.NotNil(t, trade.PostOnly)
		assert.True(t, *trade.PostOnly)
		require.NotNil(t, trade.ReduceOnly)
		assert.False(t, *trade.ReduceOnly)
	default:
		t.Fatalf("unexpected corrected payload type %T", value)
	}
}

func TestDecodeAdditionalWireFields(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"testdata/sparse-ticker.json",
		"testdata/price-statistics.json",
		"testdata/portfolio.json",
		"testdata/user-changes.json",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()

			frame, err := os.ReadFile(tc)
			require.NoError(t, err)

			message, err := ws.DecodeReceivedMessage(frame)
			require.NoError(t, err)
			assertAdditionalWireFields(t, message.Payload)
		})
	}
}

func assertAdditionalWireFields(t *testing.T, value any) {
	t.Helper()

	switch payload := value.(type) {
	case ws.IncrementalTickerInstrumentNameSubscriptionMessagePayload:
		stats := payload.Params.Data.Stats
		require.NotNil(t, stats)
		assert.Nil(t, stats.Volume)
		assert.Nil(t, stats.High)
		assert.Nil(t, stats.Low)
		require.NotNil(t, stats.PriceChange)
		assert.InDelta(t, 1, *stats.PriceChange, 1e-10)
	case ws.DeribitPriceStatisticsIndexNameSubscriptionMessagePayload:
		require.NotNil(t, payload.Params.Data.HighVolatility)
		assert.True(t, *payload.Params.Data.HighVolatility)
	case ws.UserPortfolioCurrencySubscriptionMessagePayload:
		require.NotNil(t, payload.Params.Data.SpotReserve)
		assert.InDelta(t, 1, *payload.Params.Data.SpotReserve, 1e-10)
	case ws.UserChangesInstrumentNameIntervalSubscriptionMessagePayload:
		require.Len(t, payload.Params.Data.Positions, 1)
		require.Len(t, payload.Params.Data.Orders, 1)
		require.NotNil(t, payload.Params.Data.Orders[0].MaxShow)
		assert.InDelta(t, 1, *payload.Params.Data.Orders[0].MaxShow, 1e-10)
	default:
		t.Fatalf("unexpected extended payload type %T", value)
	}
}

func TestDocumentedOptionalFields(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"testdata/confirmation-no-app.json",
		"testdata/spot-position-no-settlement.json",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()

			frame, err := os.ReadFile(tc)
			require.NoError(t, err)

			message, err := ws.DecodeReceivedMessage(frame)
			require.NoError(t, err)

			switch payload := message.Payload.(type) {
			case ws.BlockTradeConfirmationsSubscriptionMessagePayload:
				assert.Nil(t, payload.Params.Data.AppName)
			case ws.UserChangesInstrumentNameIntervalSubscriptionMessagePayload:
				require.Len(t, payload.Params.Data.Positions, 1)
				assert.Nil(t, payload.Params.Data.Positions[0].SettlementPrice)
			default:
				t.Fatalf("unexpected optional payload type %T", message.Payload)
			}
		})
	}
}
