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
 * @Date:   2018-08-22T10:34:36+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:31:08+11:00
 */

package schema

import (
	"testing"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/testutil"
	log "github.com/sirupsen/logrus"
)

func TestVerify(t *testing.T) {
	type args struct {
		proof interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Valid Chainpoint v3 Proof",
			args{
				testutil.LoadJSON(t, "proof1.json"),
			},
			false,
		},
		{
			"Falsified Chainpoint v3 Proof",
			args{
				testutil.LoadJSON(t, "falsified_proof1.json"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Verify(tt.args.proof)

			if err != nil {
				log.Error(err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
