# Changes between v2.0.2 and v2.0.3

[See Full Changelog](https://github.com/pydio/cells-client/compare/v2.0.2...v2.0.3)

- [#f07c830](https://github.com/pydio/cells-client/commit/f07c83093d55e371afe29c39cabde412144f378e): Changed ./cec to cec in last example of README
- [#1a63248](https://github.com/pydio/cells-client/commit/1a63248ecc9afd7db781605e917dc5fdc5406a5d): A few cosmetic changes after code review with JThabet
- [#76840f5](https://github.com/pydio/cells-client/commit/76840f5afcf9e1b0c5105b9c4aedeb406f83caf2): Merge remote-tracking branch 'origin/master'
- [#8cd204a](https://github.com/pydio/cells-client/commit/8cd204a4b2325e110d0aed8bcbf2ea126b9dcaa7): Remove useless quotes
- [#53129bc](https://github.com/pydio/cells-client/commit/53129bc249976a3b7412397e9af2794b2302a83c): Remove blocking check on used env variables
- [#8e020cc](https://github.com/pydio/cells-client/commit/8e020cc74f01cd213dcdce39a7c8f90133ae17f2): Add scopes that were implicit in legacy version
- [#af825ab](https://github.com/pydio/cells-client/commit/af825ab70117f8c13871ae621e31a5c2e37f51ce): Bump go version to 1.13 in Makefile
- [#65d9cf7](https://github.com/pydio/cells-client/commit/65d9cf7cc863efed160056cc6a27e2d1adfa9ed7): Update third party versions
- [#491db0c](https://github.com/pydio/cells-client/commit/491db0c7dcbb30315be46fd3609381ddcf0f987a): cosmetic changes and cleanup for the completion command examples.
- [#06e33e1](https://github.com/pydio/cells-client/commit/06e33e1b1282385d02bf760ce6b0caf40991c711): If file size is bigger than 100MB use multipart upload.
- [#d4df905](https://github.com/pydio/cells-client/commit/d4df9050f3a140e517dfe02bfaee9c0f906c0b8f): Set concurrency to 3 and part size to 50 mb
- [#0159dfd](https://github.com/pydio/cells-client/commit/0159dfdbf9b989b333a2d74f73ba690cba158d2e): Remove execution rights when creating config.json file
- [#80473ce](https://github.com/pydio/cells-client/commit/80473ce4edbe853191c3ce45353359a8e0b05f26): Add lock on RefreshAndStoreIfRequired, always update request config for each part.
- [#57bbd32](https://github.com/pydio/cells-client/commit/57bbd3220141e8b6a6a7524be9baaaa3ffadad3d): Work in progress, multi part upload (upload manager from aws sdk go)
- [#fb8b7cc](https://github.com/pydio/cells-client/commit/fb8b7cca41b212f2b8df73d43fac15bbc27d34bb): Added a RefreshAndStoreIfRequired method to facilitate token refresh handling
- [#8095817](https://github.com/pydio/cells-client/commit/8095817cdb367897b2765fb7b26501ff86c87a94): Check minimum args with cobra builtin
- [#ab6e1d1](https://github.com/pydio/cells-client/commit/ab6e1d151a2f0bcad80d676a4e78543c0e68180a): go template for version command
- [#896f231](https://github.com/pydio/cells-client/commit/896f23167cd39e03c53098e3af38fbdfb5f32a08): When refreshing should also writeFile with R/W rights the Execution rights are not required.
- [#66800ce](https://github.com/pydio/cells-client/commit/66800ceb891f07fa9496565bb0faf3c7b59eb8ad): Config json file should only have R/W, no Execution.
- [#366713b](https://github.com/pydio/cells-client/commit/366713b7e2da31e2303fa5bba6a89ebfbed64493): Changed ./cec to cec in last example of README
