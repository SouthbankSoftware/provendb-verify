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
 * @Date:   2019-04-02T13:35:55+11:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-07-09T16:52:03+10:00
 */

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/network/connstring"
	log "github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v2"
)

func handleCLI(c *cli.Context) int {
	if c.Bool("help") {
		cli.ShowAppHelpAndExit(c, 0)
	}

	if c.NArg() > 0 {
		return cliErrorf("No args should be provided")
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	uri := c.String("uri")

	cs, err := connstring.Parse(uri)
	if err != nil {
		return cliErrorf("%s", err)
	}

	if !cs.MaxPoolSizeSet {
		cs.MaxPoolSize = defaultMaxPoolSize
		cs.MaxPoolSizeSet = true
	}

	host := c.String("host")
	port := c.String("port")

	if host != "" || port != "" {
		if len(cs.Hosts) != 1 {
			return cliErrorf("'--host' or '--port' cannot be used to override multiple hosts in URI")
		}

		addr := strings.Split(cs.Hosts[0], ":")
		h := addr[0]
		p := defaultMongoDBPort

		if len(addr) >= 2 {
			p = addr[1]
		}

		if host != "" {
			h = host
		}

		if port != "" {
			d, err := strconv.Atoi(port)
			if err != nil {
				return cliErrorf("port must be an integer: %s", err)
			}
			if d <= 0 || d >= 65536 {
				return cliErrorf("port must be in the range [1, 65535]")
			}
			p = strconv.Itoa(d)
		}

		cs.Hosts = []string{h + ":" + p}
	}

	if c.IsSet("ssl") || !cs.SSLSet {
		cs.SSL = c.Bool("ssl")
		cs.SSLSet = true
	}

	if u := c.String("username"); u != "" {
		cs.Username = u
	}

	if p := c.String("password"); p != "" {
		cs.Password = p
	}

	if a := c.String("authDatabase"); a != "" {
		cs.AuthSource = a
	}

	if d := c.String("database"); d != "" {
		cs.Database = d
	}

	if c.IsSet(provenDBVersionIDKey) && c.IsSet(provenDBProofIDKey) {
		return cliErrorf("'--%s' and '--%s' cannot be both set", provenDBVersionIDKey, provenDBProofIDKey)
	}

	var versionID int64 = -1

	if v := c.String(provenDBVersionIDKey); v != provenDBVersionCurrent {
		vNum, err := strconv.Atoi(v)
		if err != nil {
			return cliErrorf("invalid '--%s': %s", provenDBVersionIDKey, err)
		}

		if vNum < 1 {
			return cliErrorf("invalid '--%s': version must be >= 1", provenDBVersionIDKey)
		}

		versionID = int64(vNum)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var proof interface{}

	if in := c.String("in"); in != "" {
		if strings.HasSuffix(in, ".zip") {
			msg, err := verifyProofArchive(ctx, in)
			if err != nil {
				return cliFalsifiedf("%s:\n\t%s", msg, err)
			}

			return cliVerifiedf("%s", msg)
		}

		proof, err = loadProof(in)
		if err != nil {
			return cliErrorf(err.Error())
		}

		fmt.Printf("Loading Chainpoint Proof `%s`...\n", in)
	}

	var (
		cols     []string
		database *mongo.Database
		opts     []interface{}
	)

	if cs.Database == "" {
		if proof == nil {
			return cliErrorf("please specify a database as the verification target")
		}
	} else {
		cOpts := options.Client()
		cOpts.ConnString = cs

		client, err := mongo.NewClientWithOptions("mongodb://localhost", cOpts)
		if err != nil {
			return cliErrorf(err.Error())
		}

		err = client.Connect(ctx)
		if err != nil {
			return cliErrorf(err.Error())
		}

		database = client.Database(cs.Database)

		if c.Bool("listVersions") {
			versions, err := getVerifiableVersions(ctx, database)
			if err != nil {
				return cliErrorf("failed to list verifiable versions: %s", err)
			}

			fmt.Printf("%-36s\t%-9s\t%-30v\t%s\n", provenDBProofIDKey, provenDBVersionKey, provenDBSubmittedKey, provenDBStatusKey)
			for _, v := range versions {
				fmt.Printf("%-36s\t%-9v\t%-30v\t%s\n", v.proofID, v.versionID, v.submitTimestamp, v.proofStatus)
			}
			return 0
		}

		colName := c.String("collection")

		if docFilter := c.String("docFilter"); colName != "" && docFilter != "" {
			opts = append(opts, docOpt{
				colName,
				docFilter,
				false,
			})
		} else if colName != "" || docFilter != "" {
			return cliErrorf("'--collection' and '--docFilter' must be both specified or left out")
		}

		if out := c.String("out"); out != "" {
			if ext := filepath.Ext(out); ext != ".json" && ext != ".txt" {
				return cliErrorf("filename in '--out' must end in either '.json' or '.txt'")
			}

			opts = append(opts, outOpt{
				out,
			})
		}

		if versionID == -1 {
			if p := c.String(provenDBProofIDKey); p != "" {
				// use proofId to get versionId
				var storedProof interface{}
				storedProof, versionID, cols, err = getProof(ctx, database, p, colName)
				if err != nil {
					return cliErrorf("cannot get Chainpoint Proof using %s %s", provenDBProofIDKey, p)
				}

				if proof == nil {
					proof = storedProof
				}
			} else {
				versionID, err = getLatestVerifiableVersion(ctx, database)
				if err != nil {
					return cliErrorf("failed to get the latest verifiable version: %s", err)
				}
			}
		}

		if proof == nil {
			proof, _, cols, err = getProof(ctx, database, versionID, colName)
			if err != nil {
				return cliErrorf("cannot get Chainpoint Proof using %s %v: %s", provenDBVersionKey, versionID, err)
			}
		}
	}

	msg, err := verifyProof(ctx, database, proof, versionID, cols, opts...)
	if err != nil {
		return cliFalsifiedf("%s:\n\t%s", msg, err)
	}

	return cliVerifiedf("%s", msg)
}

func cliVerifiedf(format string, a ...interface{}) int {
	pass := color.New(color.BgHiGreen, color.FgHiWhite, color.Bold).SprintFunc()
	args := []interface{}{pass(" PASS ")}
	args = append(args, a...)
	// use color.Output to be windows compatible for colors
	fmt.Fprintf(color.Output, "%s "+format+"\n", args...)
	return 0
}

func cliFalsifiedf(format string, a ...interface{}) int {
	fail := color.New(color.BgHiRed, color.FgHiWhite, color.Bold).SprintFunc()
	args := []interface{}{fail(" FAIL ")}
	args = append(args, a...)
	// use color.Output to be windows compatible for colors
	fmt.Fprintf(color.Output, "%s "+format+"\n", args...)
	return 2
}

func cliErrorf(format string, a ...interface{}) int {
	// use color.Output to be windows compatible for colors
	fmt.Fprintf(color.Output, format+"\n\n"+defaultErrorHelpMsg+"\n", a...)
	return 1
}

func wrap(usage string) string {
	return wrapText(usage, 60, "\n\t") + "\n\t"
}

func wrapText(text string, lineWidth int, breakStr string) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += breakStr + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped
}
