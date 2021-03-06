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

package visitor

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/bbva/qed/balloon/navigator"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
)

var fakePos = navigator.NewFakePosition

func TestCollect(t *testing.T) {

	testCases := []struct {
		visitable         Visitable
		expectedMutations []*storage.Mutation
	}{
		{
			visitable: NewRoot(fakePos([]byte{0}, 8),
				NewNode(fakePos([]byte{0}, 9),
					NewCollectable(NewCached(fakePos([]byte{0}, 7), hashing.Digest{0})),
					NewNode(fakePos([]byte{0}, 9),
						NewCollectable(NewCached(fakePos([]byte{0}, 6), hashing.Digest{1})),
						NewLeaf(fakePos([]byte{0}, 1), hashing.Digest{0}),
					),
				),
				NewNode(fakePos([]byte{0}, 9),
					NewLeaf(fakePos([]byte{0}, 1), hashing.Digest{0}),
					NewCollectable(NewCached(fakePos([]byte{0}, 8), hashing.Digest{2})),
				),
			),
			expectedMutations: []*storage.Mutation{
				{storage.HyperCachePrefix, fakePos([]byte{0}, 7).Bytes(), hashing.Digest{0}},
				{storage.HyperCachePrefix, fakePos([]byte{0}, 6).Bytes(), hashing.Digest{1}},
				{storage.HyperCachePrefix, fakePos([]byte{0}, 8).Bytes(), hashing.Digest{2}},
			},
		},
	}

	for i, c := range testCases {
		decorated := NewComputeHashVisitor(hashing.NewFakeXorHasher())
		visitor := NewCollectMutationsVisitor(decorated, storage.HyperCachePrefix)
		c.visitable.PostOrder(visitor)

		mutations := visitor.Result()
		assert.ElementsMatchf(
			t,
			mutations,
			c.expectedMutations,
			"Mutation error in test case %d",
			i,
		)
	}
}
