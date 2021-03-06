/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, S.A.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package history

import (
	"testing"

	"github.com/bbva/qed/balloon/cache"
	"github.com/bbva/qed/balloon/navigator"
	"github.com/bbva/qed/balloon/visitor"
	"github.com/bbva/qed/hashing"
	"github.com/stretchr/testify/assert"
)

func pos(index uint64, height uint16) navigator.Position {
	return NewPosition(index, height)
}

func root(pos navigator.Position, left, right visitor.Visitable) *visitor.Root {
	return visitor.NewRoot(pos, left, right)
}

func node(pos navigator.Position, left, right visitor.Visitable) *visitor.Node {
	return visitor.NewNode(pos, left, right)
}

func partialnode(pos navigator.Position, left visitor.Visitable) *visitor.PartialNode {
	return visitor.NewPartialNode(pos, left)
}

func leaf(pos navigator.Position, value byte) *visitor.Leaf {
	return visitor.NewLeaf(pos, []byte{value})
}

func leafnil(pos navigator.Position) *visitor.Leaf {
	return visitor.NewLeaf(pos, nil)
}

func cached(pos navigator.Position) *visitor.Cached {
	return visitor.NewCached(pos, hashing.Digest{0})
}

func collectable(underlying visitor.Visitable) *visitor.Collectable {
	return visitor.NewCollectable(underlying)
}

func cacheable(underlying visitor.Visitable) *visitor.Cacheable {
	return visitor.NewCacheable(underlying)
}

func TestInsertPruner(t *testing.T) {

	cache := cache.NewFakeCache(hashing.Digest{0x0})

	testCases := []struct {
		version        uint64
		eventDigest    hashing.Digest
		expectedPruned visitor.Visitable
	}{
		{
			version:        0,
			eventDigest:    hashing.Digest{0x0},
			expectedPruned: collectable(cacheable(leaf(pos(0, 0), 0))),
		},
		{
			version:     1,
			eventDigest: hashing.Digest{0x1},
			expectedPruned: collectable(cacheable(root(pos(0, 1),
				cached(pos(0, 0)),
				collectable(cacheable(leaf(pos(1, 0), 1))))),
			),
		},
		{
			version:     2,
			eventDigest: hashing.Digest{0x2},
			expectedPruned: root(pos(0, 2),
				cached(pos(0, 1)),
				partialnode(pos(2, 1),
					collectable(cacheable(leaf(pos(2, 0), 2)))),
			),
		},
		{
			version:     3,
			eventDigest: hashing.Digest{0x3},
			expectedPruned: collectable(cacheable(root(pos(0, 2),
				cached(pos(0, 1)),
				collectable(cacheable(node(pos(2, 1),
					cached(pos(2, 0)),
					collectable(cacheable(leaf(pos(3, 0), 3)))))))),
			),
		},
		{
			version:     4,
			eventDigest: hashing.Digest{0x4},
			expectedPruned: root(pos(0, 3),
				cached(pos(0, 2)),
				partialnode(pos(4, 2),
					partialnode(pos(4, 1),
						collectable(cacheable(leaf(pos(4, 0), 4))))),
			),
		},
		{
			version:     5,
			eventDigest: hashing.Digest{0x5},
			expectedPruned: root(pos(0, 3),
				cached(pos(0, 2)),
				partialnode(pos(4, 2),
					collectable(cacheable(node(pos(4, 1),
						cached(pos(4, 0)),
						collectable(cacheable(leaf(pos(5, 0), 5))))))),
			),
		},
		{
			version:     6,
			eventDigest: hashing.Digest{0x6},
			expectedPruned: root(pos(0, 3),
				cached(pos(0, 2)),
				node(pos(4, 2),
					cached(pos(4, 1)),
					partialnode(pos(6, 1),
						collectable(cacheable(leaf(pos(6, 0), 6))))),
			),
		},
		{
			version:     7,
			eventDigest: hashing.Digest{0x7},
			expectedPruned: collectable(cacheable(root(pos(0, 3),
				cached(pos(0, 2)),
				collectable(cacheable(node(pos(4, 2),
					cached(pos(4, 1)),
					collectable(cacheable(node(pos(6, 1),
						cached(pos(6, 0)),
						collectable(cacheable(leaf(pos(7, 0), 7))))))))))),
			),
		},
	}

	for i, c := range testCases {
		context := PruningContext{
			navigator:     NewHistoryTreeNavigator(c.version),
			cacheResolver: NewSingleTargetedCacheResolver(c.version),
			cache:         cache,
		}

		pruned, _ := NewInsertPruner(c.version, c.eventDigest, context).Prune()

		assert.Equalf(t, c.expectedPruned, pruned, "The pruned trees should match for test case %d", i)
	}

}

func TestSearchPruner(t *testing.T) {

	cache := cache.NewFakeCache(hashing.Digest{0x0})

	testCases := []struct {
		version        uint64
		expectedPruned visitor.Visitable
	}{
		{
			version:        0,
			expectedPruned: leafnil(pos(0, 0)),
		},
		{
			version: 1,
			expectedPruned: root(pos(0, 1),
				collectable(cached(pos(0, 0))),
				leafnil(pos(1, 0)),
			),
		},
		{
			version: 2,
			expectedPruned: root(pos(0, 2),
				collectable(cached(pos(0, 1))),
				partialnode(pos(2, 1),
					leafnil(pos(2, 0))),
			),
		},
		{
			version: 3,
			expectedPruned: root(pos(0, 2),
				collectable(cached(pos(0, 1))),
				node(pos(2, 1),
					collectable(cached(pos(2, 0))),
					leafnil(pos(3, 0))),
			),
		},
		{
			version: 4,
			expectedPruned: root(pos(0, 3),
				collectable(cached(pos(0, 2))),
				partialnode(pos(4, 2),
					partialnode(pos(4, 1),
						leafnil(pos(4, 0)))),
			),
		},
		{
			version: 5,
			expectedPruned: root(pos(0, 3),
				collectable(cached(pos(0, 2))),
				partialnode(pos(4, 2),
					node(pos(4, 1),
						collectable(cached(pos(4, 0))),
						leafnil(pos(5, 0)))),
			),
		},
		{
			version: 6,
			expectedPruned: root(pos(0, 3),
				collectable(cached(pos(0, 2))),
				node(pos(4, 2),
					collectable(cached(pos(4, 1))),
					partialnode(pos(6, 1),
						leafnil(pos(6, 0)))),
			),
		},
		{
			version: 7,
			expectedPruned: root(pos(0, 3),
				collectable(cached(pos(0, 2))),
				node(pos(4, 2),
					collectable(cached(pos(4, 1))),
					node(pos(6, 1),
						collectable(cached(pos(6, 0))),
						leafnil(pos(7, 0)))),
			),
		},
	}

	for i, c := range testCases {
		context := PruningContext{
			navigator:     NewHistoryTreeNavigator(c.version),
			cacheResolver: NewSingleTargetedCacheResolver(c.version),
			cache:         cache,
		}

		pruned, _ := NewSearchPruner(context).Prune()

		assert.Equalf(t, c.expectedPruned, pruned, "The pruned trees should match for test case %d", i)
	}

}

func TestSearchPrunerConsistency(t *testing.T) {

	cache := cache.NewFakeCache(hashing.Digest{0x0})

	testCases := []struct {
		index, version uint64
		expectedPruned visitor.Visitable
	}{
		{
			index:          0,
			version:        0,
			expectedPruned: leafnil(pos(0, 0)),
		},
		{
			index:   0,
			version: 1,
			expectedPruned: root(pos(0, 1),
				leafnil(pos(0, 0)),
				collectable(cached(pos(1, 0))),
			),
		},
		{
			index:   0,
			version: 2,
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					leafnil(pos(0, 0)),
					collectable(cached(pos(1, 0)))),
				partialnode(pos(2, 1),
					collectable(cached(pos(2, 0)))),
			),
		},
		{
			index:   0,
			version: 3,
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					leafnil(pos(0, 0)),
					collectable(cached(pos(1, 0)))),
				collectable(cached(pos(2, 1))),
			),
		},
		{
			index:   0,
			version: 4,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						leafnil(pos(0, 0)),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				partialnode(pos(4, 2),
					partialnode(pos(4, 1),
						collectable(cached(pos(4, 0))))),
			),
		},
		{
			index:   0,
			version: 5,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						leafnil(pos(0, 0)),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				partialnode(pos(4, 2),
					collectable(cached(pos(4, 1)))),
			),
		},
		{
			index:   0,
			version: 6,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						leafnil(pos(0, 0)),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				node(pos(4, 2),
					collectable(cached(pos(4, 1))),
					partialnode(pos(6, 1),
						collectable(cached(pos(6, 0))))),
			),
		},
		{
			index:   0,
			version: 7,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						leafnil(pos(0, 0)),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				collectable(cached(pos(4, 2))),
			),
		},
	}

	for i, c := range testCases {
		context := PruningContext{
			navigator:     NewHistoryTreeNavigator(c.version),
			cacheResolver: NewDoubleTargetedCacheResolver(c.index, c.version),
			cache:         cache,
		}

		pruned, _ := NewSearchPruner(context).Prune()

		assert.Equalf(t, c.expectedPruned, pruned, "The pruned trees should match for test case %d", i)
	}

}

func TestSearchPrunerIncremental(t *testing.T) {

	cache := cache.NewFakeCache(hashing.Digest{0x0})

	testCases := []struct {
		start, end     uint64
		expectedPruned visitor.Visitable
	}{
		{
			start:          0,
			end:            0,
			expectedPruned: collectable(cached(pos(0, 0))),
		},
		{
			start: 0,
			end:   1,
			expectedPruned: root(pos(0, 1),
				collectable(cached(pos(0, 0))),
				collectable(cached(pos(1, 0))),
			),
		},
		{
			start: 0,
			end:   2,
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					collectable(cached(pos(0, 0))),
					collectable(cached(pos(1, 0)))),
				partialnode(pos(2, 1),
					collectable(cached(pos(2, 0)))),
			),
		},
		{
			start: 0,
			end:   3,
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					collectable(cached(pos(0, 0))),
					collectable(cached(pos(1, 0)))),
				collectable(cached(pos(2, 1))),
			),
		},
		{
			start: 0,
			end:   4,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						collectable(cached(pos(0, 0))),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				partialnode(pos(4, 2),
					partialnode(pos(4, 1),
						collectable(cached(pos(4, 0))))),
			),
		},
		{
			start: 0,
			end:   5,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						collectable(cached(pos(0, 0))),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				partialnode(pos(4, 2),
					collectable(cached(pos(4, 1)))),
			),
		},
		{
			start: 0,
			end:   6,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						collectable(cached(pos(0, 0))),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				node(pos(4, 2),
					collectable(cached(pos(4, 1))),
					partialnode(pos(6, 1),
						collectable(cached(pos(6, 0))))),
			),
		},
		{
			start: 0,
			end:   7,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						collectable(cached(pos(0, 0))),
						collectable(cached(pos(1, 0)))),
					collectable(cached(pos(2, 1)))),
				collectable(cached(pos(4, 2))),
			),
		},
	}

	for i, c := range testCases {
		context := PruningContext{
			navigator:     NewHistoryTreeNavigator(c.end),
			cacheResolver: NewIncrementalCacheResolver(c.start, c.end),
			cache:         cache,
		}

		pruned, _ := NewSearchPruner(context).Prune()

		assert.Equalf(t, c.expectedPruned, pruned, "The pruned trees should match for test case %d", i)
	}

}

func TestVerifyPruner(t *testing.T) {

	cache := cache.NewFakeCache(hashing.Digest{0x0})

	testCases := []struct {
		index, version uint64
		eventDigest    hashing.Digest
		expectedPruned visitor.Visitable
	}{
		{
			index:          0,
			version:        0,
			eventDigest:    hashing.Digest{0x0},
			expectedPruned: leaf(pos(0, 0), 0),
		},
		{
			index:       0,
			version:     1,
			eventDigest: hashing.Digest{0x0},
			expectedPruned: root(pos(0, 1),
				leaf(pos(0, 0), 0),
				cached(pos(1, 0))),
		},
		{
			index:       1,
			version:     1,
			eventDigest: hashing.Digest{0x1},
			expectedPruned: root(pos(0, 1),
				cached(pos(0, 0)),
				leaf(pos(1, 0), 1)),
		},
		{
			index:       1,
			version:     2,
			eventDigest: hashing.Digest{0x1},
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					cached(pos(0, 0)),
					leaf(pos(1, 0), 1)),
				partialnode(pos(2, 1),
					cached(pos(2, 0)))),
		},
		{
			index:       6,
			version:     6,
			eventDigest: hashing.Digest{0x6},
			expectedPruned: root(pos(0, 3),
				cached(pos(0, 2)),
				node(pos(4, 2),
					cached(pos(4, 1)),
					partialnode(pos(6, 1),
						leaf(pos(6, 0), 6)))),
		},
		{
			index:       1,
			version:     7,
			eventDigest: hashing.Digest{0x1},
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						cached(pos(0, 0)),
						leaf(pos(1, 0), 1)),
					cached(pos(2, 1))),
				cached(pos(4, 2))),
		},
	}

	for i, c := range testCases {

		var cacheResolver CacheResolver
		if c.index == c.version {
			cacheResolver = NewSingleTargetedCacheResolver(c.version)
		} else {
			cacheResolver = NewDoubleTargetedCacheResolver(c.index, c.version)
		}
		context := PruningContext{
			navigator:     NewHistoryTreeNavigator(c.version),
			cacheResolver: cacheResolver,
			cache:         cache,
		}

		pruned, _ := NewVerifyPruner(c.eventDigest, context).Prune()

		assert.Equalf(t, c.expectedPruned, pruned, "The pruned trees should match for test case %d", i)
	}

}

func TestVerifyPrunerIncremental(t *testing.T) {

	cache := cache.NewFakeCache(hashing.Digest{0x0})

	testCases := []struct {
		start, end     uint64
		expectedPruned visitor.Visitable
	}{
		{
			start:          0,
			end:            0,
			expectedPruned: cached(pos(0, 0)),
		},
		{
			start: 0,
			end:   1,
			expectedPruned: root(pos(0, 1),
				cached(pos(0, 0)),
				cached(pos(1, 0)),
			),
		},
		{
			start: 0,
			end:   2,
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					cached(pos(0, 0)),
					cached(pos(1, 0))),
				partialnode(pos(2, 1),
					cached(pos(2, 0))),
			),
		},
		{
			start: 0,
			end:   3,
			expectedPruned: root(pos(0, 2),
				node(pos(0, 1),
					cached(pos(0, 0)),
					cached(pos(1, 0))),
				cached(pos(2, 1)),
			),
		},
		{
			start: 0,
			end:   4,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						cached(pos(0, 0)),
						cached(pos(1, 0))),
					cached(pos(2, 1))),
				partialnode(pos(4, 2),
					partialnode(pos(4, 1),
						cached(pos(4, 0)))),
			),
		},
		{
			start: 0,
			end:   5,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						cached(pos(0, 0)),
						cached(pos(1, 0))),
					cached(pos(2, 1))),
				partialnode(pos(4, 2),
					cached(pos(4, 1))),
			),
		},
		{
			start: 0,
			end:   6,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						cached(pos(0, 0)),
						cached(pos(1, 0))),
					cached(pos(2, 1))),
				node(pos(4, 2),
					cached(pos(4, 1)),
					partialnode(pos(6, 1),
						cached(pos(6, 0)))),
			),
		},
		{
			start: 0,
			end:   7,
			expectedPruned: root(pos(0, 3),
				node(pos(0, 2),
					node(pos(0, 1),
						cached(pos(0, 0)),
						cached(pos(1, 0))),
					cached(pos(2, 1))),
				cached(pos(4, 2)),
			),
		},
	}

	for i, c := range testCases {
		context := PruningContext{
			navigator:     NewHistoryTreeNavigator(c.end),
			cacheResolver: NewIncrementalCacheResolver(c.start, c.end),
			cache:         cache,
		}

		pruned, _ := NewVerifyIncrementalPruner(context).Prune()

		assert.Equalf(t, c.expectedPruned, pruned, "The pruned trees should match for test case %d", i)
	}

}
