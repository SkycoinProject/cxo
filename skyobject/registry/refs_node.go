package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// an element of the Refs
type refsElement struct {
	Hash  cipher.SHA256 // hash (blank if nil)
	upper *refsNode     // upper node or nil if the node is the Refs
}

// a branch of the Refs
type refsNode struct {
	hash   cipher.SHA256 // hash of this node
	length int           // length of this subtree
	mods   refsMod       // unsaved modifications

	leafs    []*refsElement // leafs (elements)
	branches []*refsNode    // branhces (subtree)

	upper *refsNode // upper node
}

// DB representation of the refsNode
type encodedRefsNode struct {
	Length   uint32          // length of the node
	Elements []cipher.SHA256 // if empty then all elements are nils
}

//
// loading
//

// create leaf by given hash, the name starts with
// load by analogy of loadBranch
func (r *Refs) loadLeaf(
	hash cipher.SHA256, // : hash of the leaf
	upper *refsNode, //    : upper node
) (
	re *refsElement, //    : the "loaded" leaf
) {

	re = new(refsElement)

	re.Hash = hash
	re.upper = upper

	if r.flags&HashTableIndex != 0 {
		r.addElementToIndex(re)
	}

	return
}

// create leafs by given elements
func (r *Refs) loadLeafs(
	rn *refsNode, // node that contains this leafs
	elements []cipher.SHA256, // elemets
) {

	rn.leafs = make([]*refsElement, 0, len(elements))

	for _, hash := range elements {
		rn.leafs = append(rn.leafs, r.loadLeaf(hash, rn))
	}

	rn.mods |= loadedMod // use flag to mark as loaded

	return
}

// laod branch, setting hash and upper fields and loading
// deeppre if HashTableIndex or (or and) EntireRefs flags
// set
func (r *Refs) loadBranch(
	pack Pack, //          : pack to get the elemet
	hash cipher.SHA256, // : hash of the branch
	depth int, //          : depth of the branch
	upper *refsNode, //    : upper node
) (
	br *refsNode, //        : the loaded branch
	err error, //           : error if any
) {

	if hash == (cipher.SHA256{}) {
		return nil, ErrInvalidRefs
	}

	br = new(refsNode)

	br.hash = hash
	br.upper = upper

	if r.flags&(HashTableIndex|EntireRefs) == 0 {
		return // don't load deepper (lazy loading)
	}

	// load deepper
	if err = r.loadNode(pack, br, depth); err != nil {
		br = nil // GC
		return   // failed
	}

	return
}

// 'hash' and 'upper' fields of the rn are already set;
// the depth if not 0 (e.g. the elements point to refsNode)
//
// the loadBranches doesn't load entire tree if it's not
// necessary; e.g. it depends on flags HashTableIndex and
// EntireRefs
func (r *Refs) loadBranches(
	pack Pack, //                : pack to load
	rn *refsNode, //             : the node
	depth int, //                : depth of the branches (> 0)
	elements []cipher.SHA256, // : elements of the branches
) (
	err error, //                : pack/decoding related error
) {

	rn.branches = make([]*refsNode, 0, len(elements))

	var br *refsNode
	for _, hash := range elements {
		if br, err = r.loadBranch(pack, hash, depth-1, rn); err != nil {
			return
		}
		rn.branches = append(rn.branches, br)
	}

	rn.mods |= loadedMod // use flag to mark as loaded

	return
}

// 'hash' and 'upper' fields of the rn are already set;
// the loadSubtree doesn't load entire tree if it's not
// necessary; e.g. the loading depends on falgs
// HashTableIndex and EntireRefs
func (r *Refs) loadSubtree(
	pack Pack, //                : pack to load
	rn *refsNode, //             : load subtree of this
	elements []cipher.SHA256, // : elements to load
	depth int, //                : depth of the rn (of the elements)
) (
	err error, //                : pack/decoding related error
) {

	if depth == 0 {
		r.loadLeafs(rn, elements)
		return // if the depth is 0, then the node contains leafs
	}

	// else if depth > 0, then the node contains branches
	return r.loadBranches(pack, rn, depth, elements)
}

// loadNode that already contains hash and upper fields;
// the method loads the node anyway (no flags affect it)
func (r *Refs) loadNode(
	pack Pack, //    : pack to get
	rn *refsNode, // : the branch to load
	depth int, //    : depth of the rn (> 0)
) (
	err error, //    : get, decoding or 'invalid refs' error
) {

	var ern encodedRefsNode // encoded branch
	if err = get(pack, rn.hash, &ern); err != nil {
		return // get or decoding error
	}

	rn.length = int(ern.Length)

	return r.loadSubtree(pack, rn, ern.Elements, depth) // depth is the same
}

// isLoaded returns true if the node is loaded
func (r *refsNode) isLoaded() bool {
	return r.mods&loadedMod != 0
}

// load given node if it's not loaded yet;
// e.g. if lazy loading used
func (r *Refs) loadNodeIfNeed(
	pack Pack, //    : pack to get
	rn *refsNode, // : the branch to load
	depth int, //    : depth of the rn (> 0)
) (
	err error, //    : get, decoding or 'invalid refs' error
) {

	if rn.isLoaded() == true {
		return // already loaded
	}

	return r.loadNode(pack, rn, depth)
}

//
// encoding
//

// encode a the refsNode as is
func (r *refsNode) encode(depth int) []byte {

	var ern encodedRefsNode

	ern.Length = uint32(r.length)

	if depth == 0 {

		ern.Elements = make([]cipher.SHA256, 0, len(r.leafs))
		for _, el := range r.leafs {
			ern.Elements = append(ern.Elements, el.Hash)
		}

	} else {

		ern.Elements = make([]cipher.SHA256, 0, len(r.branches))
		for _, br := range r.branches {
			ern.Elements = append(ern.Elements, br.hash)
		}

	}

	return encoder.Serialize(ern)
}

//
// index of element in Refs
//

// indexInRefs finds index of the leaf in Refs
func (r *refsElement) indexInRefs() (i int, err error) {

	if i, err = r.indexInUpper(); err != nil {
		return // invalid state
	}

	// upper node (can not be nil, but can be &Refs.refsNode)
	var j int
	if j, err = r.upper.indexInRefs(); err != nil {
		return
	}

	return i + j, nil
}

// index of the element in upper refsNode
func (r *refsElement) indexInUpper() (i int, err error) {
	var el *refsElement
	for i, el = range r.upper.leafs {
		if el == r {
			return // found
		}
	}
	return 0, ErrInvalidRefs // can't find element in upper leafs
}

// indexInRefs returns index of first element of the
// node in Refs; e.g. a refsNode has not an index, but
// it contains elements with indicies, and this mehtod
// returns index of first element
func (r *refsNode) indexInRefs() (i int, err error) {

Upper:
	for up, down := r.upper, r; up != nil; up, down = up.upper, up {
		for _, br := range up.branches {
			if br != down {
				i += br.length
			}
			continue Upper
		}
		return 0, ErrInvalidRefs // can't find in upper.branches
	}

	return
}

//
// find element by index
//

// elementByIndex finds *refsElemet by given index;
func (r *Refs) elementByIndex(
	pack Pack, //       : pack to load
	rn *refsNode, //    : the node to find inside (should be loaded)
	i int, //           : index of the needle
	depth int, //       : depth of the rn
) (
	el *refsElement, // : element if found
	err error, //       : error if any
) {

	// so, the rn is already loaded

	if depth == 0 { // take a look at leafs

		for j, el := range rn.leafs {
			if j == i {
				return el, nil // found
			}
		}

		return nil, ErrInvalidRefs // can't find the element
	}

	// else, take a look at branches

	var br *refsNode
	for _, br = range rn.branches {

		if err = r.loadNodeIfNeed(pack, br, depth-1); err != nil {
			return
		}

		if i < br.length {
			i -= br.length // subtract length of the skipped branch
			continue       // and skip the branch
		}

		break // the branch that contains the needle has been found
	}

	return r.elementByIndex(pack, br, i, depth-1)
}

//
// change hash of refsElement
//

// updateNodeHash updates hash of the node
// and clears contentMod flag
func (r *refsNode) updateHash(
	pack Pack, //    : pack to save
	depth int, //    : depth of the node
) (
	err error, //    : saving error
) {

	// encode
	var val = r.encode(depth)
	// get hash
	var hash = cipher.SumSHA256(val)

	if hash == r.hash {
		return // the hash is the same
	}

	// don't Save if the node is part of the Refs
	// it will be saved inside another method
	// with depth and degree
	if r.upper != nil {

		// save the node
		if err = pack.Set(hash, val); err != nil {
			return
		}

		// ignore the field if the node is
		// part of the Refs and set it in
		// other cases

		r.hash = hash // set the  hash

	}

	r.mods &^= contentMod // clear the flag if it has been set
	return
}

//updateHashIfNeed updates hash by given condition
func (r *refsNode) updateHashIfNeed(
	pack Pack, // : pack to save
	depth int, // : depth of the node
	need bool, // : condition
) (
	err error, // : error if any
) {

	if need == false {
		r.mods |= contentMod // modified, but not saved
		return
	}

	return r.updateHash(pack, depth)
}

// bubbleContentChanges change hashes of
// all upper nodes if LazyUpdating flag
// is not set
func (r *Refs) bubbleContentChanges(
	pack Pack, //       : pack to save
	el *refsElement, // : element to start bubbling
) (
	err error, //       : error if any
) {

	var need = r.flags&LazyUpdating == 0 // need != lazy
	var depth int

	// up.upper == nil if the up is Refs.refsNode
	for up := el.upper; up.upper != nil; up = up.upper {

		if err = up.updateHashIfNeed(pack, depth, need); err != nil {
			return // saving error
		}

		depth++ // the depth grows
	}

	return r.updateHashIfNeed(pack, need)
}

// setElementHash replaces element hash
func (r *Refs) setElementHash(
	pack Pack, //          : pack to save
	el *refsElement, //    : element to change
	hash cipher.SHA256, // : new hash
) (
	err error, //          : error if any
) {

	if el.Hash == hash {
		return // nothing to change
	}

	if r.flags&HashTableIndex != 0 {
		r.delElementFromIndex(el) // delete old
		r.addElementToIndex(el)   // add new
	}

	el.Hash = hash

	// so, length of the Refs is still the same
	// but content has been changed

	return r.bubbleContentChanges(pack, el)
}

//
// delete
//

func (r *refsNode) deleteElementByIndex(i int) {
	copy(r.leafs[i:], r.leafs[i+1:])
	r.leafs[len(r.leafs)-1] = nil
	r.leafs = r.leafs[:len(r.leafs)-1]
}

func (r *refsNode) deleteNodeByIndex(i int) {
	copy(r.branches[i:], r.branches[i+1:])
	r.branches[len(r.branches)-1] = nil
	r.branches = r.branches[:len(r.branches)-1]
}

// deleteElementByIndex deletes *refsElemet by given index
func (r *Refs) deleteElementByIndex(
	pack Pack, //    : pack to load
	rn *refsNode, // : the node to find inside (should be loaded)
	i int, //        : index of the element
	depth int, //    : depth of the rn
) (
	err error, //    : error if any
) {

	// so, the rn is already loaded

	var j int // index in upper node

	if depth == 0 { // take a look at leafs

		var el *refsElement
		for j, el = range rn.leafs {

			if j == i { // found

				if r.flags&HashTableIndex != 0 {
					r.delElementFromIndex(el) // remove from hash-table index
				}

				rn.deleteElementByIndex(j) // remove from leafs
				rn.length--                // decrement length

				if rn.length == 0 {
					return // don't update hash if the length is zero
				}

				if rn.upper == nil {
					// don't update node hash, because the rn is Refs.refsNode;
					// the Refs should be updated after
					return
				}

				err = rn.updateHashIfNeed(pack, depth,
					r.flags&LazyUpdating == 0)

				return // deleted
			}

		}

		return ErrInvalidRefs // can't find the element
	}

	// else, take a look at branches

	var br *refsNode
	for j, br = range rn.branches {

		if err = r.loadNodeIfNeed(pack, br, depth-1); err != nil {
			return
		}

		if i < br.length {
			i -= br.length // subtract length of the skipped branch
			continue       // and skip the branch
		}

		break // the branch that contains the needle has been found
	}

	// keep j for now

	if err = r.deleteElementByIndex(pack, br, i, depth-1); err != nil {
		return // an error
	}

	if br.length == 0 {
		rn.deleteNodeByIndex(j) // delete node, because it's empty
	}

	rn.length--

	if rn.length == 0 {
		return // don't update hash of the rn if its length is zeros
	}

	if rn.upper == nil {
		// don't update hash if the upepr is nil, because the rn is
		// Refs.refsNode that should be processed after
		return
	}

	return rn.updateHashIfNeed(pack, depth, r.flags&LazyUpdating == 0)
}

// deleteElement from the Refs if the Refs is loaded
// and the element is found by hash-table
func (r *Refs) deleteElement(
	pack Pack, //       : pack to save
	el *refsElement, // : element to delete
) (
	err error, //       : error if any
) {

	r.delElementFromIndex(el)

	var up = el.upper

	for i, leaf := range up.leafs {
		if leaf == el {
			up.deleteElementByIndex(i)
			up.length--

			//

			return
		}
	}

	return ErrInvalidRefs // can't find in upper leafs
}

//
// ascend
//

// ascendNode depending on depth and i (starting element), check
// changes after every call of the ascendFunc
func (r *Refs) ascendNode(
	pack Pack, //              : pack to load
	rn *refsNode, //           : loaded node
	depth int, //              : depth of the rn
	shift int, //              : index from which the node starts
	i int, //                  : starting index
	ascendFunc IterateFunc, // : the function
) (
	pass int, //               : passed elements (not skipped)
	rewind bool, //            : have to find next index from root
	err error, //              : erro if any
) {

	if depth == 0 {
		return r.ascendNodeLeafs(rn, shift, i, ascendFunc)
	}

	// else if depth > 0

	return r.ascendNodeBranches(pack, rn, depth, shift, i, ascendFunc)
}

// ascendNodeLeafs iterates over lefas starting
// from i-th element (the i is absolute index)
func (r *Refs) ascendNodeLeafs(
	rn *refsNode, //           : the node
	shift int, //              : index from which the leafs start from
	i int, //                  : index to start from (absolute index)
	ascendFunc IterateFunc, // : the function
) (
	pass int, //               : number of processed elements
	rewind bool, //            : have to find next element from root
	err error, //              : error if any
) {

	// e.g. shift + j = absolute index
	for j, el := range rn.leafs {

		if shift+j < i {
			continue // skip
		}

		// here i == shift + j

		if err = ascendFunc(i, el.Hash); err != nil {
			return
		}

		pass++ // one more element has been processed

		// check out changes of the Refs
		if rewind = r.isFindingIteratorsIndexFromRootRequired(); rewind {
			// we need to find next element from root of the Refs,
			// because the ascendFunc changes length of the Refs tree
			return
		}

		i++ // increment current absolute index

		// no changes required, continue
	}

	return
}

// ascendNodeBranches iterate element of the ndoe
func (r *Refs) ascendNodeBranches(
	pack Pack, //              : pack to load
	rn *refsNode, //           : the node
	depth int, //              : depth of the node
	shift int, //              : index from which the branches start from
	i int, //                  : starting element (absolute index)
	ascendFunc IterateFunc, // : the function
) (
	pass int, //               : processed elements
	rewind bool, //            : have to find next element from root
	err error, //              : error if any
) {

	var subpass int // pass for a subtree (for a branch)

	for _, br := range rn.branches {

		if err = r.loadNodeIfNeed(pack, br, depth-1); err != nil {
			return
		}

		if shift+br.length < i {
			shift += br.length
			continue // skip this branch finding i-th element
		}

		subpass, rewind, err = r.ascendNode(pack, br, depth-1, shift, i,
			ascendFunc)

		pass += subpass // process elements

		if err != nil || rewind == true {
			return // some error or changes
		}

		// these are local variables and incrementing has meaning only
		// if we a going to ascend next branch

		i += subpass     // increment absolute index of current element
		shift += subpass // move the shift forward

		// continue
	}

	return // done
}

//
// descend
//

// descendNode depending on depth and i (starting element), check
// changes after every call of the descendFunc
func (r *Refs) descendNode(
	pack Pack, //               : pack to load
	rn *refsNode, //            : the node
	depth int, //               : depth of the node
	shift int, //               : shift of the node starting from the end
	i int, //                   : absolute index of starting element (from)
	descendFunc IterateFunc, // : the function
) (
	pass int, //                : processed elements
	rewind bool, //             : have to find next element from root
	err error, //               : error if any
) {

	if depth == 0 {
		return r.descendNodeLeafs(rn, shift, i, descendFunc)
	}

	// else if depth > 0

	return r.descendNodeBranches(pack, rn, depth, shift, i, descendFunc)
}

// descendNodeLeafs iterates over leafs of given node
func (r *Refs) descendNodeLeafs(
	rn *refsNode, //            : the node
	shift int, //               : ending indexof the leafs of the node
	i int, //                   : absolute index to start from
	descendFunc IterateFunc, // : the function
) (
	pass int, //                : processd elements
	rewind bool, //             : have to find next element from root
	err error, //               : error if any
) {

	// so, the shift is absolute index of last element of the leafs;
	// we have to find i-th index to start iterating from it

	// so, since the shift is local variable
	// we are free to decrement it

	for k := len(rn.leafs) - 1; k >= 0; k-- {

		var el = rn.leafs[k] // current element

		if shift > i {
			shift--
			continue // skip current element finding i-th
		}

		// here i == shift

		if err = descendFunc(i, el.Hash); err != nil {
			return
		}

		pass++ // processed elements

		// check out changes of the Refs
		if rewind = r.isFindingIteratorsIndexFromRootRequired(); rewind {
			// we need to find next element from root of the Refs,
			// because the descendFunc changes length of the Refs
			return
		}

		i--     // current absolute index
		shift-- // the shift (to pass the first condition in the very loop)

		// no changes required, continue
	}

	return // done
}

// descendNodeBranches iterate over elements of the node
// starting from the element with absoulute index i
func (r *Refs) descendNodeBranches(
	pack Pack, //               : pack to load
	rn *refsNode, //            : the node
	depth int, //               : depth of the node
	shift int, //               : index of ending element of the node
	i int, //                   : index of element to start from
	descendFunc IterateFunc, // : the function
) (
	pass int, //                : processed elements
	rewind bool, //             : have to find next element from root
	err error, //               : error if any
) {

	var subpass int // pass for a subtree (for a branch)

	for k := len(rn.branches) - 1; k >= 0; k-- {

		var br = rn.branches[k] // current branch

		if err = r.loadNodeIfNeed(pack, br, depth-1); err != nil {
			return
		}

		if shift-br.length > i {
			shift -= br.length
			continue // skip this branch to find i-th element
		}

		subpass, rewind, err = r.descendNode(pack, br, depth-1, shift, i,
			descendFunc)

		pass += subpass // process elements

		if err != nil || rewind == true {
			return // some error or changes
		}

		// these are local variables and decrementing has
		// meaning only if we a going to descend next branch

		shift -= subpass // e.g. shift -= br.length
		i -= subpass     // decrement current absolute index

		// continue
	}

	return // done
}

//
// append creating slice
//

// appendCreatingSliceNodeGoUp goes up and creates
// lack nodes; see also appendCreatingSliceNode and
// appendCreatingSliceFunc
func (r *Refs) appendCreatingSliceNodeGoUp(
	rn *refsNode, //       : current node (full)
	depth int, //          : depth of the current node
	hash cipher.SHA256, // : hash to append
) (
	cn *refsNode, //       : current node (can be another then the rn)
	cdepth int, //         : current depth (depth of the cn)
) {

	for {

		// the rn is full, since we have to go up;
		// thus we have to check upper node fullness
		// and (1) add new branch or go up and repeat
		// the step (1)

		if rn.upper == nil {
			// since, we have to add a new hash to the slice (to the r),
			// then the full rn (and the rn is full here) must have
			// upper node to add another one go or upper; e.g. if
			// the rn.upper is nil, then this case is invalid and
			// this case should produce panicing
			panic("invalid case")
		}

		rn, depth = rn.upper, depth+1 // go up

		// since we are using r.upper, then the depth is > 0
		// and the rn.upper contains branches
		if len(rn.branches) == r.degree { // if it's full
			continue // go up
		}

		// otherwise we create new branch and use it;
		// fields hash and length are not set and mods
		// like contnetMod are not set too

		// e.g. one step down is here

		return r.appendCreatingSliceNode(rn, depth, hash)

	}

}

// appendCreatingSliceNode appends given hash to the Refs
// that is new slice; the method doesn't set 'length'
// and 'hash' fields for refsNode(s); the method returns
// new current node with it depth to make caler able to
// use them
func (r *Refs) appendCreatingSliceNode(
	rn *refsNode, //       : current node
	depth int, //          : depth of the current node
	hash cipher.SHA256, // : hash to append
) (
	cn *refsNode, //       : current node (can be another then the rn)
	cdepth int, //         : current depth (depth of the cn)
) {

	if depth == 0 { // leafs

		if len(rn.leafs) == r.degree { // full
			return r.appendCreatingSliceNodeGoUp(rn, depth, hash) // go up
		}

		var el = &refsElement{
			Hash:  hash,
			upper: rn,
		}

		if r.flags&HashTableIndex != 0 {
			r.addElementToIndex(el)
		}

		rn.leafs = append(rn.leafs, el)

		return rn, depth // the same
	}

	// else if depth > 0 { branches }

	// if the depth is not 1, then we should to add new branch and use it;
	// fields length, hash and mods are still lempty;

	// one more time, if we are here, then the rn has enouth place to
	// fit new brnahc and the branch should be created and used;
	// e.g. the step is 'go down', the step is opposite to the 'go up'

	var br = &refsNode{
		upper: rn,
	}

	rn.branches = append(rn.branches, br)

	return r.appendCreatingSliceNode(br, depth-1, hash)
}

//
// walk updating slice
//

// walkUpdatingSliceNode walks from given node
// through subtree setting actual length and hash
// fields and setting loadedMod flag; the method
// used after creating new Refs
func (r *Refs) walkUpdatingSliceNode(
	pack Pack, //    : pack to save
	rn *refsNode, // : the node to walk from
	depth int, //    : depth of the node
) (
	err error, //    : error if any
) {

	if depth == 0 {

		rn.length = len(rn.leafs) // it's just length of the array

	} else { // depth > 0

		var length int // use local variable

		for _, br := range rn.branches {

			if err = r.walkUpdatingSliceNode(pack, br, depth-1); err != nil {
				return // some error
			}

			length += br.length

		}

		rn.length = length // set actual length

	}

	rn.mods |= loadedMod // mark as loaded

	return rn.updateHash(pack, depth) // update hash
}

//
// free space on tail
//

// freeSpaceOnTailNode finds free space
// on tail of the node
func (r *Refs) freeSpaceOnTailNode(
	pack Pack, //    : pack to load
	rn *refsNode, // : the node
	depth int, //    : depth of the node
) (
	fsotn int, //    : free space on tail of the node
	err error, //    : error if any
) {

	if depth == 0 {

		fsotn = r.degree - len(r.leafs)
		return

	}

	// else if depth > 0

	// the fsotn = (degree ^ (depth+1)) * (degree - len(rn.branches)) +
	//         r.freeSpaceOnTailNode() of the last node in the branhces

	fsotn = pow(r.degree, depth+1) * (r.degree - len(rn.branches))

	if len(rn.branches) == 0 {
		return // done (no branches to check out the last)
	}

	var last = rn.branches[len(rn.branches)-1] // the last branch

	// load the last if need
	if err = r.loadNodeIfNeed(pack, last, depth-1); err != nil {
		return
	}

	var fsotnl int // fsotn of the last

	if fsotnl, err = r.freeSpaceOnTailNode(pack, last, depth-1); err != nil {
		return
	}

	fsotn += fsotnl

	return // done

}

//
// append to non-fresh Refs
//

// appendNodeGoUp goes up and creates lack
// nodes; see also appendNode and appendFunc
func (r *Refs) appendNodeGoUp(
	rn *refsNode, //       : current node (full)
	depth int, //          : depth of the current node
	hash cipher.SHA256, // : hash to append
) (
	cn *refsNode, //       : current node (can be another then the rn)
	cdepth int, //         : current depth (depth of the cn)
) {

	// the rn is full, since we have to go up;
	// thus we have to check upper node fullness
	// and (1) add new branch or go up and repeat
	// the step (1)

	if rn.upper == nil {
		// since, we have to add a new hash to the slice (to the r),
		// then the full rn (and the rn is full here) must have
		// upper node to add another one go or upper; e.g. if
		// the rn.upper is nil, then this case is invalid and
		// this case should produce panicing
		panic("invalid case")
	}

	cn, cdepth = rn.upper, depth+1 // go up

	// since we are using r,upper, then the depth is > 0
	// and the rn.upper contains branches
	if len(cn.branches) == r.degree { // if it's full
		return r.appendNodeGoUp(cn, cdepth, hash) // go up
	}

	// otherwise we create new branch and use it;
	// fields hash and length are not set and mods
	// like contnetMod are not set too

	// e.g. one step down is here

	var br = &refsNode{
		upper: rn,
		mods:  loadedMod | contentMod | lengthMod, // flags
	}

	cn.branches = append(cn.branches, br)

	return br, depth // e.g. br, and cdepth - 1
}

// appendNode appends given hash to the Refs; the method
// doesn't set 'length' and 'hash' fields for refsNode(s);
// the method returns new current node with it depth to
// make caler able to use them; the method sets flags
// loadedMod, contentMod and lengthMod to all touched
// nodes; the lengthMod flag is set only if necesssary
func (r *Refs) appendNode(
	rn *refsNode, //       : current node
	depth int, //          : depth of the current node
	hash cipher.SHA256, // : hash to append
) (
	cn *refsNode, //       : current node (can be another then the rn)
	cdepth int, //         : current depth (depth of the cn)
) {

	if depth == 0 { // leafs

		if len(rn.leafs) == r.degree { // full
			return r.appendNodeGoUp(rn, depth, hash) // go up
		}

		var el = &refsElement{
			Hash:  hash,
			upper: rn,
		}

		if r.flags&HashTableIndex != 0 {
			r.addElementToIndex(el)
		}

		rn.leafs = append(rn.leafs, el)
		rn.length++           // add
		rn.mods |= contentMod // but length is actual

		return rn, depth // the same
	}

	// else if depth > 0 { branches }

	// if the depth is not 1, then we should to add new branch and use it;
	// fields length, hash and mods are still lempty;

	// one more time, if we are here, then the rn has enouth place to
	// fit new brnahc and the branch should be created and used;
	// e.g. the step is 'go down', the step is opposite to the 'go up'

	var br = &refsNode{
		upper: rn,
		mods:  loadedMod | contentMod | lengthMod,
	}

	rn.branches = append(rn.branches, br)

	return r.appendNode(br, depth-1, hash)
}

//
// walk updating
//

// walkUpdatingNode walks from given node
// through subtree setting actual length and hash
// fields; the method used after appending to an
// existing Refs
func (r *Refs) walkUpdatingNode(
	pack Pack, //    : pack to save
	rn *refsNode, // : the node to walk from
	depth int, //    : depth of the node
) (
	err error, //    : error if any
) {

	if rn.mods&(lengthMod|loadedMod|contentMod) == 0 {
		return // the node in actual state
	}

	// length of a node that contains leafs is already in actual state

	if depth > 0 {

		var length int // use local variable

		for _, br := range rn.branches {

			if err = r.walkUpdatingNode(pack, br, depth-1); err != nil {
				return // some error
			}

			length += br.length

		}

		rn.length = length // set actual length

	}

	return rn.updateHash(pack, depth) // update hash
}
