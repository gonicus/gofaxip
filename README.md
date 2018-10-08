# GOfax.IP

 [![CircleCI](https://circleci.com/gh/gonicus/gofaxip/tree/master.svg?style=shield)](https://circleci.com/gh/gonicus/gofaxip/tree/master) 

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

## Installation

We recommend running GOfax.IP on Debian 8 ("Jessie"), so these instructions cover Debian in detail. Of course it is possible to install and use GOfax.IP on other Linux distributions and possibly other Unixes supported by golang, FreeSWITCH and HylaFAX.

### Dependencies

The official FreeSWITCH Debian repository can be used to obtain and install all required FreeSWITCH packages.

Adding the repository:

```
curl http://files.freeswitch.org/repo/deb/debian/freeswitch_archive_g0.pub | apt-key add -
echo 'deb http://files.freeswitch.org/repo/deb/freeswitch-1.6/ jessie main' >> /etc/apt/sources.list.d/freeswitch.list
```

### Installing packages

```
apt-get update
apt-get install hylafax-server freeswitch freeswitch-mod-commands freeswitch-mod-dptools freeswitch-mod-event-socket freeswitch-mod-sofia freeswitch-mod-spandsp freeswitch-mod-tone-stream freeswitch-mod-db freeswitch-mod-syslog freeswitch-mod-logfile
```

### GOfax.IP

See [releases](https://github.com/gonicus/gofaxip/releases) for amd64 Debian packages.

Use ```dpkg -i``` to install the latest package.

See [Building](#building) for instructions on how to build the binaries or Debian packages from source.
This is only necessary if you want to use the latest/untested version or if you need another architecture than amd64!

## Configuration

### FreeSWITCH

FreeSWITCH has to be able to place received faxes in HylaFAX' `recvq` spool. The simplest way to achieve this is to run FreeSWITCH as the `uucp` user. 

```
sudo chown -R uucp.uucp /var/log/freeswitch
sudo chown -R uucp.uucp /var/lib/freeswitch
sudo cp /usr/share/doc/gofaxip/examples/freeswitch.service /etc/systemd/system/
sudo systemctl daemon-reload
```

A very minimal FreeSWITCH configuration for GOfax.IP is provided in the repository.

```
sudo cp -r /usr/share/doc/gofaxip/examples/freeswitch/* /etc/freeswitch/
```

The SIP gateway to use has to be configured in `/etc/freeswitch/gateways/default.xml`. It is possible to configure multiple gateways for GOfax.IP.

**Depending on your installation/SIP provider you have to either:**
 * edit `/etc/freeswitch/vars.xml` and adapt the IP address FreeSWITCH should use for SIP/RTP.
 * remove the entire section concerning the `sofia_ip` variable (parameters `sip-port, rtp-ip, sip-ip, ext-rtp-ip, ext-sip-ip`)

### GOfax.IP

Currently GOfax.IP does not use HylaFAX configuration files *at all*. All configurations for both `gofaxd` and `gofaxsend` are made in the INI-style configuration file `/etc/gofax.conf` which has to be customized.

### HylaFAX

To make HylaFAX use `gofaxsend` for sending, the `SendFaxCmd` option has to be added to `/etc/hylafax/config`:

```
SendFaxCmd:		"/usr/bin/gofaxsend"
```

A sample `FaxDispatch` script is included in `/usr/share/doc/gofaxip/examples/hylafax/FaxDispatch`, the available `CALLID` values set by `gofaxd` are documented there.

To have `faxstat` show modem/channel usage in it's status output, a modem configuration file has to exist. Note that GOfax.IP currently does not use HylaFAX' modem configuration files, so they can be empty, but they have to exist for `faxstat` to show the modem.

If `/etc/gofax.conf` is configured to manage 5 (virtual) modems, you have to create the (empty) configuration files manually:

```
sudo touch /var/spool/hylafax/etc/config.freeswitch{0..4}
```

## Operation

### Starting

```
sudo systemctl start freeswitch
sudo systemctl start gofaxip
sudo systemctl restart hylafax
```

### Logging 

GOfax.IP logs everything it does to syslog. 

## Advanced Features

As the _virtual modems_ visible in HylaFAX are not tied to preconfigured lines but assigned dynamically, it is not possible to assign static telephone numbers to individual modems. Instead, GOfax.IP can query a `DynamicConfig` script before trying to send outgoing faxes which works similarly to the `DynamicConfig` feature in HylaFAX' `faxgetty`. Using the sender's user id (`owner`), it can be used to set the Callerid, TSI and Header for each individual outgoing fax. It is also possible to reject an outgoing fax.

### DynamicConfig for incoming faxes

`DynamicConfig` as used and documented in HylaFAX can be used by GOfax.IP. 
One option per line can be set, comments are note allowed. 

**Parameters**
The following arguments are provided to the `DynamicConfig` script for incoming faxes:

* Used modem name
* Caller-ID number
* Caller-ID name
* Destination (SIP To-User)
* Gateway name

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
* Job ID

**Supported options**
* `RejectCall: true` will reject the outgoing fax. The fax will instantly fail and not be retried.
* `LocalIdentifier: +1 234 567` will assign a TSI (Transmitting Station Identifier) for this call. The Default TSI can be set in `gofax.conf` in the `ident` parameter.
* `TagLine: ACME` will assign a header string to be shown on the top of each page. This does not support format strings as used by HylaFAX; if defined a header string is always shown with the current timestamp and page number as set by SpanDSP.
* `FAXNumber: 1337` will set the outgoing caller id number as used by FreeSWITCH when originating the call. 
* `Gateway: somegw` or `Gateway: gw1,gw2` will set the [SIP Gateway](https://freeswitch.org/confluence/display/FREESWITCH/Gateways+Configuration) to use for sending the fax. The gateway has to be configured in FreeSWITCH. When multiple comma delimited gateways are given they will be tried in order. By default the gateway configured in GOFax.IP's configuration file is used.

### Fallback from T.38 to SpanDSP softmodem

In rare cases we noticed problems with certain remote stations that could not successfully work with some T.38 Gateways we tested. In the case we observed, the remote tried to use T.4 1-D compression with ECM enabled. After disabling T.38 the fax was successfully received. 

To work around this rare sort of problem and improve compatiblity, GOfax.IP can identify failed transmissions and dynamically disable T.38 for the affected remote station and have FreeSWITCH use SpanDSP's pure software fax implementation. The station is identified by caller id and saved in FreeSWITCH's `mod_db`.
To enable this feature, set `softmodemfallback = true` in `gofax.conf`.

Note that this will only affect all subsequent calls from/to the remote station, assuming that the remote station will retry a failed fax. Entries in the fallback list are persistant and will not be expired by GOfax.IP. It is possible however to manually expire entries from mod_db. The used `<realm>/<key>/<value>` is `fallback/<callerid>/<unix_timestamp>`, with unix_timestamp being the time when the entry was added. See https://wiki.freeswitch.org/wiki/Mod_db for details on mod_db. 

A transmission is regarded as failed and added to the fallback database if SpanDSP reports the transmission as not successful and one of the following conditions apply:

* Negotiation has happened multiple times
* Negotiation was successful but transmitted pages contain bad rows

### Setting the Displayname for outgoing faxes

Normally the Displayname is populated with the content of the `sender` field from the qfile.
If you dont want to expose this information you can use the parameter `cidname` in `gofax.conf` to set the Displayname to the Calleridnum or any static string.

# Building

GOfax.IP is implemented in [Go](https://golang.org/doc/install), it can be built using `go get`.

```
go get github.com/gonicus/gofaxip/...
```

This will produce the binaries `gofaxd` and `gofaxsend`.

## Build debian package

You need golang and dh-golang from jessie-backports.

With golang package from debian repository:
```
echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/jessie-backports.list
apt update
apt install dh-golang dh-systemd golang -t jessie-backports
git clone https://github.com/gonicus/gofaxip
cd gofaxip
dpkg-buildpackage -us -uc -rfakeroot -b
```

If you dont have golang from debian repository installed use ```-d``` to ignore builddeps.
