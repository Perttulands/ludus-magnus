# Agent Academy CLI

Production-quality Go CLI scaffold for Agent Academy.

## Features

- Cobra-based command system
- SQLite persistence via pure Go `modernc.org/sqlite` (no CGO)
- Config and env wiring with Viper
- Styled terminal output and structured logging
- Session creation/listing workflow
- Environment diagnostics with `academy doctor`

## Quick Start

```bash
make build
./bin/academy version
./bin/academy doctor
./bin/academy session new --need "Draft onboarding flow" --mode quickstart
./bin/academy session list
```

## Commands

- `academy version`
- `academy doctor`
- `academy session new --need "..." --mode quickstart`
- `academy session list`

## Development

```bash
make test
make install
make clean
```

## Database

By default, session data is stored in:

- Linux/macOS: `${XDG_CONFIG_HOME:-~/.config}/agent-academy/academy.db`

Override with:

```bash
academy --db /path/to/academy.db session list
```
