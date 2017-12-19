package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

type User struct {
	Name   string
	Age    uint32
	Hidden []byte `enc:"-"`
}

type Feed struct {
	Posts registry.Refs `skyobject:"schema=test.Post"`
}

type Post struct {
	Head string
	Body string
	Time int64
}

func getTestConfig(prefix string) (c *Config) {

	c = NewConfig()
	c.Logger.Prefix = "[" + prefix + "] "
	c.Config.InMemoryDB = true // use in-memory DB

	c.TCP.Listen = "127.0.0.1:8087"
	c.TCP.Discovery = Addresses{}           // blank
	c.TCP.ResponseTimeout = 1 * time.Second //
	c.TCP.Pings = 0                         // no pings
	c.RPC = ""                              // no rpc

	c.UDP.Listen = "127.0.0.1:8087"
	c.UDP.Discovery = Addresses{}           // blank
	c.UDP.ResponseTimeout = 1 * time.Second //
	c.UDP.Pings = 0                         // no pings
	c.RPC = ""                              // no rpc

	if testing.Verbose() == true {
		c.Logger.Debug = true
		c.Logger.Pins = ^c.Logger.Pins // all
	}

	return

}

func getTestNode(prefix string) (n *Node) {

	var err error
	if n, err = NewNode(getTestConfig(prefix)); err != nil {
		panic(err)
	}

	return
}

func getTestRegistry() (r *registry.Registry) {

	return registry.NewRegistry(func(r *registry.Reg) {
		r.Register("test.User", User{})
		r.Register("test.Post", Post{})
		r.Register("test.Feed", Feed{})
	})

}

func assertNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_send_receive(t *testing.T) {

	var (
		fr           = make(chan *registry.Root, 100)
		onRootFilled = func(_ *Node, r *registry.Root) { fr <- r }

		sn = getTestNode("sender")

		rconf = getTestConfig("receiver")
	)

	rconf.TCP.Listen = ""             // don't listen
	rconf.UDP.Listen = ""             // don't listen
	rconf.OnRootFilled = onRootFilled // callback
	rconf.OnRootReceived = func(c *Conn, r *registry.Root) (_ error) {
		t.Logf("[%s] root received %s", c.String(), r.Short())
		return
	}
	rconf.OnFillingBreaks = func(n *Node, r *registry.Root, err error) {
		t.Logf("filling of %s breaks by %v", r.Short(), err)
	}

	var rn, err = NewNode(rconf)

	if err != nil {
		t.Fatal(err)
	}

	defer sn.Close()
	defer rn.Close()

	var pk, sk = cipher.GenerateKeyPair()

	assertNil(t, sn.Share(pk))
	assertNil(t, rn.Share(pk))

	var (
		reg = getTestRegistry()
		sc  = sn.Container()

		up *skyobject.Unpack
	)

	if up, err = sc.Unpack(sk, reg); err != nil {
		t.Fatal(err)
	}

	var r = new(registry.Root)

	r.Nonce = 9021 // random
	r.Pub = pk     // set
	r.Descriptor = []byte("hey-ho!")

	r.Refs = append(r.Refs,
		dynamicByValue(t, up, "test.User", User{"Alice", 19, nil}),
		dynamicByValue(t, up, "test.Feed", Feed{}),
	)

	// save the Root
	if err := sc.Save(up, r); err != nil {
		t.Fatal(err)
	}

	if sn.TCP().Address() == "" {
		t.Fatal("blank listening address")
	}

	// connect the nodes between
	var c *Conn
	if c, err = rn.TCP().Connect(sn.TCP().Address()); err != nil {
		t.Fatal(err)
	}

	// subscribe the connection
	if err = c.Subscribe(pk); err != nil {
		t.Fatal(err)
	}

	var rr *registry.Root // received Root

	// wait the Root
	select {
	case rr = <-fr:
	case <-time.After(10 * time.Second):

		t.Log("Root :    ", r.Hash.Hex()[:7])
		t.Log("Registry: ", r.Reg.Short())

		t.Log("[sender] objects")
		printObjects(t, "sender", sc)

		t.Log("[receiver] objects")
		printObjects(t, "receiver", rn.Container())

		t.Fatal("slow")
	}

	// TODO (kostyarin): compare Root objects

	_ = rr

}

func printObjects(t *testing.T, prefix string, c *skyobject.Container) {
	err := c.DB().CXDS().Iterate(
		func(key cipher.SHA256, rc uint32, _ []byte) (_ error) {
			t.Logf("[%s] %s %d", prefix, key.Hex()[:7], rc)
			return
		})
	if err != nil {
		t.Fatal(err)
	}
}

func dynamicByValue(
	t *testing.T,
	up *skyobject.Unpack,
	name string,
	obj interface{},
) (
	dr registry.Dynamic,
) {

	sch, err := up.Registry().SchemaByName(name)

	if err != nil {
		t.Fatal(err)
	}

	val := encoder.Serialize(obj)
	key := cipher.SumSHA256(val)

	dr.Schema = sch.Reference()
	dr.Hash = key

	if err := up.Set(key, val); err != nil {
		t.Fatal(err)
	}

	return

}

// with registry.Refs
func Test_send_receive_refs(t *testing.T) {

	var (
		fr           = make(chan *registry.Root, 100)
		onRootFilled = func(_ *Node, r *registry.Root) { fr <- r }

		sn = getTestNode("sender")

		rconf = getTestConfig("receiver")
	)

	rconf.TCP.Listen = ""             // don't listen
	rconf.UDP.Listen = ""             // don't listen
	rconf.OnRootFilled = onRootFilled // callback
	rconf.OnRootReceived = func(c *Conn, r *registry.Root) (_ error) {
		t.Logf("[%s] root received %s", c.String(), r.Short())
		return
	}
	rconf.OnFillingBreaks = func(n *Node, r *registry.Root, err error) {
		t.Logf("filling of %s breaks by %v", r.Short(), err)
	}

	var rn, err = NewNode(rconf)

	if err != nil {
		t.Fatal(err)
	}

	defer sn.Close()
	defer rn.Close()

	var pk, sk = cipher.GenerateKeyPair()

	assertNil(t, sn.Share(pk))
	assertNil(t, rn.Share(pk))

	var (
		reg = getTestRegistry()
		sc  = sn.Container()

		up *skyobject.Unpack
	)

	if up, err = sc.Unpack(sk, reg); err != nil {
		t.Fatal(err)
	}

	var r = new(registry.Root)

	r.Nonce = 9021 // random
	r.Pub = pk     // set
	r.Descriptor = []byte("hey-ho!")

	var feed Feed

	for i := 0; i < 32; i++ {

		err := feed.Posts.AppendValues(up, Post{
			Head: fmt.Sprintf("Head #%d", i),
			Body: fmt.Sprintf("Body #%d", i),
			Time: time.Now().UnixNano(),
		})

		if err != nil {
			t.Fatal(err)
		}

	}

	r.Refs = append(r.Refs,
		dynamicByValue(t, up, "test.User", User{"Alice", 19, nil}),
		dynamicByValue(t, up, "test.Feed", feed),
	)

	// save the Root
	if err := sc.Save(up, r); err != nil {
		t.Fatal(err)
	}

	if sn.TCP().Address() == "" {
		t.Fatal("blank listening address")
	}

	// connect the nodes between
	var c *Conn
	if c, err = rn.TCP().Connect(sn.TCP().Address()); err != nil {
		t.Fatal(err)
	}

	// subscribe the connection
	if err = c.Subscribe(pk); err != nil {
		t.Fatal(err)
	}

	var rr *registry.Root // received Root

	// wait the Root
	select {
	case rr = <-fr:
	case <-time.After(10 * time.Second):

		t.Log("Root :    ", r.Hash.Hex()[:7])
		t.Log("Registry: ", r.Reg.Short())

		t.Log("[sender] objects")
		printObjects(t, "sender", sc)

		t.Log("[receiver] objects")
		printObjects(t, "receiver", rn.Container())

		t.Fatal("slow")
	}

	// TODO (kostyarin): compare the Root objects

	_ = rr

}
