package cxoutils

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// RemoveRootObjects removes old Root objects from given
// *skyobject.Container keeping last n-th. Call this method
// using some interval. The method affects all feeds of
// the Container and all heads.
//
// If a feed contains more then one head, then the method
// keeps last n-th Root objects of every head.
func RemoveRootObjects(c *skyobject.Container, keepLast int) (err error) {

	for _, pk := range c.Feeds() {

		var heads []uint64
		if heads, err = c.Heads(pk); err != nil {
			return
		}

	HeadLoop:
		for _, nonce := range heads {
			var seq uint64
			if seq, err = c.LastRootSeq(pk, nonce); err != nil {
				continue // not a real error
			}

			if seq <= uint64(keepLast) {
				continue
			}

			var goDown = seq - uint64(keepLast) // positive

			for ; goDown > 0; goDown-- {

				if err = c.DelRoot(pk, nonce, goDown); err != nil {
					if err == data.ErrNotFound {
						continue HeadLoop
					}
					return // a failure (CXDS not found error?)
				}

			}

			// seq = 0 (goDown == 0)
			if err = c.DelRoot(pk, nonce, 0); err != nil {
				if err == data.ErrNotFound {
					continue HeadLoop
				}

				return
			}

			// continue HeadLoop

		} // head loop

	} // feed loop

	return
}

// RemoveObjects with rc == 0 from CXDS
func RemoveObjects(c *skyobject.Container) (err error) {

	var db = c.DB().CXDS()

	err = db.IterateDel(
		func(_ cipher.SHA256, rc uint32, _ []byte) (bool, error) {
			return rc == 0, nil
		})

	return
}

// TODO (kostyarin): RemoveObjects with "down to" feature
