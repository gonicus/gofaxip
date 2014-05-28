package gofaxlib

import (
	"fmt"
	"github.com/fiorix/go-eventsocket/eventsocket"
	"time"
)

const (
	MOD_DB_FALLBACK_REALM = "fallback"
)

func GetSoftmodemFallback(c *eventsocket.Connection, cidnum string) (bool, error) {
	if !Config.Freeswitch.SoftmodemFallback || cidnum == "" {
		return false, nil
	}

	var err error
	if c == nil {
		c, err = eventsocket.Dial(Config.Freeswitch.Socket, Config.Freeswitch.Password)
		if err != nil {
			return false, err
		}
		defer c.Close()
	}

	exists, err := FreeSwitchDBExists(c, MOD_DB_FALLBACK_REALM, cidnum)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func SetSoftmodemFallback(c *eventsocket.Connection, cidnum string, enabled bool) error {
	if !Config.Freeswitch.SoftmodemFallback || cidnum == "" {
		return nil
	}

	var err error
	if c == nil {
		c, err = eventsocket.Dial(Config.Freeswitch.Socket, Config.Freeswitch.Password)
		if err != nil {
			return err
		}
		defer c.Close()
	}

	return FreeSwitchDBInsert(c, MOD_DB_FALLBACK_REALM, cidnum, fmt.Sprintf("%d", time.Now().Unix()))
}
