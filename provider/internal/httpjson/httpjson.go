package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Request struct {
	Client      *http.Client
	URL         string
	Headers     map[string]string
	ErrorPrefix string
}

func Post[Req any, Resp any](ctx context.Context, cfg Request, req Req) (*Resp, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := post(ctx, cfg, body)
	if err != nil {
		return nil, err
	}

	var resp Resp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

func post(ctx context.Context, cfg Request, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	setHeaders(httpReq, cfg.Headers)

	httpResp, err := cfg.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s (%d): %s", cfg.ErrorPrefix, httpResp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func setHeaders(req *http.Request, headers map[string]string) {
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
}
