package gnet

import (
	"flag"
	"testing"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c.MaxConnections != MaxConnections {
		t.Errorf("unexpected MaxConnections, want %v, got %v",
			MaxConnections, c.MaxConnections)
	}
	if c.MaxMessageSize != MaxMessageSize {
		t.Errorf("unexpected MaxMessageSize, want %v, got %v",
			MaxMessageSize, c.MaxMessageSize)
	}
	if c.DialTimeout != DialTimeout {
		t.Errorf("unexpected DialTimeout, want %v, got %v",
			DialTimeout, c.DialTimeout)
	}
	if c.ReadTimeout != ReadTimeout {
		t.Errorf("unexpected ReadTimeout, want %v, got %v",
			ReadTimeout, c.ReadTimeout)
	}
	if c.WriteTimeout != WriteTimeout {
		t.Errorf("unexpected WriteTimeout, want %v, got %v",
			WriteTimeout, c.WriteTimeout)
	}
	if c.ReadQueueLen != ReadQueueLen {
		t.Errorf("unexpected ReadQueueLen, want %v, got %v",
			ReadQueueLen, c.ReadQueueLen)
	}
	if c.WriteQueueLen != WriteQueueLen {
		t.Errorf("unexpected WriteQueueLen, want %v, got %v",
			WriteQueueLen, c.WriteQueueLen)
	}
	if c.RedialTimeout != RedialTimeout {
		t.Errorf("unexpected RedialTimeout, want %v, got %v",
			RedialTimeout, c.RedialTimeout)
	}
	if c.MaxRedialTimeout != MaxRedialTimeout {
		t.Errorf("unexpected MaxRedialTimeout, want %v, got %v",
			MaxRedialTimeout, c.MaxRedialTimeout)
	}
	if c.DialsLimit != DialsLimit {
		t.Errorf("unexpected DialsLimit, want %v, got %v",
			DialsLimit, c.DialsLimit)
	}
	if c.ReadBufferSize != ReadBufferSize {
		t.Errorf("unexpected ReadBufferSize, want %v, got %v",
			ReadBufferSize, c.ReadBufferSize)
	}
	if c.WriteBufferSize != WriteBufferSize {
		t.Errorf("unexpected WriteBufferSize, want %v, got %v",
			WriteBufferSize, c.WriteBufferSize)
	}
}

func TestConfig_FromFlags(t *testing.T) {
	c := NewConfig()
	c.FromFlags()

	flag.Set("max-conns", "53")
	flag.Set("max-msg-size", "53")

	flag.Set("dial-timeout", "53ns")
	flag.Set("read-timeout", "53ns")
	flag.Set("write-timeout", "53ns")

	flag.Set("read-qlen", "53")
	flag.Set("write-qlen", "53")

	flag.Set("redial-timeout", "53ns")
	flag.Set("max-redial-timeout", "53ns")
	flag.Set("redials-limit", "53")

	flag.Set("read-buf", "53")
	flag.Set("write-buf", "53")

	if c.MaxConnections != 53 {
		t.Error("MaxConnections doesn't set from flags properly:",
			c.MaxConnections)
	}
	if c.MaxMessageSize != 53 {
		t.Error("MaxMessageSize doesn't set from flags properly:",
			c.MaxMessageSize)
	}
	if c.DialTimeout != 53 {
		t.Error("DialTimeout doesn't set from flags properly:",
			c.DialTimeout)
	}
	if c.ReadTimeout != 53 {
		t.Error("ReadTimeout doesn't set from flags properly:",
			c.ReadTimeout)
	}
	if c.WriteTimeout != 53 {
		t.Error("WriteTimeout doesn't set from flags properly:",
			c.WriteTimeout)
	}
	if c.ReadQueueLen != 53 {
		t.Error("ReadQueueLen doesn't set from flags properly:",
			c.ReadQueueLen)
	}
	if c.WriteQueueLen != 53 {
		t.Error("WriteQueueLen doesn't set from flags properly:",
			c.WriteQueueLen)
	}
	if c.RedialTimeout != 53 {
		t.Error("RedialTimeout doesn't set from flags properly:",
			c.RedialTimeout)
	}
	if c.MaxRedialTimeout != 53 {
		t.Error("MaxRedialTimeout doesn't set from flags properly:",
			c.MaxRedialTimeout)
	}
	if c.DialsLimit != 53 {
		t.Error("DialsLimit doesn't set from flags properly:",
			c.DialsLimit)
	}
	if c.ReadBufferSize != 53 {
		t.Error("ReadBufferSize doesn't set from flags properly:",
			c.ReadBufferSize)
	}
	if c.WriteBufferSize != 53 {
		t.Error("WriteBufferSize doesn't set from flags properly:",
			c.WriteBufferSize)
	}
}

func TestConfig_Validate(t *testing.T) {
	for i, c := range []Config{
		{MaxConnections: -1},
		{MaxMessageSize: -1},
		{DialTimeout: -1},
		{ReadTimeout: -1},
		{WriteTimeout: -1},
		{ReadQueueLen: -1},
		{WriteQueueLen: -1},
		{RedialTimeout: -1},
		{MaxRedialTimeout: -1},
		{DialsLimit: -1},
		{ReadBufferSize: -1},
		{WriteBufferSize: -1},
	} {
		if err := c.Validate(); err == nil {
			t.Error("missing error", i)
		}
	}

}
