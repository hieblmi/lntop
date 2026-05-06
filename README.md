# lntop

[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/hieblmi/lntop/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hieblmi/lntop)](https://goreportcard.com/report/github.com/hieblmi/lntop)
[![Godoc](https://godoc.org/github.com/hieblmi/lntop?status.svg)](https://godoc.org/github.com/hieblmi/lntop)

`lntop` is an interactive terminal dashboard for Lightning Network nodes
running [LND](https://github.com/lightningnetwork/lnd). It gives you a
live, keyboard-driven view of channels, balances, routing activity,
forwarding history, on-chain transactions, received invoices, and outgoing
payments.

The UI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Preview

The Bubble Tea UI preview from
[PR #91](https://github.com/hieblmi/lntop/pull/91#issue-4033441681):

<video src="https://github.com/user-attachments/assets/937bdd3f-b8cc-42e8-8e42-e3bbff897707" controls width="100%"></video>

[Open the preview video](https://github.com/user-attachments/assets/937bdd3f-b8cc-42e8-8e42-e3bbff897707)

## Features

- Full-screen terminal UI with a header, summary panels, menu, tables, detail
  screens, and a startup progress view.
- LND node summary: alias, LND version, chain/network, sync state, block
  height, and peer count.
- Channel summary: channel balance, pending balance, active/pending/inactive
  counts, disabled local/remote policy counts, and local-liquidity gauge.
- Wallet summary: total, confirmed, unconfirmed, locked, and reserved anchor
  channel balances.
- Accounting summary based on the currently configured forwarding-history
  window: profit, total forwarded, largest/smallest forward, most profitable
  forward, and hottest forwarding link.
- Channels table with configurable columns, sortable columns, horizontal
  column navigation, pending-channel support, channel detail view, policy
  details, disabled-policy detection, channel-age display, and optional
  local alias overrides.
- Live activity highlights in the channels table for pending HTLCs, unsettled
  balances, sent traffic, and received traffic.
- Transactions table with transaction detail view.
- Routing table fed from LND router HTLC events.
- Forwarding history table with configurable time window and maximum event
  count.
- Received table for settled invoices and keysend receives, with optional
  start-date filtering.
- Payments table for outgoing payments, including status, fees, failures,
  route summaries, route hops, and HTLC attempt details.
- Event-driven refreshes from LND subscriptions plus a 3-second polling ticker
  for aggregate balances and per-channel state changes.
- Runtime settings modal for forwarding history and received-invoice filters.
- Docker packaging scripts in [`docker/`](docker/README.md).

## Install

This branch's `go.mod` declares Go `1.25.5`. Use a Go toolchain compatible
with that version.

Raspberry Pi users: be aware that Raspbian often ships an old Go toolchain
(for example Go 1.11 in the original report). Install a modern Go release
manually if the distro package is too old. See
[#30](https://github.com/hieblmi/lntop/issues/30).

Build from source:

```sh
git clone https://github.com/hieblmi/lntop.git
cd lntop
go build -o lntop .
go install .
```

With Go modules, you can install the latest published version directly:

```sh
go install github.com/hieblmi/lntop@latest
```

Umbrel and Citadel users can install the
[Lightning Shell](https://lightningshell.app) app from the dashboard. It
includes `lntop` and should work without extra configuration.

Docker users can run `lntop` from a container. See
[`docker/README.md`](docker/README.md).

## Quick Start

Run `lntop` once:

```sh
lntop
```

On the first start, `lntop` creates `~/.lntop/config.toml` and
`~/.lntop/lntop.log`. Edit the config so it points at your LND gRPC address,
TLS certificate, and macaroon:

```toml
[network]
address = "127.0.0.1:10009"
cert = "/home/user/.lnd/tls.cert"
macaroon = "/home/user/.lnd/data/chain/bitcoin/mainnet/readonly.macaroon"
```

Then start it again:

```sh
lntop
```

Use a custom config file with:

```sh
lntop --config /path/to/config.toml
lntop -c /path/to/config.toml
```

Print the version:

```sh
lntop --version
```

Run only the event subscriber loop:

```sh
lntop pubsub
```

## Initial Config Environment Variables

These environment variables are only used when the initial config file is
created. They do not override an existing `~/.lntop/config.toml`.

| Variable | Config field |
| --- | --- |
| `LND_ADDRESS` | `network.address` |
| `CERT_PATH` | `network.cert` |
| `MACAROON_PATH` | `network.macaroon` |

Example:

```sh
LND_ADDRESS=127.0.0.1:10009 \
CERT_PATH=$HOME/.lnd/tls.cert \
MACAROON_PATH=$HOME/.lnd/data/chain/bitcoin/mainnet/readonly.macaroon \
lntop
```

## Controls

| Key | Action |
| --- | --- |
| `F2`, `m` | Open or close the view menu |
| `Up`, `Down`, `k`, `j` | Move between rows or menu items |
| `Left`, `Right`, `h`, `l` | Move between table columns |
| `Home`, `g` | Jump to the first row |
| `End`, `G` | Jump to the last row |
| `PageUp`, `PageDown` | Move by one page |
| `a` | Sort the active column ascending |
| `d` | Sort the active column descending |
| `Enter` | Open/close details for channels, transactions, and payments |
| `c` | On channels, load disabled-channel counts for the selected peer |
| `F9` | Open runtime data settings |
| `/`, `w` | Open runtime settings from the forwarding-history view |
| `Esc` | Cancel the runtime settings modal |
| `F10`, `q`, `Ctrl-C` | Quit |

Runtime settings are applied to the current session only. Persist them in
`config.toml` if you want them to survive restarts.

## Views

### Channels

The channels view is the default view. It lists open and pending channels and
can show status, peer alias, local/remote balance, capacity, total sent and
received, pending HTLC count, unsettled amount, commit fee, last update,
approximate age, privacy, channel id, channel point, short channel id, update
count, outgoing fees, incoming fees, and LND inbound fees.

Press `Enter` on a channel to open details. The detail view shows channel
state, capacity and balances, channel point, peer pubkey, peer alias, peer
capacity and channel count, outgoing and incoming policies, disabled-policy
state, and pending HTLCs. Press `c` to fetch disabled-channel counts for the
selected peer.

Configure it with `[views.channels]`:

```toml
[views.channels]
columns = [
  "STATUS", "ALIAS", "GAUGE", "LOCAL", "REMOTE", "CAP",
  "SENT", "RECEIVED", "HTLC", "UNSETTLED", "CFEE",
  "LAST UPDATE", "AGE", "PRIVATE", "ID", "CHANNEL_POINT",
]

[views.channels.options]
AGE = { color = "color" }
```

`AGE.color = "color"` enables colored channel-age output in terminals with
256-color support.

### Transactions

The transactions view lists on-chain wallet transactions. Press `Enter` to
open details including date, amount, fee, block height, confirmations, block
hash, transaction hash, and destination addresses.

Configure it with `[views.transactions]`:

```toml
[views.transactions]
columns = ["DATE", "HEIGHT", "CONFIR", "AMOUNT", "FEE", "ADDRESSES"]
```

### Routing

The routing view displays live HTLC routing events from LND. Events can be:

- `active`: HTLC pending
- `settled`: preimage revealed and HTLC removed
- `failed`: payment failed at a downstream node
- `linkfail`: payment failed at this node

Routing events are not persisted by LND for this view. The view starts empty
when `lntop` starts and the in-memory routing log is lost when you exit.

Configure it with `[views.routing]`:

```toml
[views.routing]
columns = [
  "DIR", "STATUS", "IN_CHANNEL", "IN_ALIAS",
  "OUT_CHANNEL", "OUT_ALIAS", "AMOUNT", "FEE",
  "LAST UPDATE", "DETAIL",
]
```

### Forwarding History

The forwarding-history view displays historical forwarding events from LND's
`ForwardingHistory` RPC. The summary accounting panel uses the same filtered
event set, so changing the forwarding window changes both the table and the
summary.

Configure it with `[views.fwdinghist]`:

```toml
[views.fwdinghist]
columns = ["ALIAS_IN", "ALIAS_OUT", "AMT_IN", "AMT_OUT", "FEE", "TIMESTAMP_NS"]

[views.fwdinghist.options]
START_TIME = { start_time = "-12h" }
MAX_NUM_EVENTS = { max_num_events = "333" }
```

`START_TIME` accepts a Unix timestamp, an empty value for all history, or a
negative range ending in `s`, `m`, `h`, `d`, `w`, `M`, or `y`, for example
`-30m`, `-12h`, `-7d`, or `-1M`.

`MAX_NUM_EVENTS` limits how many events LND returns. `0` means all events.
Higher values can noticeably increase loading time because `lntop` enriches
forwarding events with peer aliases.

### Received

The received view lists settled invoices and keysend receives. It shows the
receive type, settle/creation time, amount, memo, and payment hash.

Configure it with `[views.received]`:

```toml
[views.received]
columns = ["TYPE", "TIME", "AMOUNT", "MEMO", "R_HASH"]

[views.received.options]
START_DATE = { start_date = "2025-09-01" }
```

`START_DATE` is optional. When set, it must be `YYYY-MM-DD` in local time.
Invoices settled before that local date are hidden. Leave it blank or omit it
to show all settled invoices.

### Payments

The payments view lists outgoing payments. It supports payment type, creation
time, status, amount, fee, HTLC attempts, failure reason, index, payment hash,
preimage, and payment request. Press `Enter` to open payment details,
including decoded invoice metadata, route summary, route hops, and attempt
failure details.

Configure it with `[views.payments]`:

```toml
[views.payments]
columns = [
  "TYPE", "TIME", "STATUS", "AMOUNT", "AMOUNT_MSAT",
  "FEE", "FEE_MSAT", "ATTEMPTS", "FAILURE",
  "INDEX", "HASH", "PREIMAGE", "REQUEST",
]
```

## Configuration Reference

`lntop` uses TOML. If `--config` is not supplied, it reads
`~/.lntop/config.toml`.

### Logger

```toml
[logger]
type = "production"
dest = "/home/user/.lntop/lntop.log"
```

| Field | Description |
| --- | --- |
| `type` | `production` or `development`; unknown values fall back to development logging |
| `dest` | Log output path |

### Network

```toml
[network]
name = "lnd"
type = "lnd"
address = "127.0.0.1:10009"
cert = "/home/user/.lnd/tls.cert"
macaroon = "/home/user/.lnd/data/chain/bitcoin/mainnet/readonly.macaroon"
macaroon_timeout = 60
macaroon_ip = ""
max_msg_recv_size = 52428800
conn_timeout = 1000000
pool_capacity = 6
```

| Field | Description |
| --- | --- |
| `name` | Human label for the backend |
| `type` | `lnd` for real LND, `mock` for development/testing |
| `address` | LND gRPC address in `host:port` format. Legacy `//host:port` values are accepted |
| `cert` | Path to LND `tls.cert`; if empty, TLS is still used without loading a cert file |
| `macaroon` | Path to the macaroon. The generated config uses `readonly.macaroon` |
| `macaroon_timeout` | Timeout constraint added to the macaroon, in seconds |
| `macaroon_ip` | Optional IP-lock constraint for the macaroon |
| `max_msg_recv_size` | Maximum gRPC receive message size, in bytes |
| `conn_timeout` | gRPC connection-pool reuse timeout as a Go duration value in nanoseconds |
| `pool_capacity` | gRPC connection-pool size. The LND backend enforces a minimum of 6 |

### Alias Overrides

Not all peers publish useful aliases. You can annotate pubkeys yourself:

```toml
[network.aliases]
035e4ff418fc8b5554c5d9eea66396c227bd429a3251c8cbc711002ba215bfc226 = "Wallet of Satoshi"
03864ef025fde8fb587d989186ce6a4a186895ee44a926bfc370e2c366597a3f8f = "-=[ACINQ]=-"
```

Forced aliases are displayed in a different color so they can be distinguished
from aliases advertised by the network.

### View Columns

Each table view has a `columns` array. Add, remove, or reorder columns by
editing the matching `[views.<name>]` section. Column names are case-sensitive.

| View | Supported columns |
| --- | --- |
| `channels` | `STATUS`, `ALIAS`, `GAUGE`, `LOCAL`, `REMOTE`, `CAP`, `SENT`, `RECEIVED`, `HTLC`, `UNSETTLED`, `CFEE`, `LAST UPDATE`, `AGE`, `PRIVATE`, `ID`, `CHANNEL_POINT`, `SCID`, `NUPD`, `BASE_OUT`, `RATE_OUT`, `BASE_IN`, `RATE_IN`, `INBOUND_BASE`, `INBOUND_RATE` |
| `transactions` | `DATE`, `HEIGHT`, `CONFIR`, `AMOUNT`, `FEE`, `ADDRESSES`, `TXHASH`, `BLOCKHASH` |
| `routing` | `DIR`, `STATUS`, `IN_CHANNEL`, `IN_ALIAS`, `IN_SCID`, `IN_HTLC`, `IN_TIMELOCK`, `OUT_CHANNEL`, `OUT_ALIAS`, `OUT_SCID`, `OUT_HTLC`, `OUT_TIMELOCK`, `AMOUNT`, `FEE`, `LAST UPDATE`, `DETAIL`, `INBOUND_BASE_IN`, `INBOUND_RATE_IN` |
| `fwdinghist` | `ALIAS_IN`, `ALIAS_OUT`, `AMT_IN`, `AMT_OUT`, `FEE`, `TIMESTAMP_NS`, `CHAN_ID_IN`, `CHAN_ID_OUT`, `INBOUND_BASE_IN`, `INBOUND_RATE_IN` |
| `received` | `TYPE`, `TIME`, `AMOUNT`, `MEMO`, `R_HASH` |
| `payments` | `TYPE`, `TIME`, `STATUS`, `AMOUNT`, `AMOUNT_MSAT`, `FEE`, `FEE_MSAT`, `ATTEMPTS`, `FAILURE`, `INDEX`, `HASH`, `PREIMAGE`, `REQUEST` |

Inbound fee columns require LND versions that expose inbound fee fields
(LND 0.18 and newer).

## Docker

If you prefer to run `lntop` from a Docker container, `cd docker` and follow
the instructions in [`docker/README.md`](docker/README.md).
