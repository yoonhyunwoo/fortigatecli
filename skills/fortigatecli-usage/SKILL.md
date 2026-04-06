---
name: "fortigatecli-usage"
description: "Use this skill when you need practical command examples and operating guidance for fortigatecli in this repository."
argument-hint: "[optional-command-area]"
allowed-tools: "Read, Grep, Bash(go run ./cmd/fortigatecli*), Bash(go build ./cmd/fortigatecli), Bash(go test ./...)"
---

# fortigatecli-usage

Use this skill when a user wants to know how to operate `fortigatecli` from this repository.

## Current contract

- The CLI entrypoint is `./cmd/fortigatecli`.
- Credentials are stored through `fortigatecli auth init`.
- Default output is JSON.
- `cmdb`, `monitor`, and `raw get` support read-oriented flags.
- `--all-vdoms` is available on read commands such as `cmdb`, `monitor`, and `raw get`.
- `system backup` is stdout-oriented.
- `system backup export` writes to a file and may fail with `403` if the API token lacks backup permission.

## Basic setup

```bash
go run ./cmd/fortigatecli auth init \
  --host https://<FORTIGATE_HOST> \
  --token <API_TOKEN>
```

## Common commands

```bash
go run ./cmd/fortigatecli system status
go run ./cmd/fortigatecli system hostname
go run ./cmd/fortigatecli vpn ipsec status --count 1
go run ./cmd/fortigatecli cmdb list firewall/address --count 10
go run ./cmd/fortigatecli discovery capabilities cmdb firewall/address
```

## Useful read patterns

```bash
go run ./cmd/fortigatecli monitor interfaces --eq name=wan1
go run ./cmd/fortigatecli raw get /api/v2/cmdb/firewall/address --count 5
go run ./cmd/fortigatecli cmdb show firewall/address <name>
go run ./cmd/fortigatecli cmdb address list --page-size 50 --page 2
go run ./cmd/fortigatecli cmdb list firewall/address --all-vdoms
```

## Output shaping

```bash
go run ./cmd/fortigatecli cmdb list firewall/address \
  --query '.results[*]' \
  --select name \
  --select subnet
```

```bash
go run ./cmd/fortigatecli cmdb list firewall/address \
  --query '.results[*]' \
  --select name \
  --output table
```

## Backup

Print to stdout:

```bash
go run ./cmd/fortigatecli system backup
```

Export to file:

```bash
go run ./cmd/fortigatecli system backup export \
  --scope global \
  --output /tmp/fortigate.conf
```

Dry run:

```bash
go run ./cmd/fortigatecli system backup export \
  --scope vdom \
  --vdom root \
  --output /tmp/fortigate.conf \
  --dry-run
```

## Execution rules

1. Prefer `go run ./cmd/fortigatecli ...` when giving interactive examples.
2. Use `cmdb list ...` to discover actual object names before using `cmdb show ...`.
3. Treat `403` on backup/export as a token permission or FortiGate policy issue before assuming a CLI bug.
4. Use `--all-vdoms` only on read commands that actually need multi-VDOM fanout.
5. Keep `raw get` as the escape hatch when a higher-level alias is missing.

## Notes

- `cmdb show <resource> <mkey>` requires an existing object name or ID on the target device.
- `vpn ipsec tunnel <name>` is implemented as monitor-based status detail, not config detail.
- Discovery commands are safe wrappers around schema/field/capability inspection and are narrower than `raw get`.
