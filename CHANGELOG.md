## [0.1.1] - 2026-04-19

### Added
- feat(ci): enhance release workflows and scripts
- feat(vscode): implement VSCode extension with tests and CI (#33, #35, #36, #37) (#82)
-  implement maven-indexer-cli dependency resolver (#19) (#80)
-  add integration tests for Node collector wiring (#25) (#78)
-  implement NestJS decorator parser for API endpoint collection (#77)
-  implement Express route parser using tree-sitter-javascript (#22) (#75)
-  implement Fastify route parser using tree-sitter-javascript (#23) (#76)
-  add unit tests for Python collector Collect() (#30) (#74)
-  implement Flask route parser (#28) (#73)
-  implement Django REST Framework parser (#27) (#72)
-  implement FastAPI route parser (#71)
-  add settings management and Postman API client (#69)
- feat(java-parser): Phase 3-4 JAX-RS + Feign parsers, parser refactor (#68)
-  add integration tests with sample projects and separate workflow (#65)
- feat(java-parser): Phase 0-2 实现 + 线程安全修复 (#66)
-  add path params and response examples to Postman formatter (#63)
- feat(curl): add path param substitution and form param support (#31) (#62)
-  wire Gin/Echo/Fiber parsers into Collect() and add unit tests (#14) (#60)
-  implement Fiber route parser using go/ast (#13) (#59)
-  implement Echo route parser using go/ast (#12) (#58)
-  implement Gin route parser using go/ast (#57)
-  implement --help output and integration tests for apilot-cli (#56)
-  add build.Version / build.Date ldflags wiring (#51)
-  add npm wrapper, goreleaser config, scripts, and update CI workflows for Go
-  initial APilot project setup migrated from easy-api

### Fixed
-  support Netflix Feign interfaces without @FeignClient annotation (#81)
-  improve test scripts with better error handling and debugging (#67)

### Changed
-  extract api-model module and replace FormatOptions with typed Params (#54)
-  rename api-collector-support-* to api-collector-* and fix formater typo to formatter

### Improved
- docs: add Node.js/TypeScript parsing strategy research notes (#21) (#79)
- chore: split integration tests by framework using matrix strategy (#70)
- test: add tests for scripts/install.js across all platforms (#64)
- [api-formatter-markdown] Implement simple and detailed templates (#61)
- docs: mark issues #3, #4, #6, #7 as completed in todo.md (#55)
- chore(npm): add .npmignore to keep published package lean (#40) (#53)
- chore: move skills under .agent and add init script (#52)
- docs: fix CI table, config dir, and binary name in architecture.md (#50)
- doc: add contributing-formatters and contributing-collectors guides (#49)
- docs: add implementation todo list
- Initial commit

---

# APilot Changelog

## Unreleased

- Initial release — migrated from [easy-api](https://github.com/tangcent/easy-api) Go engine layer
- Renamed project to APilot
- Added support for Java (Spring MVC, JAX-RS, Feign), Go (Gin, Echo, Fiber), Node.js (Express, Fastify, NestJS), Python (FastAPI, Django REST, Flask)
- Output formats: Markdown, cURL, Postman Collection v2.1
- VSCode extension with bundled binary support
- External plugin system via stdin/stdout JSON protocol
