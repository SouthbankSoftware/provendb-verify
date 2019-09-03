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
 * @Last modified time: 2019-08-28T16:16:24+10:00
 */

package binary

import (
	"compress/zlib"
	"encoding/base64"
	"io"

	"github.com/vmihailenco/msgpack"
)

// Binary2Proof reads a binary stream into a Chainpoint Proof
func Binary2Proof(r io.Reader, v interface{}) error {
	msgpackR, err := zlib.NewReader(r)
	if err != nil {
		return err
	}

	defer msgpackR.Close()

	return msgpack.NewDecoder(msgpackR).UseJSONTag(true).Decode(v)
}

// Base642Proof reads a base64 binary stream into a Chainpoint Proof
func Base642Proof(r io.Reader, v interface{}) error {
	zlibR := base64.NewDecoder(base64.StdEncoding, r)
	return Binary2Proof(zlibR, v)
}

// Proof2Binary writes a Chainpoint Proof into a binary stream
func Proof2Binary(proof interface{}, w io.Writer) (err error) {
	msgpackW := zlib.NewWriter(w)

	defer msgpackW.Close()

	err = msgpack.NewEncoder(msgpackW).UseJSONTag(true).Encode(&proof)
	if err != nil {
		return err
	}

	return nil
}

// Proof2Base64 writes a Chainpoint Proof into a base64 binary stream
func Proof2Base64(proof interface{}, w io.Writer) (err error) {
	zlibW := base64.NewEncoder(base64.StdEncoding, w)

	defer zlibW.Close()

	return Proof2Binary(proof, zlibW)
}
