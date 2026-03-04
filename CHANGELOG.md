# Changelog

## [0.4.0](https://github.com/RevoTale/blog/compare/v0.3.3...v0.4.0) (2026-03-04)


### Features

* devcontainer ([b32bf30](https://github.com/RevoTale/blog/commit/b32bf300ded389cf5f83ca9c972f11e2c9533c90))
* fix the proxying the iamages to our infrastructure ([04475c5](https://github.com/RevoTale/blog/commit/04475c54736b28f1615f50e4f30ddbc2e2e5e009))
* install deadcode validator and remove dead code ([7b22eb5](https://github.com/RevoTale/blog/commit/7b22eb5be09cd03d747cd368b048b2a2bfab5841))
* logging of the data resolvers timing. ([af6d30a](https://github.com/RevoTale/blog/commit/af6d30a660149f3426f2750115f1948d88447a5b))
* setup dev environment separately from https://github.com/RevoTale/backend ([62c34e4](https://github.com/RevoTale/blog/commit/62c34e4cbb48c8ceb134a660ae9ed4329ea0c4a3))


### Bug Fixes

* devcontainer image ([5b9e60a](https://github.com/RevoTale/blog/commit/5b9e60a4ea09aa84680c3b35e128b739d3baad65))
* image sizes wrong ([aa7e35c](https://github.com/RevoTale/blog/commit/aa7e35c1c82f5fe8610b7b376b2e4346e6daace3))
* wrong image sized used that did not match the server image thumbnails ([2aaa6de](https://github.com/RevoTale/blog/commit/2aaa6de0b02660de34af880d484473c057223207))

## [0.3.3](https://github.com/RevoTale/blog/compare/v0.3.2...v0.3.3) (2026-03-03)


### Bug Fixes

* broken images due to dynamic preset size ([02b9247](https://github.com/RevoTale/blog/commit/02b9247bb6b01197041e0f0f50aeb2a26d25d945))

## [0.3.2](https://github.com/RevoTale/blog/compare/v0.3.1...v0.3.2) (2026-03-03)


### Bug Fixes

* second-level headers has tag h1 in the markdown rendering. That hurts SEO ([9f2def4](https://github.com/RevoTale/blog/commit/9f2def4d84896e4d5c58f88f92df5c493a300ecd))

## [0.3.1](https://github.com/RevoTale/blog/compare/v0.3.0...v0.3.1) (2026-03-03)


### Bug Fixes

* indexing disabled due to robos tags ([37b7b3e](https://github.com/RevoTale/blog/commit/37b7b3e410708ad55a219d496a08c0312f6b7719))

## [0.3.0](https://github.com/RevoTale/blog/compare/v0.2.0...v0.3.0) (2026-03-03)


### Features

* language switcher and footer design adjustments ([b7f5dff](https://github.com/RevoTale/blog/commit/b7f5dffb7e58d2529dde0b250a28a324c9c0de67))
* rss feed button ([f8cf21d](https://github.com/RevoTale/blog/commit/f8cf21d35da52b9b54cf08be815320d5abe0121c))
* seo sitemaps and rss feeds. ([6c494dc](https://github.com/RevoTale/blog/commit/6c494dc33f1402272a6369d110c75a28d0a3f749))
* separate image component similat to NextJS. Custom loader for the our internal infrastructure ([b700217](https://github.com/RevoTale/blog/commit/b70021759f41422b0864ba70c54cd6e1475aaf57))

## [0.2.0](https://github.com/RevoTale/blog/compare/v0.1.0...v0.2.0) (2026-03-03)


### Features

* `public` directory with static asssets that serves from the root and replacated what `public` dir does in nextjs ([25d36ff](https://github.com/RevoTale/blog/commit/25d36ff3665cab6d7cbb7038c0dcf3952fafc246))
* 404 page ([5db4662](https://github.com/RevoTale/blog/commit/5db4662b20073cb57cadbb392c6602d4d54c3465))
* add a sidebar filters with the tags, short/long tales. Add canonical pagess for all them ([ec3a535](https://github.com/RevoTale/blog/commit/ec3a5354a81673f2ff74fb2a2970d7ecbfcd56e2))
* add public code link ([ff27927](https://github.com/RevoTale/blog/commit/ff279278aca4b239a1b4e97daf7b521ac4ddc5a0))
* add the search bar ([51f8884](https://github.com/RevoTale/blog/commit/51f8884e9f2eebbfe45a4f22d88e0cdcd8806dea))
* badge for not visited notes. ([d2d2619](https://github.com/RevoTale/blog/commit/d2d2619f6560016bcf86112fdde24ed740534fe3))
* CI ([c5a7a03](https://github.com/RevoTale/blog/commit/c5a7a0393da7a21e1bf43d5292cc9a35e2390eac))
* clear button for the search bar ([fdc04b8](https://github.com/RevoTale/blog/commit/fdc04b827bcf270ba774e37bb4c45452792c7bd1))
* design the i18 core. Translate all pages the same way I do current on the NextJS-written blog. ([b01a437](https://github.com/RevoTale/blog/commit/b01a437cbab93bd78d7d95395220ba187eba0cfe))
* display the badge  for unvisited pages on the bottom level and fix the issue with the browser privacy forbidding it. ([bbb80c2](https://github.com/RevoTale/blog/commit/bbb80c2c5ed5bf7b8ed584aa3f08157d091dbe8e))
* display the real techonolgy stack used for the blog in the footer ([0e501b7](https://github.com/RevoTale/blog/commit/0e501b72274336c5ed237cc857311422de0c8116))
* document the architecture choices to not forget them later ([f847513](https://github.com/RevoTale/blog/commit/f8475136e0c06e36101886f50237854953e93bd1))
* extract the http server to the `framework` module to do conver tto the  complete frameowkr later. ([7bac77b](https://github.com/RevoTale/blog/commit/7bac77b26d6e7bd782f443fbe01924d79f65d5b8))
* fix CI ([3a6bf54](https://github.com/RevoTale/blog/commit/3a6bf5412daacab3146b4fc6cb862bfe921d9cea))
* gzip compression and static build info ([5ffcbe2](https://github.com/RevoTale/blog/commit/5ffcbe2d9cfa2c30d6483d2c068ec7ac3e97456b))
* live state changes will live under`/.live` subpath to avoid the real routes and caching collisions. ([f24bab6](https://github.com/RevoTale/blog/commit/f24bab6ff9fc4dcab2fe87f29e1c7e7479ae7e30))
* make the author name always blue ([784081e](https://github.com/RevoTale/blog/commit/784081ee9d5390b11bba2b91718145b9fff5a62f))
* make the NextJS-like metadata generator pattern and live replacement. ([7f6b9a8](https://github.com/RevoTale/blog/commit/7f6b9a8600339cdd4dc7293cc8ea95b4e3560188))
* mark the generated code to avoid confison ([82e4e0f](https://github.com/RevoTale/blog/commit/82e4e0fe3c695844c782aee8a241a452bcd53b15))
* migrate metadata of the layout and pages from the legacy RevoTale blog NextJS app. ([7f6b9a8](https://github.com/RevoTale/blog/commit/7f6b9a8600339cdd4dc7293cc8ea95b4e3560188))
* migrate the copy button from internal blog ([9368bfb](https://github.com/RevoTale/blog/commit/9368bfb247f5c4e8b90d54e8847766e8dafb7fad))
* remove the `datastar`. Migrate to the to the `htmx`. ([de33357](https://github.com/RevoTale/blog/commit/de333578be22e523a3edacdc159bae6bace278c0))
* reogranize the datat resolvers to share the single namespace and one generate file with the definition. Much easier to read ([a4f3d0c](https://github.com/RevoTale/blog/commit/a4f3d0cc3667cf23f3fbf4d8f7620268a833087c))
* Replace the `/static` url with the `/.revotale` url. ([7bac77b](https://github.com/RevoTale/blog/commit/7bac77b26d6e7bd782f443fbe01924d79f65d5b8))
* ue the system level fonts for better readability and redcue the download size ([6b50b7f](https://github.com/RevoTale/blog/commit/6b50b7f5e8d22c861a30f19524c8a51f98443f4e))
* use the esbuild for the statiuc files path hashing and minifiiing. ([edd11c9](https://github.com/RevoTale/blog/commit/edd11c92a54c1ee8ed4c1fe2205097c82558f27c))


### Bug Fixes

* add all font variants: ([db2aa7e](https://github.com/RevoTale/blog/commit/db2aa7e14486f8a34113d3db046e5c2e1e67b181))
* **deps:** update all non-major dependencies ([79ac1b1](https://github.com/RevoTale/blog/commit/79ac1b14661f6660c4f4d32aba9394da096119ab))
* **deps:** update all non-major dependencies ([a2dc121](https://github.com/RevoTale/blog/commit/a2dc12197b730eac679cdf3437b1d9be37cfa4ea))
* golint validtion fails ([7bb9f23](https://github.com/RevoTale/blog/commit/7bb9f23b39fe224f323594458590730e5bbaf98d))
* L styles adjustment to be more mobile froiendly and remove unnesesarry elements ([db2aa7e](https://github.com/RevoTale/blog/commit/db2aa7e14486f8a34113d3db046e5c2e1e67b181))
* remove the udnerline ([758bc80](https://github.com/RevoTale/blog/commit/758bc80a312b67425ab178b4f84824db8dca8e97))
* search bar becomes block too late ([f9a9dcf](https://github.com/RevoTale/blog/commit/f9a9dcfe928751cf464e19a0f3853447d5b67e89))
* single blog post  centered well ([f68657e](https://github.com/RevoTale/blog/commit/f68657e3a5ad1870bc5e2cf0be2679a2e8548977))
* single page background and footer colors ([379619f](https://github.com/RevoTale/blog/commit/379619fece8ba2893848be7c9a4918248a6a46cc))

## Changelog

All notable changes to this project will be documented in this file.
