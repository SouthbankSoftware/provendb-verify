/*
 * @Author: guiguan
 * @Date:   2020-06-16T15:14:48+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2020-06-16T19:26:48+10:00
 */

package httputil

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	"golang.org/x/net/context/ctxhttp"
)

const (
	maxNumRetry = 10
)

var (
	// ErrHTTP404NotFound is the HTTP error: 404 Not Found
	ErrHTTP404NotFound = errors.New("404 Not Found")
)

// HTTPGet gets the URL
func HTTPGet(ctx context.Context, url string) (rc io.ReadCloser, er error) {
	var (
		retryCount = 0
		resp       *http.Response
	)

	for {
		res, err := ctxhttp.Get(ctx, &http.Client{}, url)
		if err != nil {
			er = err
			return
		}
		resp = res

		if sc := resp.StatusCode; sc == 429 {
			// 429: Too Many Requests

			// randomly wait 100 to 1000 ms before retrying
			nBig, err := rand.Int(rand.Reader, big.NewInt(900))
			if err != nil {
				er = err
				return
			}
			n := nBig.Int64()

			time.Sleep(time.Duration(n) * time.Millisecond)

			retryCount++
			if retryCount <= maxNumRetry {
				continue
			} else {
				er = fmt.Errorf("still getting %s from %s after %d retries", resp.Status, url, maxNumRetry)
				return
			}
		} else if sc >= 400 {
			var errMsg string

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = string(bodyBytes)
			}

			if sc == 404 {
				er = fmt.Errorf("got %w from %s: %s", ErrHTTP404NotFound, url, errMsg)
				return
			}

			er = fmt.Errorf("got %s from %s: %s", resp.Status, url, errMsg)
			return
		}

		break
	}

	rc = resp.Body
	return
}

// UnmarshalHTTPGetJSON unmarshals the URL result as a JSON
func UnmarshalHTTPGetJSON(ctx context.Context, url string, v interface{}) error {
	body, err := HTTPGet(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	err = json.NewDecoder(body).Decode(v)
	if err != nil {
		return err
	}

	return nil
}

// HTTPGetJSON gets the URL result as a JSON object
func HTTPGetJSON(ctx context.Context, url string) (obj interface{}, er error) {
	er = UnmarshalHTTPGetJSON(ctx, url, &obj)
	return
}
