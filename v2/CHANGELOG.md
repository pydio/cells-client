# Changes between v2.0.5 and v2.0.6

[See Full Changelog](https://github.com/pydio/cells-client/compare/v2.0.5...v2.0.6)

- [#95464ea](https://github.com/pydio/cells-client/commit/95464ea705fcb1705901b0adb81bbdc907dc1dc9): Update dependencies
- [#827a991](https://github.com/pydio/cells-client/commit/827a99196b1efb9017130896aa058cf554a7be45): Do not advert tech commands (in doc), various minor fixes
- [#e1cb4ed](https://github.com/pydio/cells-client/commit/e1cb4edccb5811c06c2b2cd4a35a69a03dc98d65): Added a randSeed to make sure that each time the state is random
- [#6749bd7](https://github.com/pydio/cells-client/commit/6749bd7db2ea7a46fdae30ee1a3d83e28709e862): Update cells-sdk-go dependency
- [#3b9a722](https://github.com/pydio/cells-client/commit/3b9a722dc140e44e7df56ee4334acf29fe98b307): Revert to v2.0.5 state
- [#80b8b97](https://github.com/pydio/cells-client/commit/80b8b979c800735e301ecbc7bd397c283f6398a1): update cells-sdk-go dependency
- [#d32433a](https://github.com/pydio/cells-client/commit/d32433a549e3ad2182223d2cb0c3f4f1ba4ee3de): reverted rights on folder
- [#b2b0d0e](https://github.com/pydio/cells-client/commit/b2b0d0ea1ac2b9fad668bc9c0753a71b2cbb7b28): Make defaultConfigFilePath method private, we should only use GetConfigFilePath()
- [#e099ae9](https://github.com/pydio/cells-client/commit/e099ae988df210aaf83d2b816110081cbae89fd9): create file with RW.
- [#0473b02](https://github.com/pydio/cells-client/commit/0473b02aa17d0ea5134661ef9c6f818ae51658bd): Files and folders should only have Read and Write.
- [#e14df98](https://github.com/pydio/cells-client/commit/e14df986b9188f384447f07dc0b77e925aaf9a52): Removed a breaking change from a previous commit.
- [#6f41dc9](https://github.com/pydio/cells-client/commit/6f41dc9d5057be20e11a8e7aed657a61162f9262): Info command displays a table with User and URL of the active configuration.
- [#0549573](https://github.com/pydio/cells-client/commit/0549573a0f7158ad77b8c1e718d16b04dc01c78e): Added messages on the update process to be more descriptive on what is happening during the process.
- [#3908366](https://github.com/pydio/cells-client/commit/3908366c9f45b1df3504dc9d35c618bf78654f95): renamed import name to be easier to understand
- [#f825947](https://github.com/pydio/cells-client/commit/f825947b65cb63414f411ec372d619be11a8fa9f): Info command now will display the server URL and User of the current configuration
- [#fce03e4](https://github.com/pydio/cells-client/commit/fce03e49c2820b8662c1290521eedd07acb829f4): Updated the message to be grammatically correct
- [#dbd8303](https://github.com/pydio/cells-client/commit/dbd83035d911a7f9a266cc5a071007bb5ce04fa5): Cleaner name for the old binary after update
- [#2ea3c2c](https://github.com/pydio/cells-client/commit/2ea3c2cc84ce032dab1ea98ee22aeaeb3f8078bf): Remove unstable channel option.
- [#11ad35c](https://github.com/pydio/cells-client/commit/11ad35c295cb3f3594d1309c5e72f56af4ad3162): Update command has 2 new flags to select (dev, unstable) update channels.
- [#5be703d](https://github.com/pydio/cells-client/commit/5be703d9dad451da71384b7e81edc98f1975c6c4): Removed uber zap, not used
- [#f7de1c1](https://github.com/pydio/cells-client/commit/f7de1c15a0aa09af38ede960c364f96b57b5942e): Bump dependency versions
- [#078826a](https://github.com/pydio/cells-client/commit/078826a0085ffaaca230a3e5a3bc300c731e60d4): Fix error message
- [#544f505](https://github.com/pydio/cells-client/commit/544f50519e5f1db6c38da6fc92847a2da78f42a8): Added command info, to display the current server that the commands will be applied on.
- [#e376f51](https://github.com/pydio/cells-client/commit/e376f5110b98122ba6401ea3e1db1f63823a255e): update go mod
- [#50acae2](https://github.com/pydio/cells-client/commit/50acae22382aa081fe6017c0a144cc5a97951bd2): Displays the current user by parsing the ID token, also updated a variable name for easier readability
- [#da68493](https://github.com/pydio/cells-client/commit/da68493210edb88c0561c98347d640bf0a0ec27e): Displays the current user by parsing the ID token, also updated a variable name for easier readability
- [#ef21a53](https://github.com/pydio/cells-client/commit/ef21a53ce49a6e40674be90ed58d140ff7a0325c): Hides the command doc (it is still usable but will just not be displayed in the command helper)
- [#c68a149](https://github.com/pydio/cells-client/commit/c68a149480be941fb8101a156f3e503430c915b1): Do not statNode if p is empty (not required for ./cec ls <empty>)
- [#0ba9211](https://github.com/pydio/cells-client/commit/0ba92115d1bf0307f826154935b5e7b0a5318254): Added a randSeed to make sure that each time the state is random
