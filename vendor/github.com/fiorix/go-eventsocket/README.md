# eventsocket

FreeSWITCH [Event Socket](http://wiki.freeswitch.org/wiki/Event_Socket) library
for the [Go programming language](http://golang.org).

It supports both inbound and outbound event socket connections, acting either
as a client connecting to FreeSWITCH or as a server accepting connections
from FreeSWITCH to control calls.

This code has not been tested in production and is considered alpha. Use at
your own risk.

## Installing

Make sure $GOPATH is set, and use the following command to install:

	go get github.com/fiorix/go-eventsocket/eventsocket

The library is currently a single file, so feel free to drop into any project
without bothering to install.

## Usage

There are simple and clear examples of usage under the *examples* directory. A
client that connects to FreeSWITCH and originate a call, pointing to an
Event Socket server, which answers the call and instructs FreeSWITCH to play
an audio file.
