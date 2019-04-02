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
 * @Date:   2019-03-18T14:26:10+11:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:31:32+11:00
 */

package status

// VerificationStatus represents the verification status type
type VerificationStatus int

const (
	// VerificationStatusUnverifiable represents the unverifiable status of a proof verification
	VerificationStatusUnverifiable VerificationStatus = iota
	// VerificationStatusFalsified represents the falsified status of a proof verification
	VerificationStatusFalsified
	// VerificationStatusVerified represents the verified status of a proof verification
	VerificationStatusVerified
)

// VerificationStatusError combines an error with its `VerificationStatus`
type VerificationStatusError struct {
	Status VerificationStatus
	Err    error
}

// NewVerificationStatusError creates a new `VerificationStatusError`
func NewVerificationStatusError(status VerificationStatus, err error) *VerificationStatusError {
	return &VerificationStatusError{
		Status: status,
		Err:    err,
	}
}

func (v *VerificationStatusError) Error() string {
	return v.Err.Error()
}
