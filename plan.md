# Plan: NATS JWT Terraform Provider

## TL;DR

Build `terraform-provider-natsjwt` — a Go-based Terraform provider using the plugin framework that manages NATS JWT credentials offline (no running server needed). One resource (`natsjwt_nkey`) generates NKeys; five data sources compute signed JWTs for operators, accounts, system accounts, users, and a config helper that outputs NATS server configuration. JWTs are made deterministic by pinning `IssuedAt=0` and `JTI=""` so plans are stable when inputs are unchanged. Module path: `github.com/m3nowak/terraform-provider-natsjwt`.

## Steps

### Phase 1: Project Scaffolding

1. Initialize Go module `github.com/m3nowak/terraform-provider-natsjwt` with `go mod init`
2. Add dependencies:
   - `github.com/hashicorp/terraform-plugin-framework` (latest stable, ~v1.14+)
   - `github.com/hashicorp/terraform-plugin-testing` (for acceptance tests)
   - `github.com/nats-io/nkeys` (v0.4.x)
   - `github.com/nats-io/jwt/v2` (v2.8.x)
3. Create `main.go` — provider server entrypoint using `providerserver.Serve()` with address `registry.terraform.io/m3nowak/natsjwt`
4. Create `internal/provider/provider.go` — provider definition implementing `provider.Provider` interface:
   - `Schema()` — empty (no provider-level config needed, all inputs are per-resource/data-source)
   - `Configure()` — no-op
   - `Resources()` — returns `[natsjwt_nkey]`
   - `DataSources()` — returns `[natsjwt_operator, natsjwt_account, natsjwt_system_account, natsjwt_user, natsjwt_config_helper]`
5. Create `mise.toml` tasks for: `build`, `install`, `test`, `testacc`, `generate` (docs), `lint`
6. Create `.goreleaser.yml` for release builds
7. Create `examples/` directory structure for tfplugindocs

### Phase 2: `natsjwt_nkey` Resource

8. Create `internal/provider/resource_nkey.go`:
   - **Schema inputs**:
     - `keepers` — `schema.MapAttribute{ElementType: types.StringType, Optional: true}` with `RequiresReplaceIfValuesNotNull()` plan modifier
     - `type` — `schema.StringAttribute{Required: true}` with validator accepting `"operator"`, `"account"`, `"user"` only; `RequiresReplace()` plan modifier
   - **Schema outputs** (Computed):
     - `seed` — `schema.StringAttribute{Computed: true, Sensitive: true}` with `UseStateForUnknown()` plan modifier
     - `public_key` — `schema.StringAttribute{Computed: true}` with `UseStateForUnknown()` plan modifier
   - **Create**: Call `nkeys.CreatePair()` with appropriate `PrefixByte` based on `type`. Store seed and public key in state.
   - **Read**: Re-derive public key from stored seed using `nkeys.FromSeed()`. If seed is gone, mark resource for recreation.
   - **Update**: Only `keepers` can change without replacement; no re-generation needed if only keepers change without null transitions. Seed/public_key stay the same.
   - **Delete**: No-op (state removal)
   - **ImportState**: Not supported (seeds are generated, not importable)

9. Create `internal/provider/resource_nkey_test.go`:
   - Test creating each key type (operator, account, user)
   - Test that seed prefix matches type (SO*, SA*, SU\*)
   - Test keepers-triggered replacement
   - Test invalid type is rejected
   - Test public key format validation (O*, A*, U\*)

### Phase 3: `natsjwt_operator` Data Source

10. Create `internal/provider/datasource_operator.go`:
    - **Schema inputs**:
      - `name` — `StringAttribute{Required: true}` — operator name
      - `seed` — `StringAttribute{Required: true, Sensitive: true}` — operator seed (validated as operator type via `nkeys.DecodeSeed`)
      - `signing_keys` — `ListAttribute{ElementType: types.StringType, Optional: true}` — additional signing key public keys
      - `account_server_url` — `StringAttribute{Optional: true}`
      - `operator_service_urls` — `ListAttribute{ElementType: types.StringType, Optional: true}`
      - `system_account` — `StringAttribute{Optional: true}` — public key of system account
      - `strict_signing_key_usage` — `BoolAttribute{Optional: true}` — default false
      - `tags` — `ListAttribute{ElementType: types.StringType, Optional: true}`
    - **Schema outputs** (Computed):
      - `public_key` — `StringAttribute{Computed: true}`
      - `jwt` — `StringAttribute{Computed: true}`
    - **Read logic**:
      1. Parse seed with `nkeys.FromSeed()`, validate it's operator type via `nkeys.DecodeSeed()`
      2. Get public key from keypair
      3. Create `jwt.NewOperatorClaims(publicKey)`
      4. Set all fields from inputs
      5. **Pin deterministic fields**: `claims.IssuedAt = 0`, `claims.ID = ""`
      6. Encode: `jwtString, err := claims.Encode(keypair)`
      7. Set `public_key` and `jwt` outputs
    - **Validators**: Custom validator on `seed` that checks it decodes as an operator seed

11. Create `internal/provider/datasource_operator_test.go`:
    - Test basic operator JWT generation
    - Test all optional fields are included in JWT
    - Test wrong seed type (account seed) is rejected
    - Test JWT is stable across multiple reads with same inputs
    - Test JWT output can be decoded and fields match inputs

### Phase 4: `natsjwt_account` Data Source

12. Create `internal/provider/datasource_account.go`:
    - **Schema inputs**:
      - `name` — `StringAttribute{Required: true}`
      - `seed` — `StringAttribute{Required: true, Sensitive: true}` — account seed (validated as account type)
      - `operator_seed` — `StringAttribute{Required: true, Sensitive: true}` — operator/signing key seed (validated as operator type)
      - `signing_keys` — `ListAttribute{ElementType: types.StringType, Optional: true}`
      - `description` — `StringAttribute{Optional: true}`
      - `info_url` — `StringAttribute{Optional: true}`
      - `tags` — `ListAttribute{ElementType: types.StringType, Optional: true}`
      - **NatsLimits block** (Optional, SingleNestedAttribute):
        - `subs` — `Int64Attribute{Optional: true}` — default -1 (unlimited)
        - `data` — `Int64Attribute{Optional: true}` — default -1
        - `payload` — `Int64Attribute{Optional: true}` — default -1
      - **AccountLimits block** (Optional, SingleNestedAttribute):
        - `imports` — `Int64Attribute{Optional: true}` — default -1
        - `exports` — `Int64Attribute{Optional: true}` — default -1
        - `wildcard_exports` — `BoolAttribute{Optional: true}` — default true
        - `disallow_bearer` — `BoolAttribute{Optional: true}` — default false
        - `conn` — `Int64Attribute{Optional: true}` — default -1
        - `leaf_node_conn` — `Int64Attribute{Optional: true}` — default -1
      - **JetStream limits** — `ListNestedAttribute{Optional: true}` where each block contains:
        - `tier` — `StringAttribute{Optional: true}` — empty/omitted = global limits; "R1"/"R3"/etc = tiered
        - `mem_storage` — `Int64Attribute{Optional: true}` — default 0 (disabled)
        - `disk_storage` — `Int64Attribute{Optional: true}` — default 0 (disabled)
        - `streams` — `Int64Attribute{Optional: true}` — default -1
        - `consumer` — `Int64Attribute{Optional: true}` — default -1
        - `max_ack_pending` — `Int64Attribute{Optional: true}` — default -1
        - `mem_max_stream_bytes` — `Int64Attribute{Optional: true}` — default 0
        - `disk_max_stream_bytes` — `Int64Attribute{Optional: true}` — default 0
        - `max_bytes_required` — `BoolAttribute{Optional: true}` — default false
      - **DefaultPermissions block** (Optional, SingleNestedAttribute):
        - `pub_allow` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `pub_deny` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `sub_allow` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `sub_deny` — `ListAttribute{ElementType: types.StringType, Optional: true}`
      - **Trace block** (Optional, SingleNestedAttribute):
        - `destination` — `StringAttribute{Optional: true}`
        - `sampling` — `Int64Attribute{Optional: true}` — 0-100
    - **Schema outputs** (Computed):
      - `public_key` — `StringAttribute{Computed: true}`
      - `jwt` — `StringAttribute{Computed: true}`
    - **Read logic**:
      1. Parse account seed, validate as account type
      2. Parse operator seed, validate as operator type
      3. Get account public key
      4. Create `jwt.NewAccountClaims(accountPubKey)`
      5. Map all input fields to claims struct
      6. Handle JetStream limits: iterate list blocks, block with no tier → global `JetStreamLimits`, blocks with tier → `JetStreamTieredLimits[tier]`
      7. Pin `IssuedAt = 0`, `ID = ""`
      8. Encode with operator keypair: `claims.Encode(operatorKP)`
      9. Output public_key and jwt

13. Create `internal/provider/datasource_account_test.go`:
    - Test basic account JWT
    - Test JetStream global limits
    - Test JetStream tiered limits (R1 + R3)
    - Test default permissions
    - Test seed type validation (reject user seed as account seed)
    - Test JWT stability

### Phase 5: `natsjwt_system_account` Data Source

14. Create `internal/provider/datasource_system_account.go`:
    - **Same schema as `natsjwt_account`** — reuse type definitions via shared helper functions
    - **Different defaults** to match what nsc uses when creating a system account:
      - `exports` default: includes `$SYS.>` public service export
      - Other defaults matching nsc's system account creation behavior
    - Internally, delegate to the same claim-building logic but with system-account-specific defaults applied before user overrides

15. Create `internal/provider/datasource_system_account_test.go`:
    - Test system account has correct default exports
    - Test defaults can be overridden
    - Test JWT includes system account characteristics

### Phase 6: `natsjwt_user` Data Source

16. Create `internal/provider/datasource_user.go`:
    - **Schema inputs**:
      - `name` — `StringAttribute{Required: true}`
      - `seed` — `StringAttribute{Required: true, Sensitive: true}` — user seed (validated as user type)
      - `account_seed` — `StringAttribute{Required: true, Sensitive: true}` — account/signing key seed (validated as account type)
      - `issuer_account` — `StringAttribute{Optional: true}` — set when using signing key instead of account key
      - **Permissions block** (Optional, SingleNestedAttribute):
        - `pub_allow` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `pub_deny` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `sub_allow` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `sub_deny` — `ListAttribute{ElementType: types.StringType, Optional: true}`
        - `resp_max_msgs` — `Int64Attribute{Optional: true}` — response permission max messages
        - `resp_ttl` — `StringAttribute{Optional: true}` — duration string
      - **Limits block** (Optional, SingleNestedAttribute):
        - `subs` — `Int64Attribute{Optional: true}` — default -1
        - `data` — `Int64Attribute{Optional: true}` — default -1
        - `payload` — `Int64Attribute{Optional: true}` — default -1
      - `bearer_token` — `BoolAttribute{Optional: true}` — default false
      - `allowed_connection_types` — `ListAttribute{ElementType: types.StringType, Optional: true}` — STANDARD, WEBSOCKET, LEAFNODE, MQTT, etc.
      - `source_networks` — `ListAttribute{ElementType: types.StringType, Optional: true}` — CIDR list
      - **Time restrictions** — `ListNestedAttribute{Optional: true}`:
        - `start` — `StringAttribute{Required: true}` — "HH:MM:SS"
        - `end` — `StringAttribute{Required: true}` — "HH:MM:SS"
      - `locale` — `StringAttribute{Optional: true}` — timezone for time restrictions
      - `tags` — `ListAttribute{ElementType: types.StringType, Optional: true}`
    - **Schema outputs** (Computed):
      - `public_key` — `StringAttribute{Computed: true}`
      - `jwt` — `StringAttribute{Computed: true}`
    - **Read logic**: Similar pattern — validate seeds, create `jwt.NewUserClaims()`, map fields, pin timestamps, encode with account keypair

17. Create `internal/provider/datasource_user_test.go`:
    - Test basic user JWT
    - Test permissions (pub/sub allow/deny)
    - Test connection type restrictions
    - Test time restrictions
    - Test source network CIDR validation
    - Test seed type validation

### Phase 7: `natsjwt_config_helper` Data Source

18. Create `internal/provider/datasource_config_helper.go`:
    - **Schema inputs**:
      - `operator_jwt` — `StringAttribute{Required: true}`
      - `account_jwts` — `ListAttribute{ElementType: types.StringType, Optional: true}`
      - `system_account_jwt` — `StringAttribute{Optional: true}`
      - `resolver_type` — `StringAttribute{Optional: true}` — default "MEMORY", validated to only accept "MEMORY" for now
    - **Schema outputs** (Computed):
      - `server_config` — `StringAttribute{Computed: true}` — complete config snippet
      - `operator` — `StringAttribute{Computed: true}` — just the `operator: <jwt>` line value
      - `system_account` — `StringAttribute{Computed: true}` — just the system account public key
      - `resolver` — `StringAttribute{Computed: true}` — "MEMORY"
      - `resolver_preload` — `MapAttribute{ElementType: types.StringType, Computed: true}` — `{pubkey: jwt}` map
    - **Read logic**:
      1. Decode operator JWT to extract operator claims (no seed needed — just parse)
      2. Decode system account JWT to get its public key
      3. Decode each account JWT to get its public key
      4. Build `resolver_preload` map: `accountPubKey → accountJWT` for all accounts including system account
      5. Build `server_config` string in NATS conf format:
         ```
         operator: <operator_jwt>
         system_account: <system_account_pubkey>
         resolver: MEMORY
         resolver_preload: {
           <pubkey1>: <jwt1>
           <pubkey2>: <jwt2>
         }
         ```
      6. Set individual outputs

19. Create `internal/provider/datasource_config_helper_test.go`:
    - Test full config generation with operator + system account + regular accounts
    - Test resolver_preload contains all accounts
    - Test server_config format is valid NATS config
    - Test individual outputs match components

### Phase 8: Shared Helpers & Validators

20. Create `internal/provider/validators.go`:
    - `seedTypeValidator(expectedType nkeys.PrefixByte)` — returns a `validator.String` that validates a seed decodes to the expected key type
    - `publicKeyTypeValidator(expectedType nkeys.PrefixByte)` — validates public key prefix
    - `connectionTypeValidator()` — validates allowed connection type strings

21. Create `internal/provider/helpers.go`:
    - `buildJetStreamLimits(blocks []JetStreamLimitsModel) (jwt.JetStreamLimits, jwt.JetStreamTieredLimits)` — converts TF model to JWT structs
    - `buildPermissions(model PermissionsModel) jwt.Permissions`
    - `encodeDeterministic(claims jwt.Claims, kp nkeys.KeyPair) (string, error)` — pins IssuedAt/ID before encoding

### Phase 9: Integration Tests

22. Create `internal/provider/integration_test.go`:
    - Full end-to-end test using `terraform-plugin-testing`:
      1. Create operator nkey → operator data source → account nkeys → account data sources → user nkeys → user data sources → config helper
      2. Verify the complete chain produces valid JWTs
      3. Verify config helper output is valid
      4. Verify JWT claims can be decoded and verified against issuer public keys
    - Test using external seed strings (simulating vault/secret manager input)
    - Test that system account data source produces correct defaults

### Phase 10: Documentation & README

23. Create `templates/` directory with doc templates for tfplugindocs
24. Run `tfplugindocs generate` to produce `docs/` from schemas
25. Create `README.md` with:
    - Provider description and purpose
    - Installation instructions (Terraform registry)
    - Quick start example showing full operator → account → user → config flow
    - Resource reference: `natsjwt_nkey`
    - Data source references: `natsjwt_operator`, `natsjwt_account`, `natsjwt_system_account`, `natsjwt_user`, `natsjwt_config_helper`
    - Notes on seed sensitivity and state management
    - Compatibility notes (NATS 2.11/2.12, Terraform 1.x)

26. Create `examples/full-setup/main.tf` — complete working example:
    ```hcl
    resource "natsjwt_nkey" "operator" { type = "operator" }
    data "natsjwt_operator" "main" {
      name = "my-operator"
      seed = natsjwt_nkey.operator.seed
      system_account = data.natsjwt_system_account.sys.public_key
    }
    # ... accounts, users, config_helper
    ```

### Phase 11: CI/CD

27. Create `.github/workflows/test.yml` — runs unit + acceptance tests on PRs
28. Create `.github/workflows/release.yml` — goreleaser-based release on tag push, publishes to Terraform Registry

## Project Structure

```
terraform-provider-natsjwt/
├── main.go
├── go.mod / go.sum
├── mise.toml
├── mise.toml (tools + tasks)
├── .goreleaser.yml
├── README.md
├── internal/provider/
│   ├── provider.go
│   ├── resource_nkey.go (+test)
│   ├── datasource_operator.go (+test)
│   ├── datasource_account.go (+test)
│   ├── datasource_system_account.go (+test)
│   ├── datasource_user.go (+test)
│   ├── datasource_config_helper.go (+test)
│   ├── validators.go
│   ├── helpers.go
│   └── integration_test.go
├── examples/
├── templates/
└── docs/ (generated)
```

## Verification

- `go test ./...` — all unit tests pass
- `mise run testacc` — acceptance tests using terraform-plugin-testing pass (creates real TF plans with the provider)
- Manual test: `terraform init && terraform plan && terraform apply` with example config, verify generated JWTs decode correctly with `nsc describe jwt`
- Verify `server_config` output from config_helper works with `nats-server -c <generated_config>`
- Verify JWT stability: run `terraform plan` twice with no input changes, confirm no diff

## Decisions

- **JWT determinism**: Pin `IssuedAt = 0` and `ID = ""` — JWTs are stable across plans when inputs are unchanged
- **Module path**: `github.com/m3nowak/terraform-provider-natsjwt`
- **JetStream limits**: Nested list block with optional `tier` key — no tier = global, tier = "R1"/"R3" for tiered limits
- **System account**: Separate `natsjwt_system_account` data source with system-account-appropriate defaults
- **Provider config**: None needed — all inputs are per-resource/data-source (no server connection)
- **Account expiry**: Not supported (non-goal per idea.md)
- **Resolver types**: Only MEMORY for now; schema includes `resolver_type` for future extensibility
