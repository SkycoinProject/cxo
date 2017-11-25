package node

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
)

// defaults
const (

	// server defaults

	EnableRPC      bool   = true        // default RPC pin
	Listen         string = ""          // default listening address
	EnableListener bool   = true        // listen by default
	RemoteClose    bool   = false       // default remote-closing pin
	RPCAddress     string = "[::]:8878" // default RPC address
	InMemoryDB     bool   = false       // default database placement pin

	// PingInterval is default interval by which server send pings
	// to connections that doesn't communicate. Actually, the
	// interval can be increased x2
	PingInterval    time.Duration = 2 * time.Second
	ResponseTimeout time.Duration = 5 * time.Second // default
	PublicServer    bool          = false           // default

	// default tree is
	//   server: ~/.skycoin/cxo/{cxds.db, idx.db}

	skycoinDataDir = ".skycoin"
	cxoSubDir      = "cxo"
)

// log pins
const (
	MsgPin       log.Pin = 1 << iota // msgs
	SubscrPin                        // subscriptions
	ConnPin                          // connect/disconnect
	RootPin                          // root receive etc
	FillPin                          // fill/drop
	HandlePin                        // handle a message
	DiscoveryPin                     // discovery
)

// default DB file names
const (
	CXDS  string = "cxds.db"
	IdxDB string = "idx.db"
)

// DataDir returns path to default data directory
func DataDir() string {
	usr, err := user.Current()
	if err != nil {
		panic(err) // fatal
	}
	if usr.HomeDir == "" {
		panic("empty home dir")
	}
	return filepath.Join(usr.HomeDir, skycoinDataDir, cxoSubDir)
}

func initDataDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// Addresses are discovery addresses
type Addresses []string

// String implements flag.Value interface
func (a *Addresses) String() string {
	return fmt.Sprintf("%v", []string(*a))
}

// Set implements flag.Value interface
func (a *Addresses) Set(addr string) error {
	*a = append(*a, addr)
	return nil
}

// A Config represents configurations
// of a Node. The config contains configurations
// for gnet.Pool and  for log.Logger. If logger of
// gnet.Config is nil, then logger of Config
// will be used
type Config struct {
	gnet.Config // pool configurations

	Log log.Config // logger configurations (logger of Node)

	// Skyobject configuration
	Skyobject *skyobject.Config

	// EnableRPC server
	EnableRPC bool
	// RPCAddress if enabled
	RPCAddress string
	// Listen on address (empty for
	// arbitrary assignment)
	Listen string
	// EnableListener turns on/off listening
	EnableListener bool

	// RemoteClose allows closing the
	// server using RPC
	RemoteClose bool

	// PingInterval used to ping clients
	// Set to 0 to disable pings
	PingInterval time.Duration

	// ResponseTimeout used by methods that requires response.
	// Zero timeout means infinity. Negative timeout causes panic
	ResponseTimeout time.Duration

	// InMemoryDB uses database in memory.
	// See also DB field
	InMemoryDB bool
	// DBPath is path to database file.
	// Because DB consist of two files,
	// the DBPath will be concated with
	// extensions ".cxds" and ".idx".
	// See also DB field. If it's
	DBPath string
	// DataDir is directory with data files
	// this directory contains default DB
	// and if it's not blank string, then
	// node creates the diretory if it does
	// not exist. If the DBPath is blank
	// then and database is not in memory
	// then the Node will use (or create and
	// use) databases under the DataDir. Even
	// if the DataDir is blank string (e.g.
	// current work directory). They named
	// "cxds.db" and "idx.db". See also
	// DB field
	DataDir string

	// PublicServer never keeps secret feeds it share
	PublicServer bool

	// ServiceDiscovery addresses
	DiscoveryAddresses Addresses

	//
	// callbacks
	//

	// connections create/close, this callbacks
	// perform in own goroutines

	OnCreateConnection func(c *Conn)
	OnCloseConnection  func(c *Conn)

	// subscribe/unsubscribe from a remote peer

	// OnSubscribeRemote called while a remote peer wants to
	// subscribe to feed of this (local) node. This callback
	// never called if subscription rejected by any reason.
	// If this callback returns a non-nil error the subscription
	// will be rejected, even if it's ok. This callback should
	// not block, because it performs inside message handling
	// goroutine and long freeze breaks connection
	OnSubscribeRemote func(c *Conn, feed cipher.PubKey) (reject error)
	// OnUnsubscribeRemote called while a remote peer wants
	// to unsubscribe from feed of this (local) node. This
	// callback never called if remote peer is not susbcribed.
	// This callback should not block, because it performs inside
	// message handling goroutine and long freeze breaks connection
	OnUnsubscribeRemote func(c *Conn, feed cipher.PubKey)

	// root objects

	// OnRootReceived is callback that called
	// when Client receive new Root object.
	// The callback never called for rejected
	// Roots (including "already exists"). This callback
	// performs in own goroutine. You can't use
	// Root of this callback anywhere because it
	// is not saved and filled yet. This callback doesn't
	// called if received a Roto already exists
	OnRootReceived func(c *Conn, root *skyobject.Root)
	// OnRootFilled is callback that called when
	// Client finishes filling received Root object.
	// This callback performs in own goroutine. The
	// Root is full and holded during this callabck.
	// You can use it anywhere
	OnRootFilled func(c *Conn, root *skyobject.Root)
	// OnFillingBreaks occurs when a filling Root
	// can't be filled up because connection breaks.
	// The Root will be removed after this callback
	// with all related objects. The Root is not full
	// and can't be used in skyobject methods.This
	// callback should not block because it performs
	// in handling goroutine
	OnFillingBreaks func(c *Conn, root *skyobject.Root, err error)

	// database

	// DB is database you can provide to use instead of
	// default. If this argument is nil (default) then
	// default DB will be created. Otherwise this
	// DB will be used. But node closes this DB on
	// close anyway
	DB *data.DB
}

// NewConfig returns Config
// filled with default values
func NewConfig() (sc Config) {
	sc.Config = gnet.NewConfig()
	sc.Log = log.NewConfig()
	sc.Skyobject = skyobject.NewConfig()
	sc.EnableRPC = EnableRPC
	sc.RPCAddress = RPCAddress
	sc.Listen = Listen
	sc.EnableListener = EnableListener
	sc.RemoteClose = RemoteClose
	sc.PingInterval = PingInterval
	sc.InMemoryDB = InMemoryDB
	sc.DataDir = DataDir()
	sc.DBPath = ""
	sc.ResponseTimeout = ResponseTimeout
	sc.PublicServer = PublicServer
	sc.Config.OnDial = OnDialFilter
	return
}

// FromFlags obtains value from command line flags.
// Call the method before `flag.Parse` for example
//
//     c := node.NewConfig()
//     c.FromFlags()
//     flag.Parse()
//
func (s *Config) FromFlags() {
	s.Config.FromFlags()
	s.Log.FromFlags()

	flag.BoolVar(&s.EnableRPC,
		"rpc",
		s.EnableRPC,
		"enable RPC server")
	flag.StringVar(&s.RPCAddress,
		"rpc-address",
		s.RPCAddress,
		"address of RPC server")
	flag.StringVar(&s.Listen,
		"address",
		s.Listen,
		"listening address (pass empty string to arbitrary assignment by OS)")
	flag.BoolVar(&s.EnableListener,
		"enable-listening",
		s.EnableListener,
		"enable listening pin")
	flag.BoolVar(&s.RemoteClose,
		"remote-close",
		s.RemoteClose,
		"allow closing the server using RPC")
	flag.DurationVar(&s.PingInterval,
		"ping",
		s.PingInterval,
		"interval to send pings (0 = disable)")
	flag.BoolVar(&s.InMemoryDB,
		"mem-db",
		s.InMemoryDB,
		"use in-memory database")
	flag.StringVar(&s.DataDir,
		"data-dir",
		s.DataDir,
		"directory with data")
	flag.StringVar(&s.DBPath,
		"db-path",
		s.DBPath,
		"path to database")
	flag.DurationVar(&s.ResponseTimeout,
		"response-tm",
		s.ResponseTimeout,
		"response timeout (0 = infinity)")
	flag.BoolVar(&s.PublicServer,
		"public-server",
		s.PublicServer,
		"make the server public")
	flag.Var(&s.DiscoveryAddresses,
		"discovery-address",
		"address of service discovery")

	// TODO: skyobject.Configs from flags

	return
}

// OnDialFilter is gnet.OnDial callback that rejects redialing if
// remote peer closes connection
func OnDialFilter(c *gnet.Conn, err error) (reject error) {
	if ne, ok := err.(net.Error); ok {
		if ne.Temporary() {
			return // nil
		}
		if oe, ok := err.(*net.OpError); ok {
			// reading fails with EOF if remote peer closes connection,
			// but writing ...
			if se, ok := oe.Err.(*os.SyscallError); ok {
				if errno, ok := se.Err.(syscall.Errno); ok {
					if errno == 0x68 {
						// "connection reset by peer"
						return err // reject
					}
				}
			}
		}
	} else if err == io.EOF { // connection has been closed by peer
		return err // reject
	}
	return // nil (unknowm case)
}
