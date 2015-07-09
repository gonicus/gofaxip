# GOfax.IP

GOfax.IP is a HylaFAX backend/connector providing Fax over IP support for HylaFAX using FreeSWITCH and SpanDSP through FreeSWITCH's mod_spandsp.

In contrast to solutions like t38modem, iaxmodem and mod_spandsp's softmodem feature, GOfax.IP does not emulate fax modem devices but replaces HylaFAX's `faxgetty` and `faxsend` processes to communicate directly with FreeSWITCH using FreeSWITCH's Event Socket interface.

GOfax.IP is designed to provide a standalone fax server together with HylaFAX and a minimal FreeSWITCH setup; the necessary FreeSWITCH configuration is provided.

## Features

* SIP connectivity to PBXes, Media Gateways/SBCs and SIP Providers with or without registration
* Failover using multiple gateways
* Support for Fax over IP using T.38 **and/or** T.30 audio over G.711
* Native SIP endpoint, no modem emulation
* Support for an arbitrary number of lines (depending on the used hardware)
* Extensive logging and reporting: Writing `xferfaxlog` for sent/received faxes; Writing session log files for all sent/received faxes
* Support for modem status reporting/querying using HylaFAX native tools and clients: `faxstat` etc.
* Call screening using HylaFAX' `DynamicConfig`

## Components

GOfax.IP consists of two commands that replace their native HylaFAX conterparts
* `gofaxsend` is used instead of HylaFAX' `faxsend `
* `gofaxd` is used instead of HylaFAX' `faxgetty`. Only one instance of `gofaxd` is necessary regardless of the number of receiving channels. 

## Building

GOfax.IP is implemented in Go/golang. It was developed and tested with Go version 1.2.1. As Go binaries are (almost) statically linked, they can be built on any system and just placed on the target system.  

### Using Makefile

A simple `Makefile` is included that will set up `GOPATH` and embed the current HEAD's short commit hash into the version printed when running `gofaxd -version` or `gofaxsend -version`. Provided that the Go compiler is installed and the `go` command is executable, build by just typing `make` in the repository's root. 

### Building manually

Alternatively, GOfax.IP can be built manually using `go`:

```
export GOPATH=$(pwd)
go install gofaxd gofaxsend
```

The resulting binaries `gofaxd` and `gofaxsend` are located in the `bin` directory. 

## Installation

We recommend running GOfax.IP on Debian 8 ("Jessie"), so these instructions cover Debian in detail. Of course it is possible to install and use GOfax.IP on other Linux distributions and possibly other Unixes supported by golang, FreeSWITCH and HylaFAX.

### Dependencies

The official FreeSWITCH Debian repository can be used to obtain and install all required FreeSWITCH packages.

Adding the repository:

```
echo 'deb http://files.freeswitch.org/repo/deb/debian/ jessie main' >> /etc/apt/sources.list.d/freeswitch.list
curl http://files.freeswitch.org/repo/deb/debian/freeswitch_archive_g0.pub | apt-key add -
```

### Installing packages

```
apt-get update
apt-get install hylafax-server freeswitch freeswitch-mod-commands freeswitch-mod-dptools freeswitch-mod-event-socket freeswitch-mod-logfile freeswitch-mod-sofia freeswitch-mod-spandsp freeswitch-mod-timerfd freeswitch-mod-tone-stream freeswitch-mod-db freeswitch-sysvinit
```

It is recommended to run gofaxd using a process management system as it doesn't daemonize by itself. A configuration file for the supervisor process control system is provided in the GOfax.IP repository. To use it, the supervisor packae has to be installed:

```
sudo apt-get install supervisor
sudo cp config/supervisor/conf.d/gofaxd.conf /etc/supervisor/conf.d/
```

### GOfax.IP

We do not provide packages (yet), so for now the two binaries have to be installed manually. 

```
sudo cp bin/* /usr/local/sbin/
```

## Configuration

### FreeSWITCH

FreeSWITCH has to be able to place received faxes in HylaFAX' `recvq` spool. The simplest way to achieve this is to run FreeSWICH as the `uucp` user. 

```
sudo chown -R uucp.uucp /var/log/freeswitch
sudo chown -R uucp.uucp /var/lib/freeswitch
sudo cp config/default/freeswitch /etc/default/
```

A very minimal FreeSWITCH configuration for GOfax.IP is provided in the repository.

```
sudo cp -r config/freeswitch /etc/freeswitch/
```

The SIP gateway to use has to be configured in `/etc/freeswitch/gateways/default.xml`. It is possible to configure multiple gateways for GOfax.IP.

**You have to edit `/etc/freeswitch/vars.xml` and adapt the IP address FreeSWITCH should use for SIP/RTP.**

### GOfax.IP

Currently GOfax.IP does not use HylaFAX configuration files *at all*. All configuration for both `gofaxd` and `gofaxsend` is done in the INI-style configuration file `/etc/gofax.conf`. A sample config file is provided and has to be customized.

```
sudo cp config/gofax.conf /etc/
```

### HylaFAX

To make HylaFAX use `gofaxsend` for sending, the `SendFaxCmd` option has to be added to `/etc/hylafax/config`:

```
SendFaxCmd:		"/usr/local/sbin/gofaxsend"
```

A sample `FaxDispatch` script is included in `config/hylafax/FaxDispatch`, the available `CALLID` values set by `gofaxd` are documented there. 

To have `faxstat` show modem/channel usage in it's status output, a modem configuration file has to exist. Note that GOfax.IP currently does not use HylaFAX' modem configuration files, so they can be empty, but they have to exist for `faxstat` to show the modem.

If `/etc/gofax.conf` is configured to manage 5 (virtual) modems, you have to create the (empty) configuration files manually:

```
sudo touch /var/spool/hylafax/etc/config.freeswitch{0..4}
```

## Operation

### Starting

```
sudo /etc/init.d/freeswitch start
sudo /etc/init.d/supervisor restart
sudo /etc/init.d/hylafax restart
```

### Logging 

GOfax.IP logs everything it does to syslog. 

## Advanced Features

As the _virtual modems_ visible in HylaFAX are not tied to preconfigured lines but assigned dynamically, it is not feasible to assign static telephone numbers to individual modems. Instead, GOfax.IP can query a `DynamicConfig` script before trying to send outgoing faxes that works similar to the `DynamicConfig` feature in HylaFAX' `faxgetty`. Using the sender's user id (`owner`), it can be used to set the Callerid, TSI and Header for each individual outgoing fax. It is also possible to reject an outgoing fax.

### DynamicConfig for incoming faxes

`DynamicConfig` as used and documented in HylaFAX can be used by GOfax.IP. 
One option per line can be set, comments are note allowed. 

**Parameters**
The following arguments are provided to the `DynamicConfig` script for incoming faxes:

* Used modem name
* Caller-ID number
* Caller-ID name
* Destination number (SIP To-User)

**Supported options**
* `RejectCall: true` will reject the call. Default is to allow the call
* `LocalIdentifier: +1 234 567` will assign a CSI (Called Station Identifier) that will be used for this fax reception. The Default CSI can be set in `gofax.conf` in the `ident` parameter.

### DynamicConfig for outgoing faxes

**This is a special feature of GOfax.IP, a similar mechanism does not exist in traditional HylaFAX installations.**

**Parameters**
The following arguments are provided to the `DynamicConfig` script for outgoing faxes:

* Used modem name
* Owner (User ID as set `sendfax -o` or the `FAXUSER` environment variable, optinally verified by PAM)
* Destination number

**Supported options**
* `RejectCall: true` will reject the outgoing fax. The fax will instantly fail and not be retried.
* `LocalIdentifier: +1 234 567` will assign a TSI (Transmitting Station Identifier) for this call. The Default TSI can be set in `gofax.conf` in the `ident` parameter.
* `TagLine: ACME` will assign a header string to be shown on the top of each page. This does not support format strings as used by HylaFAX; if defined a header string is always shown with the current timestamp and page number as set by SpanDSP.
* `FAXNumber: 1337` will set the outgoing caller id number as used by FreeSWITCH when originating the call. 

### Fallback from T.38 to SpanDSP softmodem

In rare cases we noticed problems with certain remote stations that could not successfully negotiate with some T.38 Gateways we tested. In the case we observed, the remote tried to use T.4 1-D compression with ECM enabled. After disabling T.38 the fax was successfully received. 

To work around this rare sort of problem and improve compatiblity, GOfax.IP can identify failed transmissions and dynamically disable T.38 for the affected remote station and have FreeSWITCH use SpanDSP's pure software fax implementation. The station is identified by caller id and saved in FreeSWITCH's `mod_db`.
To enable this feature, set `softmodemfallback = true` in `gofax.conf`.

Note that this will only affect all subsequent calls from/to the remote station, assuming that the remote station will retry a failed fax. Entries in the fallback list are persistant and will not be expired by GOfax.IP. It is possible however to manually expire entries from mod_db. The used `<realm>/<key>/<value>` is `fallback/<callerid>/<unix_timestamp>`, with unix_timestamp being the time when the entry was added. See https://wiki.freeswitch.org/wiki/Mod_db for details on mod_db. 

A transmission is regarded as failed and added to the fallback database if SpanDSP reports the transmission as not successful and one of the following conditions apply:

* Negotiation has happened multiple times
* Negotiation was successful but transmitted pages contain bad rows
