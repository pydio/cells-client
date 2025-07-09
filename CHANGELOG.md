# Changes between v4.3.1 and v4.4.0-alpha3

[See Full Changelog](https://github.com/pydio/cells-client/compare/v4.3.1...v4.4.0-alpha3)

- [#1f91404](https://github.com/pydio/cells-client/commit/1f914045e7e822344b1eaef061b8a6d1d500497c): chore: adapt readme now that we have cooler tables
- [#7b3dd88](https://github.com/pydio/cells-client/commit/7b3dd88752ca7ad8dba9c1b4751b03bda5213f64): chore: build with go 1.24
- [#c2b451e](https://github.com/pydio/cells-client/commit/c2b451e506fe6f91eb9ed90e506de95a9fb1ce4f): fix: adapt job tables for latest version of the tablewriter library
- [#0d2b40a](https://github.com/pydio/cells-client/commit/0d2b40a007045da46c6fc45b13562db2620651a3): chore: update TPs
- [#a24cb3d](https://github.com/pydio/cells-client/commit/a24cb3db75032344af3823ed0a05ca4dad4817a1): chore(jobs): fix a few typos in the in-line doc.
- [#ce7b841](https://github.com/pydio/cells-client/commit/ce7b8415c0862e05275aec3892d27ef2541efad2): fix(jobs): make confirmation case insensitive, fix failsafe when trying to force remove more than one system job at a time.
- [#27a2ed1](https://github.com/pydio/cells-client/commit/27a2ed1889c4eb2bd95348840986fca39f650f8d): fix(jobs) confirmation is required for only system jobs
- [#31c6049](https://github.com/pydio/cells-client/commit/31c60498a88fe80456a9e49b87d8983efa555fd0): fix(jobs) fix delete confirmation. User can either use --force parameter or typing for their confirmation
- [#08339b0](https://github.com/pydio/cells-client/commit/08339b03e2e7ae7ba9f4c39245e13128c27ecd7d): fix: small adaptations after review with Tran
- [#7bd7797](https://github.com/pydio/cells-client/commit/7bd7797595dfb203e3ea41811d34b3fb2e89a5ef): fix(jobs) filter operators
- [#3e43ffc](https://github.com/pydio/cells-client/commit/3e43ffc9a1c90b017f2f6b69b08a182369b86751): feat: finally add a command to relog in an existing account
- [#92e3024](https://github.com/pydio/cells-client/commit/92e302434cc2256fad99dbf8d26f20ac075e529f): chore: improve inline doc
- [#d06fb54](https://github.com/pydio/cells-client/commit/d06fb548f14bdcca98d18ba719ef37874a602a69): chore: improve and standardize both inline doc and variable names
- [#009be29](https://github.com/pydio/cells-client/commit/009be29f72fb7a4413648f5f77d9455431278a74): chore: add dependency to enterprise SDK + a few updates
- [#34bd55e](https://github.com/pydio/cells-client/commit/34bd55e184545ad40ae176176a47424bc1d94d2f): fix(jobs): typo in command helps
- [#bb84a79](https://github.com/pydio/cells-client/commit/bb84a7938c85e4adecb3c3e4d4af8002253b9046): fix(jobs): example command
- [#62deda4](https://github.com/pydio/cells-client/commit/62deda47a99be9283e7937183440673a31fec9f3): fix(jobs): fix force parameter
- [#fc7d818](https://github.com/pydio/cells-client/commit/fc7d81831dc412805735b3097ab1635f65281e15): feat(jobs) add commands for jobs management
- [#9d7c9a3](https://github.com/pydio/cells-client/commit/9d7c9a309b5ac4a3bf472c59614cff23625694b1): fix: do not try to call mkdir when the target leaf folder already exists
- [#879fac7](https://github.com/pydio/cells-client/commit/879fac75c8a56f0060f64e430b3f22029134bae2): fix(mkdir): explicitly set "Collection" type when creating distant folder
- [#95b4178](https://github.com/pydio/cells-client/commit/95b417881172468a4f82793b909a55ed0f81a3e0): chore: cosmetic change
- [#6067e9e](https://github.com/pydio/cells-client/commit/6067e9e40e7c57a2b38b81984e23b08b48588b3e): chore: use latest AWS SDK
- [#f6a7160](https://github.com/pydio/cells-client/commit/f6a7160e640992f2db45a29eb5d8bd250ecdb564): Release v4.4.0-alpha02
- [#942393e](https://github.com/pydio/cells-client/commit/942393ee4a18836249cfc2a03d695b8e33081495): chore: update TPs
- [#a251ba2](https://github.com/pydio/cells-client/commit/a251ba2f000478d2255e360a4b2a72dad63266be): Release v4.4.0-alpha01
- [#e22f3af](https://github.com/pydio/cells-client/commit/e22f3af67a2e5b9cb7b18da4409362632a9399fa): fix: finally update table writer library and adapt code
- [#b04f8e3](https://github.com/pydio/cells-client/commit/b04f8e328bd34fdb963091358a3a875352dfd55c): feat: add a hint for the end user if we encounter the XAmzContentSHA256Mismatch error
- [#9f72b74](https://github.com/pydio/cells-client/commit/9f72b747c79cf9ab3c07205a72c67c7db3d55edc): fix: finally update the AWS SDK to the latest version for v4.4 branch
