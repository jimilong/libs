package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/libs/grpc/interceptor/tracing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func HttpCall(method, url string, req, resp interface{}, ctx context.Context) error {
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		parentCtx := parentSpan.Context()
		// start a new Span to wrap HTTP request
		span := tracing.DefaultTracer.StartSpan(
			"HttpCall",
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.HTTPUrl), Value: url},
			opentracing.Tag{Key: string(ext.HTTPMethod), Value: method},
		)

		// make sure the Span is finished once we're done
		defer span.Finish()
	}

	cli := &http.Client{
		Timeout: time.Second * 20,
	}

	buf := bytes.NewBuffer(nil)
	if req != nil {
		if err := json.NewEncoder(buf).Encode(req); err != nil {
			return err
		}
	}
	request, err := http.NewRequest(method, url, buf)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json; charset=utf-8")
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	response, err := cli.Do(request)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode > 300 {
		return fmt.Errorf("Method: %s. Response Code: %d %s\nbody: %s", method, response.StatusCode, response.Status, body)
	}

	return json.Unmarshal(body, resp)
}
