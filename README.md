# fortigatecli

FortiGate read-focused CLI for day-to-day operations.

It wraps common FortiGate REST API reads with safer command surfaces for:

- system and platform status
- routing
- firewall objects and policies
- VPN status
- logs and observability
- discovery/schema inspection
- CMDB object lookup
- multi-VDOM fanout
- config backup export

## Build

```bash
go build ./cmd/fortigatecli
```

## Configure

```bash
go run ./cmd/fortigatecli auth init \
  --host https://<FORTIGATE_HOST> \
  --token <API_TOKEN>
```

## Common commands

```bash
go run ./cmd/fortigatecli system status
go run ./cmd/fortigatecli system hostname
go run ./cmd/fortigatecli routing table
go run ./cmd/fortigatecli firewall addresses
go run ./cmd/fortigatecli vpn ipsec status --count 1
go run ./cmd/fortigatecli logs traffic list --count 20
go run ./cmd/fortigatecli discovery capabilities cmdb firewall/address
go run ./cmd/fortigatecli cmdb show firewall/address <name>
go run ./cmd/fortigatecli cmdb list firewall/address --all-vdoms
```

## Output shaping

```bash
go run ./cmd/fortigatecli cmdb list firewall/address \
  --query '.results[*]' \
  --select name \
  --select subnet \
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

## Notes

- `cmdb show <resource> <mkey>` requires a real object name or ID on the target FortiGate.
- `system backup` and `system backup export` can return `403` if the API token does not have sufficient permission.
- `raw get` remains the escape hatch when a domain alias is missing.
- Additional usage guidance is available in [`skills/fortigatecli-usage/SKILL.md`](skills/fortigatecli-usage/SKILL.md).
