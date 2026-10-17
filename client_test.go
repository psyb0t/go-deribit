package deribit_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/psyb0t/go-deribit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAuthorization = "Bearer EXAMPLE-DO-NOT-USE"

func newTestClient(
	t *testing.T,
	handler http.HandlerFunc,
) *deribit.ClientWithResponses {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := deribit.NewClientWithResponses(
		server.URL+"/api/v2",
		deribit.WithRequestEditorFn(
			func(_ context.Context, request *http.Request) error {
				request.Header.Set("Authorization", testAuthorization)

				return nil
			},
		),
	)
	require.NoError(t, err)

	return client
}

func TestPublicGetTime(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/public/get_time", r.URL.Path)
		assert.Equal(t, testAuthorization, r.Header.Get("Authorization"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Empty(t, body)

		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"jsonrpc":"2.0","result":123456789}`)
		assert.NoError(t, err)
	})

	response, err := client.PublicGetTimeWithResponse(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())
	require.NotNil(t, response.JSON200)
	assert.Equal(t, 123456789, response.JSON200.Result)
}

func TestPrivateGetAccountSummary(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/private/get_account_summary", r.URL.Path)
		assert.Equal(t, "BTC", r.URL.Query().Get("currency"))
		assert.Equal(t, testAuthorization, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{
			"jsonrpc":"2.0",
			"result":{"currency":"BTC","balance":1.23456789}
		}`)
		assert.NoError(t, err)
	})

	response, err := client.PrivateGetAccountSummaryWithResponse(
		context.Background(),
		&deribit.PrivateGetAccountSummaryParams{Currency: "BTC"},
	)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	assert.Equal(t, "BTC", response.JSON200.Result.Currency)
	assert.InDelta(t, 1.23456789, response.JSON200.Result.Balance, 0)
}

func TestUnauthorizedResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	response, err := client.PrivateGetAccountSummaryWithResponse(
		context.Background(),
		&deribit.PrivateGetAccountSummaryParams{Currency: "BTC"},
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
	assert.Nil(t, response.JSON200)
}

func TestHistoricalVolatilityPairs(t *testing.T) {
	t.Parallel()

	var response deribit.PublicGetHistoricalVolatilityResponse

	err := json.Unmarshal([]byte(`{
		"jsonrpc":"2.0","result":[[123456789,12.3456789]]
	}`), &response)
	require.NoError(t, err)
	require.Len(t, response.Result, 1)
	assert.Equal(t, []float64{123456789, 12.3456789}, response.Result[0])
}

func TestInstrumentTickSizeSteps(t *testing.T) {
	t.Parallel()

	var response deribit.PublicGetInstrumentResponse

	err := json.Unmarshal([]byte(`{
		"jsonrpc":"2.0",
		"result":{"tick_size_steps":[{"above_price":100,"tick_size":0.001}]}
	}`), &response)
	require.NoError(t, err)
	require.NotNil(t, response.Result.TickSizeSteps)
	require.Len(t, *response.Result.TickSizeSteps, 1)
	step := (*response.Result.TickSizeSteps)[0]
	require.NotNil(t, step.TickSize)
	assert.InDelta(t, 0.001, *step.TickSize, 0)
}

func TestPositionMoveArray(t *testing.T) {
	t.Parallel()

	var response deribit.PrivatePositionMoveResponse

	err := json.Unmarshal([]byte(`{
		"jsonrpc":"2.0","result":[{"trades":[]}]
	}`), &response)
	require.NoError(t, err)
	require.Len(t, response.Result, 1)
	assert.Empty(t, response.Result[0].Trades)
}

func TestNullableTransactionCommission(t *testing.T) {
	t.Parallel()

	var entry deribit.TransactionLog

	require.NoError(t, json.Unmarshal([]byte(`{"commission":null}`), &entry))
	assert.Nil(t, entry.Commission)
	require.NoError(t, json.Unmarshal([]byte(`{"commission":0}`), &entry))
	require.NotNil(t, entry.Commission)
	assert.Zero(t, *entry.Commission)
}
