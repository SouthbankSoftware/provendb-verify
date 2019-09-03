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
 * @Date:   2018-08-28T11:26:28+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-08-28T16:33:29+10:00
 */

package binary

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/testutil"
)

func TestBase642Proof(t *testing.T) {
	p2 := testutil.LoadFile(t, "proof2_base64.txt")
	defer p2.Close()

	p3 := testutil.LoadFile(t, "proof3_base64.txt")
	defer p3.Close()

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name      string
		args      args
		wantProof interface{}
		wantErr   bool
	}{
		{
			"Decode base64 string - proof2_base64.txt",
			args{
				p2,
			},
			testutil.LoadJSON(t, "proof2.json"),
			false,
		},
		{
			"Decode base64 string - proof3_base64.txt",
			args{
				p3,
			},
			testutil.LoadJSON(t, "proof3.json"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotProof interface{}
			err := Base642Proof(tt.args.r, &gotProof)

			if (err != nil) != tt.wantErr {
				t.Errorf("Base642Proof() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotProof, tt.wantProof) {
				t.Errorf("Base642Proof() = %v, want %v", gotProof, tt.wantProof)
			}
		})
	}
}

func isEqualBase64(t *testing.T, a, b string) bool {
	bufA := bytes.NewBufferString(a)
	bufB := bytes.NewBufferString(b)

	var proofA, proofB interface{}

	err := Base642Proof(bufA, &proofA)
	if err != nil {
		t.Fatal(err)
	}

	err = Base642Proof(bufB, &proofB)
	if err != nil {
		t.Fatal(err)
	}

	return reflect.DeepEqual(proofA, proofB)
}

func TestProof2Base64(t *testing.T) {
	type args struct {
		proof interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"Encode Chainpoint Proof - proof2.json",
			args{
				testutil.LoadJSON(t, "proof2.json"),
			},
			testutil.LoadString(t, "proof2_base64.txt"),
			false,
		},
		{
			"Encode Chainpoint Proof - proof3.json",
			args{
				testutil.LoadJSON(t, "proof3.json"),
			},
			testutil.LoadString(t, "proof3_base64.txt"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := Proof2Base64(tt.args.proof, w); (err != nil) != tt.wantErr {
				t.Errorf("Proof2Base64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); !isEqualBase64(t, gotW, tt.wantW) {
				t.Errorf("Proof2Base64() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
