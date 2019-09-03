/*
 * provendb-verify
 * Copyright (C) 2019  Southbank Software Ltd.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 * @Author: guiguan
 * @Date:   2018-08-24T09:56:10+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-09-02T17:36:03+10:00
 */

package anchor

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
	"os"
	"time"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/status"
	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/sync/errgroup"
)

var (
	bcToken = ""
)

const (
	maxNumRetry = 10
)

func init() {
	if token, ok := os.LookupEnv("PROVENDB_VERIFY_BCTOKEN"); ok {
		bcToken = token
	}
}

// ShowProgress is the flag indicates whether to show anchor verification progress as log messages
var ShowProgress = true

// Verify verifies anchor info in a given evaluated Proof JSON and returns nil if succeed, and error
// otherwise. When the proof is verifiable and falsified, the returned error is a type of `VerificationStatusError`.
func Verify(ctx context.Context, evaluatedProof interface{}) (er error) {
	defer func() {
		if r := recover(); r != nil {
			// type assertion panics are treated as `VerificationStatusFalsified`
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}

		if er != nil {
			// add error prefix
			err := fmt.Errorf("failed to verify Proof anchors: %s", er)

			if se, ok := er.(*status.VerificationStatusError); ok {
				se.Err = err
			} else {
				er = err
			}
		}
	}()

	return verifyBranches(ctx,
		evaluatedProof.(map[string]interface{})["branches"].([]interface{}))
}

func verifyBranches(ctx context.Context, branches []interface{}) (er error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	eg, egCtx := errgroup.WithContext(ctx)

	for _, branch := range branches {
		branch := branch.(map[string]interface{})

		switch l := branch["label"]; l {
		case "eth_anchor_branch":
			eg.Go(func() error {
				return verifyEthereumBranch(egCtx, branch)
			})
		case "btc_anchor_branch":
			eg.Go(func() error {
				return verifyBitcoinBranch(egCtx, branch)
			})
		case "pdb_doc_branch":
		case "cal_anchor_branch":
			eg.Go(func() error {
				return verifyCalendarBranch(egCtx, branch)
			})
		default:
			return status.NewVerificationStatusError(status.VerificationStatusFalsified, fmt.Errorf("unsupported branch type: %s", l))
		}

		if bsI := branch["branches"]; bsI != nil {
			bs := bsI.([]interface{})

			eg.Go(func() error {
				return verifyBranches(egCtx, bs)
			})
		}
	}

	return eg.Wait()
}

func verifyCalendarBranch(ctx context.Context, branch map[string]interface{}) (er error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	if ShowProgress {
		fmt.Println("Verifying Chainpoint anchor...")
	}

	anchors := branch["anchors"].([]interface{})

	eg, egCtx := errgroup.WithContext(ctx)

	for _, anchor := range anchors {
		anchor := anchor.(map[string]interface{})
		uris := anchor["uris"].([]interface{})
		expectedValue := anchor["expected_value"].(string)

		eg.Go(func() (er error) {
			return verifyAnchorURIs(egCtx, uris, expectedValue)
		})
	}

	return eg.Wait()
}

func verifyEthereumBranch(ctx context.Context, branch map[string]interface{}) (er error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	if ShowProgress {
		fmt.Println("Verifying Ethereum anchor...")
	}

	anchors := branch["anchors"].([]interface{})

	eg, egCtx := errgroup.WithContext(ctx)

	for _, anchor := range anchors {
		anchor := anchor.(map[string]interface{})
		uris := anchor["uris"].([]interface{})
		expectedValue := anchor["expected_value"].(string)

		eg.Go(func() (er error) {
			return verifyAnchorURIs(egCtx, uris, expectedValue)
		})
	}

	return eg.Wait()
}

func verifyBitcoinBranch(ctx context.Context, branch map[string]interface{}) (er error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	if ShowProgress {
		fmt.Println("Verifying Bitcoin anchor...")
	}

	anchors := branch["anchors"].([]interface{})

	eg, egCtx := errgroup.WithContext(ctx)

	for _, anchor := range anchors {
		anchor := anchor.(map[string]interface{})

		blockHeight := anchor["anchor_id"].(string)
		uris := anchor["uris"].([]interface{})
		expectedValue := anchor["expected_value"].(string)

		eg.Go(func() error {
			return verifyAnchorURIs(egCtx, uris, expectedValue)
		})

		eg.Go(func() error {
			return verifyBitcoinBlockMerkleRoot(egCtx, blockHeight, expectedValue)
		})
	}

	txID := branch["btcTxId"].(string)
	expectedValue := branch["opReturnValue"].(string)

	eg.Go(func() error {
		return verifyBitcoinTxOpReturn(egCtx, txID, expectedValue)
	})

	return eg.Wait()
}

func verifyBitcoinBlockMerkleRoot(ctx context.Context, blockHeight string, expectedValue string) (er error) {
	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	if ShowProgress {
		fmt.Println("Verifying Bitcoin block merkle root...")
	}

	json, err := httpGetJSON(ctx, fmt.Sprintf("https://api.blockcypher.com/v1/btc/main/blocks/%s?txstart=1&limit=1&token=%s", blockHeight, bcToken))
	if err != nil {
		return err
	}

	jsonM := json.(map[string]interface{})

	if errStr := jsonM["error"]; errStr != nil {
		return errors.New(errStr.(string))
	}

	actualValue := jsonM["mrkl_root"]

	if actualValue != expectedValue {
		return status.NewVerificationStatusError(
			status.VerificationStatusFalsified,
			fmt.Errorf("Bitcoin block height %s has merkle root %s, but expect %s", blockHeight, actualValue, expectedValue),
		)
	}

	if ShowProgress {
		fmt.Printf("Bitcoin block height %s has merkle root %s\n", blockHeight, actualValue)
	}

	return nil
}

func verifyBitcoinTxOpReturn(ctx context.Context, txID string, expectedValue string) (er error) {
	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	if ShowProgress {
		fmt.Println("Verifying Bitcoin transaction OP_RETURN...")
	}

	json, err := httpGetJSON(ctx, fmt.Sprintf("https://api.blockcypher.com/v1/btc/main/txs/%s?token=%s", txID, bcToken))
	if err != nil {
		return err
	}

	jsonM := json.(map[string]interface{})

	if errStr := jsonM["error"]; errStr != nil {
		return errors.New(errStr.(string))
	}

	actualValue := jsonM["outputs"].([]interface{})[0].(map[string]interface{})["data_hex"]

	if actualValue != expectedValue {
		return status.NewVerificationStatusError(
			status.VerificationStatusFalsified,
			fmt.Errorf("Bitcoin transaction %s has OP_RETURN %s, but expect %s", txID, actualValue, expectedValue),
		)
	}

	if ShowProgress {
		fmt.Printf("Bitcoin transaction %s has OP_RETURN %s\n", txID, actualValue)
	}

	return nil
}

func verifyAnchorURIs(ctx context.Context, uris []interface{}, expectedValue string) (er error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	defer func() {
		if r := recover(); r != nil {
			er = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}
	}()

	eg, egCtx := errgroup.WithContext(ctx)

	for _, uri := range uris {
		uri := uri.(string)

		eg.Go(func() error {
			body, err := httpGet(egCtx, uri)
			if err != nil {
				return err
			}
			defer body.Close()

			bodyBytes, err := ioutil.ReadAll(body)
			if err != nil {
				return err
			}

			actualValue := string(bodyBytes)

			if actualValue != expectedValue {
				return status.NewVerificationStatusError(
					status.VerificationStatusFalsified,
					fmt.Errorf("anchor URI %s returns %s, but expect %s", uri, actualValue, expectedValue),
				)
			}

			return nil
		})
	}

	return eg.Wait()
}

func httpGet(ctx context.Context, url string) (body io.ReadCloser, err error) {
	var (
		retryCount = 0
		resp       *http.Response
	)

	for {
		resp, err = ctxhttp.Get(ctx, &http.Client{}, url)
		if err != nil {
			return nil, err
		}

		if sc := resp.StatusCode; sc == 429 {
			// 429: Too Many Requests

			// randomly wait 100 to 1000 ms before retrying
			nBig, err := rand.Int(rand.Reader, big.NewInt(900))
			if err != nil {
				return nil, err
			}
			n := nBig.Int64()

			time.Sleep(time.Duration(n) * time.Millisecond)

			retryCount++
			if retryCount <= maxNumRetry {
				continue
			} else {
				return nil, fmt.Errorf("still getting %s from %s after %d retries", resp.Status, url, maxNumRetry)
			}
		} else if sc >= 400 {
			return nil, fmt.Errorf("got %s from %s", resp.Status, url)
		}

		break
	}

	return resp.Body, nil
}

func httpGetJSON(ctx context.Context, url string) (result interface{}, err error) {
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}

	defer body.Close()

	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
