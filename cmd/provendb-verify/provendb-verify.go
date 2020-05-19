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
 * @Date:   2018-08-01T13:23:16+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2020-05-19T17:31:02+10:00
 */

package main

import (
	"os"

	cli "gopkg.in/urfave/cli.v2"
)

var (
	cmdVersion = "0.0.0"
)

const (
	cmdName                       = "provendb-verify"
	versionIDCurrent              = "current"
	defaultMongoDBPort            = "27017"
	defaultMongoDBURI             = "mongodb://localhost:" + defaultMongoDBPort
	defaultErrorHelpMsg           = "try '" + cmdName + " -h' for more information"
	defaultMaxPoolSize            = uint16(30)
	docFilterFormatHelpMsg        = `MongoDB extended JSON format, such as, '{"_id": {"$oid": "5b6a6a1646e0fb00080aac8c"}}'`
	idKey                         = "_id"
	mongoDBSystemPrefix           = "system."
	provenDBMetaPrefix            = "_provendb"
	provenDBIgnoredSuffix         = "pdbignore"
	provenDBVersionProofs         = provenDBMetaPrefix + "_versionProofs"
	provenDBDocMetaKey            = provenDBMetaPrefix + "_metadata"
	provenDBDocMetaIDKey          = provenDBDocMetaKey + "." + idKey
	provenDBMinVersionKey         = "minVersion"
	provenDBForgottenKey          = "forgotten"
	provenDBDocMetaMinVersionKey  = provenDBDocMetaKey + "." + provenDBMinVersionKey
	provenDBDocMetaMaxVersionKey  = provenDBDocMetaKey + ".maxVersion"
	provenDBVersionKey            = "version"
	provenDBVersionIDKey          = provenDBVersionKey + "Id"
	provenDBVersionCurrent        = "current"
	provenDBProofIDKey            = "proofId"
	provenDBSubmittedKey          = "submitted"
	provenDBStatusKey             = "status"
	provenDBScopeKey              = "scope"
	provenDBFilterKey             = "filter"
	provenDBScopeCollection       = "collection"
	provenDBScopeDatabase         = "database"
	provenDBDetailsKey            = "details"
	provenDBCollectionsKey        = "collections"
	provenDBDetailsCollectionsKey = provenDBDetailsKey + "." + provenDBCollectionsKey
	provenDBNameKey               = "name"
	provenDBProofKey              = "proof"
	provenDBHashKey               = "hash"
	provenDBDocBranch             = "pdb_doc_branch"
)

type proofType string

var (
	debug,
	skipDocCheck,
	verifyAnchorIndependently bool
	proofTypes = struct {
		database proofType
		document proofType
		raw      proofType
	}{
		database: "database",
		document: "document",
		raw:      "raw",
	}
)

func main() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`

	app := &cli.App{
		Name:      "provendb-verify",
		Version:   cmdVersion,
		Usage:     "ProvenDB Open Source Verification CLI",
		ArgsUsage: " ",
		HideHelp:  true,
		Flags: []cli.Flag{
			// TODO: support dumpfile
			// &cli.StringFlag{
			// 	Name:  "file",
			// 	Usage: wrap("specify a mongodump `File` as the verification target. In this case, other MongoDB authentication options are ignored"),
			// },
			&cli.StringFlag{
				Name:  "uri",
				Usage: wrap("specify a resolvable MongoDB `URI` connection string as the verification target. When using this option with others, such as '--ssl=false', other explicitly specified options will always take precedence"),
				Value: defaultMongoDBURI,
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: wrap("specify a MongoDB `HOST` as the verification target"),
			},
			&cli.StringFlag{
				Name:  "port",
				Usage: wrap("specify a MongoDB `PORT` as the verification target"),
			},
			&cli.BoolFlag{
				Name:        "ssl",
				Usage:       wrap("use SSL for the MongoDB connection"),
				DefaultText: "",
			},
			&cli.StringFlag{
				Name:    "username",
				Aliases: []string{"u"},
				Usage:   wrap("specify a `USERNAME` for the MongoDB authentication"),
			},
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   wrap("specify a `PASSWORD` for the MongoDB authentication"),
			},
			&cli.StringSliceFlag{
				Name:        "ignoredCollections",
				Usage:       wrap("specify a comma seperated list of ignored collections"),
				DefaultText: "",
			},
			&cli.StringFlag{
				Name:    "authDatabase",
				Aliases: []string{"adb"},
				Usage:   wrap("specify a `DATABASE` to be used for authentication (ignored for a ProvenDB connection)"),
			},
			&cli.StringFlag{
				Name:    "database",
				Aliases: []string{"db"},
				Usage:   wrap("specify a `DATABASE` as the verification target (ignored for a ProvenDB connection)"),
			},
			&cli.StringFlag{
				Name:    provenDBProofIDKey,
				Aliases: []string{"pid"},
				Usage:   wrap("specify a ProvenDB Proof `ID` and use the version in that Proof as '--versionId'"),
			},
			&cli.StringFlag{
				Name:    provenDBVersionIDKey,
				Aliases: []string{"vid"},
				Usage:   wrap("specify a `VERSION` to be verified. Use '" + versionIDCurrent + "' to verify the most recent version"),
				Value:   versionIDCurrent,
			},
			&cli.StringFlag{
				Name:    "in",
				Aliases: []string{"i"},
				Usage:   wrap("specify a `PATH` to a ProvenDB Proof Archive (.zip) or an external Chainpoint Proof either in base64 (.txt) or JSON (.json). The (.txt) or (.json) will be used to verify the database or document, instead of using the stored one in ProvenDB. If the database or document is not specified, the (.txt) or (.json) itself will only be verified. You can use '--out' to output such (.txt) or (.json)"),
			},
			&cli.StringFlag{
				Name:  "pubKey",
				Usage: wrap("specify a `PATH` to a RSA public key (.pem) to verify the signature contained in a Proof"),
			},
			&cli.BoolFlag{
				Name:    "listVersions",
				Aliases: []string{"ls"},
				Usage:   wrap("list all the verifiable versions along with ProvenDB Proof IDs for the target MongoDB database"),
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"col"},
				Usage:   wrap("specify the collection `NAME` of the document to be verified. When using this option, '--docFilter' must be provided to get that document"),
			},
			&cli.StringFlag{
				Name:    "docFilter",
				Aliases: []string{"df"},
				Usage:   wrap("specify a `FILTER` to get the document as the verification target, which must be in " + docFilterFormatHelpMsg + ". When using this option, '--collection' must be provided. Ignoring this, the provided MongoDB database will be verified. This option can be combined with '--versionId' to verify a document in that specific version"),
			},
			&cli.StringFlag{
				Name:    "out",
				Aliases: []string{"o"},
				Usage:   wrap("specify a `PATH` to output the Chainpoint Proof when verified. Then filename in the PATH must end with either '.json' (for JSON) or '.txt' (for compressed binary in base64)"),
			},
			&cli.BoolFlag{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   wrap("show this usage information"),
			},
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       wrap("print out debug information"),
				Destination: &debug,
			},
			&cli.BoolFlag{
				Name:        "skipDocCheck",
				Usage:       wrap("skip checking document hash against document metadata"),
				Destination: &skipDocCheck,
			},
			&cli.BoolFlag{
				Name:        "verifyAnchorIndependently",
				Usage:       wrap("verify a proof's anchor independently, which does not rely on the proof's anchor URI to do the verification"),
				Value:       true,
				Destination: &verifyAnchorIndependently,
			},
		},
		Action: func(c *cli.Context) error {
			os.Exit(handleCLI(c))
			return nil
		},
	}

	app.Run(os.Args)
}
