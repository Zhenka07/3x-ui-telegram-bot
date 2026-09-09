# Contributing

[Russian version](CONTRIBUTING.md)

1. Open an issue describing the task or bug.
2. Create a separate feature branch for your changes.
3. Do not commit secrets, tokens, or real client connection links.
4. Add automated tests covering the modified behavior.
5. Run `go test ./...` and `go vet ./...` before submitting code.
6. Describe 3x-ui compatibility and manual verification steps in your pull request.

Use anonymized test data when updating panel API integration. Real UUIDs, passwords, Reality keys, Telegram tokens, server IP addresses, and `x-ui.db` files are strictly forbidden in issues and commits.

All functions and methods must include standard Go doc comments in English, and redundant inline comments inside function bodies are not permitted.
