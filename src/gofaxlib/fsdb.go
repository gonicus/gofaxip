package gofaxlib

import (
	"fmt"
	"github.com/fiorix/go-eventsocket/eventsocket"
)

// FreeSwitchDBInsert inserts a value into FreeSWITCH's mod_db key/value database
func FreeSwitchDBInsert(c *eventsocket.Connection, realm string, key string, value string) error {
	_, err := c.Send(fmt.Sprintf("api db insert/%s/%s/%s", realm, key, value))
	if err != nil {
		return err
	}
	return nil
}

// FreeSwitchDBDelete deletes a value from FreeSWITCH's mod_db key/value database
func FreeSwitchDBDelete(c *eventsocket.Connection, realm string, key string) error {
	_, err := c.Send(fmt.Sprintf("api db delete/%s/%s", realm, key))
	if err != nil {
		return err
	}
	return nil
}

// FreeSwitchDBSelect retreives a value from FreeSWITCH's mod_db key/value database
func FreeSwitchDBSelect(c *eventsocket.Connection, realm string, key string) (string, error) {
	result, err := c.Send(fmt.Sprintf("api db select/%s/%s", realm, key))
	if err != nil {
		return "", err
	}
	return result.Body, nil
}

// FreeSwitchDBExists checks if a value exists in FreeSWITCH's mod_db key/value database
func FreeSwitchDBExists(c *eventsocket.Connection, realm string, key string) (bool, error) {
	result, err := c.Send(fmt.Sprintf("api db exists/%s/%s", realm, key))
	if err != nil {
		return false, err
	}

	return result.Body == "true", nil
}
