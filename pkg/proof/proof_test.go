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
 * @Last modified time: 2019-04-02T13:32:02+11:00
 */

package proof

import (
	"context"
	"reflect"
	"testing"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/status"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/testutil"
)

func TestVerify(t *testing.T) {
	p2Falsified := testutil.LoadFile(t, "falsified_proof2_base64.txt")
	defer p2Falsified.Close()

	p3 := testutil.LoadFile(t, "proof3_base64.txt")
	defer p3.Close()

	p3Falsified := testutil.LoadFile(t, "falsified_proof3_base64.txt")
	defer p3Falsified.Close()

	type args struct {
		ctx      context.Context
		rawProof interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantSt  status.VerificationStatus
		wantErr bool
	}{
		{
			"Verify Chainpoint Proof (base64) - falsified_proof2_base64.txt",
			args{
				context.Background(),
				p2Falsified,
			},
			status.VerificationStatusFalsified,
			true,
		},
		{
			"Verify Chainpoint Proof (base64) - unverifiable_proof2.json",
			args{
				context.Background(),
				testutil.LoadJSON(t, "unverifiable_proof2.json"),
			},
			status.VerificationStatusUnverifiable,
			true,
		},
		{
			"Verify Chainpoint Proof (base64) - proof3_base64.txt",
			args{
				context.Background(),
				p3,
			},
			status.VerificationStatusVerified,
			false,
		},
		{
			"Verify Chainpoint Proof (inteface{}) - proof3.json",
			args{
				context.Background(),
				testutil.LoadJSON(t, "proof3.json"),
			},
			status.VerificationStatusVerified,
			false,
		},
		{
			"Verify Chainpoint Proof (base64) - falsified_proof3_base64.txt",
			args{
				context.Background(),
				p3Falsified,
			},
			status.VerificationStatusFalsified,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSt, err := Verify(tt.args.ctx, tt.args.rawProof)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSt, tt.wantSt) {
				t.Errorf("Verify() = %v, want %v", gotSt, tt.wantSt)
			}
		})
	}
}
