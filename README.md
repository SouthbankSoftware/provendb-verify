# provendb-verify

ProvenDB Open Source Verification CLI

[![Powered by ProvenDB](https://img.shields.io/static/v1.svg?label=Powered%20By&message=ProvenDB&color=35b3d4&labelColor=1c4d6b&link=https:/provendb.com&link=https:/provendb.com&logo=data:image/png;base64,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)](http://provendb.com) [![License](https://img.shields.io/github/license/SouthbankSoftware/provendb-verify.svg)](https://github.com/SouthbankSoftware/provendb-verify/blob/master/LICENSE)

Test status

[![provendb-verify-test](http://concourse.provendb.com/api/v1/pipelines/provendb-verify-test/jobs/test/badge)](http://concourse.provendb.com/teams/main/pipelines/provendb-verify-test)

Deploy status

[![provendb-verify-deploy](http://concourse.provendb.com/api/v1/pipelines/provendb-verify-deploy/jobs/build-and-deploy/badge)](http://concourse.provendb.com/teams/main/pipelines/provendb-verify-deploy)

`provendb-verify` is an open source command line tool that allows ProvenDB/Chainpoint Proofs or Archives to be verified independently of the ProvenDB DBaaS.

* Build: `make`
* Usage: `./provendb-verify -h`
* Build for all platforms: `make build-all`
* [ReadMe.io doc](https://provendb.readme.io/docs/independently-validating-your-blockchain-proofs#section-use-provendb-verify-to-validate-your-proofs)

## Makefile Options

Following Makefile options are global to all `go` commands

### `APP_VERSION`

The CLI release version

Example:

```bash
APP_VERSION=0.0.2 make

./provendb-verify -v
# provendb-verify version 0.0.2
```

### `BC_TOKEN`

The [BlockCypher](https://www.blockcypher.com/) access token to be used in the CLI

Example:

```bash
BC_TOKEN=${YOUR_TOKEN_HERE} make test-dev
```

## Packages

### [`merkle`](https://github.com/SouthbankSoftware/provendb-verify/tree/master/pkg/merkle)

Contains the generic hashing interface for bags

### [`merkle.chainpoint`](https://github.com/SouthbankSoftware/provendb-verify/tree/master/pkg/merkle/chainpoint)

Contains the Chainpoint flavored merkle tree implementation for above hashing interface

### [`crypto`](https://github.com/SouthbankSoftware/provendb-verify/tree/master/pkg/crypto)

Contains convenient helpers for generating hashes, such as sha256

### [`proof`](https://github.com/SouthbankSoftware/provendb-verify/tree/master/pkg/proof)

Contains all the logics to manipulate and verify ProvenDB/Chainpoint Proofs
