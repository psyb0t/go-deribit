package deribit_test

import (
	"context"
	"net/http"
	"os"

	"github.com/psyb0t/go-deribit"
)

//nolint:testableexamples // Calls the external Deribit API.
func Example() {
	auth := deribit.WithRequestEditorFn(
		func(_ context.Context, req *http.Request) error {
			req.Header.Set(
				"Authorization",
				"Bearer "+os.Getenv("DERIBIT_ACCESS_TOKEN"),
			)

			return nil
		},
	)

	client, err := deribit.NewClientWithResponses(
		"https://test.deribit.com/api/v2",
		auth,
	)
	if err != nil {
		panic(err)
	}

	res, err := client.PublicGetTimeWithResponse(context.Background())
	if err != nil {
		panic(err)
	}

	_ = res.JSON200
}
