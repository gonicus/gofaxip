package gofaxlib

import (
	"fmt"
	"github.com/fiorix/go-eventsocket/eventsocket"
)

func FreeSwitchDBInsert(c *eventsocket.Connection, realm string, key string, value string) error {
	_, err := c.Send(fmt.Sprintf("api db insert/%s/%s/%s", realm, key, value))
	if err != nil {
		return err
	}
	return nil
}

func FreeSwitchDBDelete(c *eventsocket.Connection, realm string, key string) error {
	_, err := c.Send(fmt.Sprintf("api db delete/%s/%s", realm, key))
	if err != nil {
		return err
	}
	return nil
}

func FreeSwitchDBSelect(c *eventsocket.Connection, realm string, key string) (string, error) {
	result, err := c.Send(fmt.Sprintf("api db select/%s/%s", realm, key))
	if err != nil {
		return "", err
	}
	return result.Body, nil
}

func FreeSwitchDBExists(c *eventsocket.Connection, realm string, key string) (bool, error) {
	result, err := c.Send(fmt.Sprintf("api db exists/%s/%s", realm, key))
	if err != nil {
		return false, err
	}

	return result.Body == "true", nil
}
