# Config Security & Validation (Phase 13)

MUSICA config now includes schema versioning, normalization, and validation.

## Schema

```yaml
version: 1
default_server: my-server
servers:
  - type: navidrome|jellyfin
    name: my-server
    url: https://example.com
    username: user
    password: secret
```

## Guarantees

- config version is migrated to current schema version on load/save
- server type is normalized to lowercase
- names/usernames are trimmed
- trailing slash removed from URLs
- duplicate server names are rejected
- default server must exist in server list
- URL format is validated
- passwords are required and can be redacted via `ServerConfig.Redacted()`

## Notes

- config file permissions remain `0600`
- setup prevents duplicate server names
