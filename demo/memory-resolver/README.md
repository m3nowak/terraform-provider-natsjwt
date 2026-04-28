# Demo

Demo showcasing generating operator, accounts, users and using them in local nats server running in `resolver: MEMORY` mode.  
To run it you will need [mise](https://mise.jdx.dev/) installed.

## Setup

```sh
# git clone ...
# cd to repo
mise trust .
mise trust ./demo/memory-resolver

mise install
```

## Running the demo

To start the server:

```sh
cd demo/memory-resolver
mise r run-server
```

Then, while the server is running:

```sh
cd demo/memory-resolver

nats sub hello # subscribe to hello subject

#in other terminal - cd to demo/memory-resolver folder
nats pub hello world # publish to hello subject

# By default, nats here uses app-user.creds creds, which have all permissions
# We can change it to app-user2.creds, which is limmited to `app` prefix to check if the server properly restricts permissions
nats pub hello world --creds ./app-user2.creds # should fail
```

## What's going on

File main.tf in demo/memory-resolver contains all resources and data objects needed to generate server configuration:

- nkeys and jwts for:
    - operator
    - system account
    - application account
- nats_config local file, using natsjwt_config_helper for generating part of server config

Also following objects are created, in order to generate users to communicate with running server:

- for each of 2 users:
    - nkey
    - user jwt
    - local file containing creds

