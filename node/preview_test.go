package node

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

func Test_preview(t *testing.T) {

	var (
		sn    = getTestNode("sender")
		rconf = getTestConfig("receiver")
	)

	rconf.TCP.Listen = "" // don't listen
	rconf.UDP.Listen = "" // don't listen

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

	// preview

	err = c.Preview(pk,
		func(pp registry.Pack, r *registry.Root) (subscribe bool) {

			if len(r.Refs) != 2 {
				t.Error("wrong Root.Refs length")
				return
			}

			var usr User

			if err := r.Refs[0].Value(pp, &usr); err != nil {
				t.Error(err)
				return
			}

			if usr.Name != "Alice" {
				t.Error("wrong user name:", usr.Name)
			}

			if usr.Age != 19 {
				t.Error("wrong user age:", usr.Age)
			}

			var feed Feed

			if err := r.Refs[1].Value(pp, &feed); err != nil {
				t.Error(err)
				return
			}

			if ln, err := feed.Posts.Len(pp); err != nil {
				t.Error(err)
			} else if ln != 0 {
				t.Error("wrong length of Posts")
			}

			return false
		})

}
