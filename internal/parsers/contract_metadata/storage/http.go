package storage

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// HTTP Storage prefixes
const (
	PrefixHTTP  = "http"
	PrefixHTTPS = "https"
)

// HTTPStorage -
type HTTPStorage struct {
	timeout time.Duration

	client *http.Client
}

// HTTPStorageOption -
type HTTPStorageOption func(*HTTPStorage)

// WithTimeoutHTTP -
func WithTimeoutHTTP(timeout time.Duration) HTTPStorageOption {
	return func(s *HTTPStorage) {
		if timeout != 0 {
			s.timeout = timeout
		}
	}
}

// NewHTTPStorage -
func NewHTTPStorage(opts ...HTTPStorageOption) HTTPStorage {
	s := HTTPStorage{
		timeout: defaultTimeout,
	}

	for i := range opts {
		opts[i](&s)
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	s.client = &http.Client{
		Timeout:   s.timeout,
		Transport: t,
	}

	return s
}

// Get -
func (s HTTPStorage) Get(value string, output interface{}) error {
	req, err := http.NewRequest("GET", value, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Wrap(ErrHTTPRequest, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Invalid status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return errors.Wrap(ErrJSONDecoding, err.Error())
	}

	return nil
}
