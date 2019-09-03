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
 * @Date:   2018-08-17T10:48:15+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-08-28T16:34:15+10:00
 */

package proof

import (
	"context"
	"fmt"
	"io"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/anchor"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/binary"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/eval"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/schema"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/status"
)

// Verify verifies a given Chainpoint Proof in either base64 binary string or JSON interface{}
func Verify(ctx context.Context, rawProof interface{}) (st status.VerificationStatus, er error) {
	var proof interface{}

	switch p := rawProof.(type) {
	case io.Reader:
		var pf interface{}
		err := binary.Base642Proof(p, &pf)
		if err != nil {
			er = err
			return
		}
		proof = pf
	case map[string]interface{}:
		proof = p
	default:
		er = fmt.Errorf("unsupported Chainpoint Proof format %T", p)
		return
	}

	err := schema.Verify(proof)
	if err != nil {
		st = status.VerificationStatusFalsified
		er = err
		return
	}

	evaluatedProof, err := eval.Eval(proof)
	if err != nil {
		st = status.VerificationStatusFalsified
		er = err
		return
	}

	err = anchor.Verify(ctx, evaluatedProof)
	if err != nil {
		if se, ok := err.(*status.VerificationStatusError); ok {
			st = se.Status
			er = se.Err
			return
		}

		er = err
		return
	}

	return status.VerificationStatusVerified, nil
}
