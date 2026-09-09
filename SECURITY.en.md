# Security Policy

[Russian version](SECURITY.md)

## Reporting a Vulnerability

Do not publish active exploits, tokens, passwords, or panel credentials in public issues. Use the GitHub Private Vulnerability Reporting feature under the Security tab of the repository or contact the maintainer privately.

## Supported Versions

Security patches are published exclusively for the latest stable release of the project.

## Mandatory Operational Rules

- keep the 3x-ui web panel behind a firewall or bound to `127.0.0.1`;
- restrict bot interaction exclusively to administrators via Telegram ID allowlist `ADMIN_IDS`;
- enable TLS certificate verification for production HTTPS panels;
- restrict configuration file access permissions with `chmod 600` on `.env`;
- regularly rotate panel administrative credentials and bot tokens;
- run the bot service under a dedicated unprivileged operating system user;
- never commit configuration files `.env` or SQLite database files `*.db` to Git.
