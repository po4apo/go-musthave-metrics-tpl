package agent

import (
	"errors"
	"net"
	"net/http"
	"syscall"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/retry"
)

type RetryClient struct {
	http.Client
}

func NewRetryClient() *RetryClient {
	return &RetryClient{}
}

func isConnError(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED)
}

func (c *RetryClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response

	err := retry.Do(func() error {
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return err
			}
			req.Body = body
		}

		var err error
		resp, err = c.Client.Do(req)
		return err
	}, isConnError)

	return resp, err
}
