# Demo

Simple demo folder. To use it you will need [mise](https://mise.jdx.dev/) installed.

## Setup

```sh
# git clone ...
# cd to repo
mise trust .
mise trust ./demo

mise install
```

## Running the demo

To start the server:

```sh
cd demo
mise r run-server
```

Then, while the server is running:

```sh
cd demo

nats sub hello # subscibe to hello subject

#in other terminal - cd to demo folder
nats pub hello world # publish to hello subject

# By default, nats here uses app-user.creds creds, which have all permissions
# We can change it to app-user2.creds, which is limmited to `app` prefix to check if the server properly restricts permissions
nats pub hello world --creds ./app-user2.creds # should fail
```
