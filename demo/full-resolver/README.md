# Demo

Demo showcasing generating operator, accounts, users and using them in local nats server running ["full" resolver](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt/resolver#full).  
To run it you will need [mise](https://mise.jdx.dev/) installed.
## Setup

```sh
# git clone ...
# cd to repo
mise trust .
mise trust ./demo/full-resolver

mise install
```

## Demo is split into 2 stages
First stage generates minimal config for nats-server.  
Second stage uploads new account to the running nats-server.


To run the stage one:

```sh
cd demo/full-resolver

# this performs apply for stage one and starts nats server
mise r run-server
```

Then, while the server is running, in other terminal, run stage two:

```sh
cd demo/full-resolver

# this performs apply for stage two - apply of terraform module in stage2 folder
mise r tf-apply-s2


nats sub hello # subscibe to hello subject

#in other terminal - cd to demo/full-resolver folder
nats pub hello world # publish to hello subject

# By default, nats here uses app-user.creds creds, which have all permissions
# We can change it to app-user2.creds, which is limmited to `app` prefix to check if the server properly restricts permissions
nats pub hello world --creds ./app-user2.creds # should fail
```

## What's going on

The demo is split into two stages because the NATS server must already be running before account JWTs can be pushed to it via the NATS-based resolver.

### Stage 1 (`stage1/server.tf`)

Generates the minimal configuration needed to start the NATS server:

- nkeys and JWTs for:
    - operator
    - system account
    - sys user (with full permissions, used to authenticate with the server)
- `natsjwt_config_helper` for generating the `operator` and `system_account` part of server config
- local files for:
    - `sys-user.creds` — credentials used to authenticate stage 2 with the running server
    - `operator.nk` — operator seed, needed to sign the application account JWT in stage 2
    - `nats-server.conf` — full server configuration including a `resolver: { type: full }` block

### Stage 2 (`stage2/jwt.tf`)

Connects to the already-running NATS server and pushes account JWTs:

- configures the provider with `nats_url` and `creds` (sys-user credentials from stage 1)
- nkeys and JWTs for:
    - application account (with JetStream limits)
    - 2 application users with different permissions
- `natsjwt_resolver_account` resource to push the application account JWT to the running server
- local files for:
    - `app-user.creds` and `app-user2.creds` — user credentials to test connections
