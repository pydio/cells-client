# Changes between v2.0.1 and v2.0.2

[See Full Changelog](https://github.com/pydio/cells-client/compare/v2.0.1...v2.0.2)

- [#880cb92](https://github.com/pydio/cells-client/commit/880cb929e80d721884bfba53ff8a9c65dcad0083): Merge remote-tracking branch 'origin/master'
- [#21dce03](https://github.com/pydio/cells-client/commit/21dce03f72cb43f11f4a044f6570231b1b14f8a9): Cosmetic change
- [#e1c5f81](https://github.com/pydio/cells-client/commit/e1c5f81efadbefb0729d0893608a60d6de9f6465): Ignore recycle_bin when using a wildcard
- [#b38913e](https://github.com/pydio/cells-client/commit/b38913e36d9bb14bc9f6dbafac73720c2a7be7b3): Change wildcard character to % as CHANGELOG.html CHANGELOG.md cmd common go.mod go.sum LICENSE main.go Makefile README.md rest is a glob
- [#260ef3c](https://github.com/pydio/cells-client/commit/260ef3c4a9e4607ff9259c4cd66121b8d5f9866f): Use cobra minimumArgs check
- [#bd3b411](https://github.com/pydio/cells-client/commit/bd3b411b13140812d6bd8f78515bc1d0477ded26): Version more info
- [#1f3effc](https://github.com/pydio/cells-client/commit/1f3effc88fb5ea3456163a5276246421cea6b9d1): Fix error handling
- [#f219ae0](https://github.com/pydio/cells-client/commit/f219ae07f0a2922de639e81d2ec3d750caaf16bf): Make us of cobra minimumArgs check.
- [#af54424](https://github.com/pydio/cells-client/commit/af54424f85cd1ea950d147240677e005ae73ca3f): Skip verify applied to refresh token.
- [#7c46d5c](https://github.com/pydio/cells-client/commit/7c46d5ca9807ec53aca6a67f9282349475dfdd3b): Skip verify for oauth configuration was not wired.
- [#83bd7e8](https://github.com/pydio/cells-client/commit/83bd7e889951710f058c8bcdbf4b91061b545fc7): Added a confirmation & --force flag for the ./cec rm command
- [#4c17143](https://github.com/pydio/cells-client/commit/4c17143965de9c64a50cce7a94f20bc9c4636869): Removed message with job uuid (added a more explicit message if nodes are deleted)
- [#04df236](https://github.com/pydio/cells-client/commit/04df236b2325419271aae4faa4c509243471c2a8): If target folder + CHANGELOG.html CHANGELOG.md cmd common go.mod go.sum LICENSE main.go Makefile README.md rest delete only the children
- [#0a8c0d4](https://github.com/pydio/cells-client/commit/0a8c0d4398f7b5f5002b96b13b1721dffe0f2871): If there is nothing to delete be more verbose and do not attempt to delete the nil target
- [#ce8a4ff](https://github.com/pydio/cells-client/commit/ce8a4ff323cd3f9dc25e3ec93975ffa18594d53f): Make sure to append the node to the same list when deleting multiple targets
- [#827b33c](https://github.com/pydio/cells-client/commit/827b33c4c66ca31e31938a1a026460e03813768a): Update cells-sdk-go and remove need for id/secret parameters
