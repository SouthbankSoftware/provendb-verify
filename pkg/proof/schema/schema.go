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
 * @Last modified time: 2019-04-02T13:31:19+11:00
 */

package schema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ChainpointProofSchemaV3 is the JSON schema used to verify a Proof JSON in v3
const ChainpointProofSchemaV3 = `{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "additionalProperties": false,
  "definitions": {
    "branch": {
      "additionalProperties": false,
      "properties": {
        "label": {
          "description": "An aritrary text branch label. Can contain up to 64 letters, numbers, hyphen, underscore, or period characters.",
          "pattern": "^[a-zA-Z0-9-_\\.]*$",
          "title": "The Label Schema",
          "type": "string",
          "minLength": 0,
          "maxLength": 64
        },
        "branches": {
          "items": {
            "$ref": "#/definitions/branch"
          },
          "type": "array",
          "uniqueItems": true
        },
        "ops": {
          "items": {
            "$ref": "#/definitions/operation"
          },
          "type": "array"
        }
      },
      "required": [
        "ops"
      ],
      "type": "object"
    },
    "anchor": {
      "additionalProperties": false,
      "properties": {
        "type": {
          "description": "A trust anchor",
          "title": "One of the known trust anchor types. Calendar (cal), Ethereum (eth), and Bitcoin (btc).",
          "type": "string",
          "enum": [
            "cal",
            "eth",
            "btc"
          ]
        },
        "anchor_id": {
          "description": "An identifier used to look up embedded anchor data. e.g. a Bitcoin transaction or block ID.",
          "title": "A service specific unique ID for this anchor",
          "type": "string",
          "minLength": 1,
          "maxLength": 512
        },
        "uris": {
          "items": {
            "description": "A URI used to lookup and retrieve the exact hash resource required to validate this anchor. The URI MUST return only a Hexadecimal hash value as a string. The URI MUST also contain the current 'anchor_id' value to lookup the URI resource. This strict requirement is to allow automated clients to retrieve and validate intermediate hashes when verifying a proof. The body value returned by the URI MUST be of even length and match the regex /^[a-fA-F0-9]+$/.",
            "title": "A URI for retrieving a hash value for this item",
            "type": "string",
            "format": "uri",
            "minLength": 1,
            "maxLength": 512
          },
          "type": "array",
          "uniqueItems": true
        }
      },
      "required": [
        "type",
        "anchor_id"
      ],
      "type": "object"
    },
    "operation": {
      "additionalProperties": false,
      "properties": {
        "l": {
          "description": "Concatenate the byte array value of this property to the left of the prior state of the hash (value|prior_hash).",
          "title": "Concatenate value with left side of previous value",
          "type": "string",
          "minLength": 1,
          "maxLength": 512
        },
        "r": {
          "description": "Concatenate the byte array value of this property to the right of the prior state of the hash (prior_hash|value).",
          "title": "Concatenate value with right side of previous value",
          "type": "string",
          "minLength": 1,
          "maxLength": 512
        },
        "op": {
          "description": "A hashing operation from the SHA2 or SHA3 families of hash functions to apply to a left or right operation hash value. The special value of 'sha-256-x2' performs a 'sha-256' twice in a row.",
          "title": "The hashing operation to apply to a left or right hash",
          "type": "string",
          "enum": [
            "sha-224",
            "sha-256",
            "sha-384",
            "sha-512",
            "sha3-224",
            "sha3-256",
            "sha3-384",
            "sha3-512",
            "sha-256-x2"
          ]
        },
        "anchors": {
          "items": {
            "$ref": "#/definitions/anchor"
          },
          "type": "array",
          "uniqueItems": true
        }
      },
      "type": "object"
    }
  },
  "description": "This document contains a schema for validating an instance of a Chainpoint v3 Proof.",
  "id": "http://example.com/example.json",
  "properties": {
    "@context": {
      "default": "https://w3id.org/chainpoint/v3",
      "description": "A registered JSON-LD context URI for this document type",
      "title": "The JSON-LD @context",
      "type": "string",
      "enum": [
        "https://w3id.org/chainpoint/v3"
      ]
    },
    "type": {
      "default": "Chainpoint",
      "description": "The JSON-LD Type",
      "title": "The JSON-LD Type",
      "type": "string",
      "enum": [
        "Chainpoint"
      ]
    },
    "hash": {
      "description": "The even length Hexadecimal output of a cryptographic one-way hash function representing the data to be anchored.",
      "pattern": "^[a-fA-F0-9]{40,128}$",
      "title": "The hash to be anchored",
      "type": "string"
    },
    "hash_id_node": {
      "description": "The Type 1 (timestamp) UUID used to identify and track a hash or retrieve a Chainpoint proof from a Chainpoint Node",
      "pattern": "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$",
      "title": "A Type 1 (timestamp) UUID that identifies a hash",
      "type": "string"
    },
    "hash_submitted_node_at": {
      "description": "The timestamp, in ISO8601 form, extracted from the hash_id_node that represents the time the hash was submitted to Chainpoint Node. Must be in \"2017-03-23T11:30:33Z\" form with granularity only to seconds and UTC zone.",
      "pattern": "^\\d{4}-\\d\\d-\\d\\dT\\d\\d:\\d\\d:\\d\\dZ$",
      "title": "An ISO8601 timestamp, extracted from hash_id_node",
      "type": "string"
    },
    "hash_id_core": {
      "description": "The Type 1 (timestamp) UUID used to by Chainpoint Node to identify and track a hash or retrieve a Chainpoint proof from Chainpoint Core",
      "pattern": "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$",
      "title": "A Type 1 (timestamp) UUID that identifies a hash",
      "type": "string"
    },
    "hash_submitted_core_at": {
      "description": "The timestamp, in ISO8601 form, extracted from the hash_id_core that represents the time the hash was submitted to Chainpoint Core. Must be in \"2017-03-23T11:30:33Z\" form with granularity only to seconds and UTC zone.",
      "pattern": "^\\d{4}-\\d\\d-\\d\\dT\\d\\d:\\d\\d:\\d\\dZ$",
      "title": "An ISO8601 timestamp, extracted from hash_id_core",
      "type": "string"
    },
    "branches": {
      "items": {
        "$ref": "#/definitions/branch"
      },
      "type": "array",
      "uniqueItems": true
    }
  },
  "required": [
    "@context",
    "type",
    "hash",
    "hash_id_node",
    "hash_submitted_node_at",
    "hash_id_core",
    "hash_submitted_core_at",
    "branches"
  ],
  "title": "Chainpoint v3 JSON Schema.",
  "type": "object"
}`

// Verify verifies a Proof JSON interface{}
func Verify(proof interface{}) (err error) {
	schemaLoader := gojsonschema.NewStringLoader(ChainpointProofSchemaV3)
	proofLoader := gojsonschema.NewGoLoader(proof)

	result, err := gojsonschema.Validate(schemaLoader, proofLoader)
	if err != nil {
		return err
	}

	if result.Valid() {
		return nil
	}

	var b strings.Builder

	fmt.Fprintf(&b, "failed to pass Chainpoint v3 JSON schema:\n")

	for _, desc := range result.Errors() {
		fmt.Fprintf(&b, "- %s\n", desc)
	}

	return errors.New(b.String())
}
