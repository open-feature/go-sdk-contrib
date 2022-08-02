package http_service_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	models "github.com/open-feature/flagd/pkg/model"
	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
	mocks "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http/mocks"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	schemav1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type TestFetchFlagArgs struct {
	name string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	body interface{}
	url  string
	ctx  of.EvaluationContext
	err  error
}

func TestFetchFlag(t *testing.T) {
	tests := []TestFetchFlagArgs{
		{
			name: "happy path",
			body: map[string]interface{}{
				"food": "bars",
			},
			url: "GET/MY/FLAG",
			ctx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			mockHttpResponseCode: 200,
			err:                  nil,
		},
		{
			name: "200 response cannot unmarshal",
			body: "string",
			url:  "GET/MY/FLAG",
			ctx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			mockHttpResponseCode: 200,
			err:                  errors.New(models.ParseErrorCode),
		},
		{
			name: "non 200 response cannot unmarshal",
			body: "string",
			url:  "GET/MY/FLAG",
			ctx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			mockHttpResponseCode: 400,
			err:                  errors.New(models.ParseErrorCode),
		},
		{
			name: "non 200 response",
			body: schemav1.ErrorResponse{
				ErrorCode: models.FlagNotFoundErrorCode,
			},
			url: "GET/MY/FLAG",
			ctx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			mockHttpResponseCode: 404,
			err:                  errors.New(models.FlagNotFoundErrorCode),
		},
		{
			name: "500 response",
			url:  "GET/MY/FLAG",
			ctx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			mockHttpResponseCode: 500,
			err:                  errors.New(models.GeneralErrorCode),
		},
		{
			name: "fall through",
			body: schemav1.ErrorResponse{
				ErrorCode: "",
			},
			url:                  "GET/MY/FLAG",
			mockHttpResponseCode: 400,
			err:                  errors.New(models.GeneralErrorCode),
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := mocks.NewMockiHTTPClient(ctrl)
		bodyM, err := json.Marshal(test.body)
		if err != nil {
			t.Error(err)
		}
		test.mockHttpResponseBody = &http.Response{
			StatusCode: test.mockHttpResponseCode,
			Body:       io.NopCloser(bytes.NewReader(bodyM)),
		}
		mock.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(
			&http.Response{
				StatusCode: test.mockHttpResponseCode,
				Body:       io.NopCloser(bytes.NewReader(bodyM)),
			},
			test.mockErr,
		)
		svc := service.HTTPService{
			Client: mock,
		}
		target := map[string]interface{}{}
		err = svc.FetchFlag(test.url, test.ctx, &target)

		if test.err != nil && !assert.EqualError(t, err, test.err.Error()) {
			t.Errorf("%s: unexpected value for error expected %v recieved %v", test.name, test.err, err)
		}
	}
}
