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
 * @Date:   2018-08-22T13:22:09+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2020-05-19T15:06:37+10:00
 */

package eval

import (
	"reflect"
	"testing"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/testutil"
	log "github.com/sirupsen/logrus"
)

func TestEval(t *testing.T) {
	type args struct {
		proof interface{}
	}
	tests := []struct {
		name       string
		args       args
		wantResult interface{}
		wantErr    bool
	}{
		{
			"Evaluate Chainpoint v3 Proof - proof1.json",
			args{
				testutil.LoadJSON(t, "proof1.json"),
			},
			testutil.LoadJSON(t, "evaluated_proof1.json"),
			false,
		},
		{
			"Evaluate Chainpoint v3 Proof - proof2.json",
			args{
				testutil.LoadJSON(t, "proof2.json"),
			},
			testutil.LoadJSON(t, "evaluated_proof2.json"),
			false,
		},
		{
			"Evaluate Chainpoint v3 Proof - proof3.json",
			args{
				testutil.LoadJSON(t, "proof3.json"),
			},
			testutil.LoadJSON(t, "evaluated_proof3.json"),
			false,
		},
		{
			"Evaluate Chainpoint v3 Proof - proof4.json",
			args{
				testutil.LoadJSON(t, "proof4.json"),
			},
			testutil.LoadJSON(t, "evaluated_proof4.json"),
			false,
		},
		{
			"Evaluate Chainpoint v3 Proof - proof5.json",
			args{
				testutil.LoadJSON(t, "proof5.json"),
			},
			testutil.LoadJSON(t, "evaluated_proof5.json"),
			false,
		},
		{
			"Evaluate corrupted JSON",
			args{
				"I am not JSON",
			},
			// nil map
			*new(map[string]interface{}),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := Eval(tt.args.proof)
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("Eval() = %v, want %v", gotResult, tt.wantResult)
			}

			if err != nil {
				log.Error(err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
