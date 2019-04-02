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
 * @Date:   2018-08-01T14:59:49+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:24:53+11:00
 */

package sha256

import (
	"crypto/sha256"
)

// EmptyString returns sha256 of an empty string
var EmptyString = []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55}

// HashByteArray hashes given byte array into 32 bytes sha256 value
func HashByteArray(bs ...[]byte) []byte {
	hasher := sha256.New()

	for _, b := range bs {
		hasher.Write(b)
	}

	return hasher.Sum(nil)
}

// HashString hashes given string into 32 bytes sha256 value
func HashString(ss ...string) []byte {
	bs := make([][]byte, len(ss))

	for i, s := range ss {
		bs[i] = []byte(s)
	}

	return HashByteArray(bs...)
}
