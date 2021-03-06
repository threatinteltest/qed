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

package hyper

import (
	"testing"

	"github.com/bbva/qed/balloon/navigator"
	"github.com/stretchr/testify/require"
)

func TestRoot(t *testing.T) {

	testCases := []struct {
		numBits     uint16
		expectedPos navigator.Position
	}{
		{8, NewPosition(make([]byte, 1), 8)},
		{256, NewPosition(make([]byte, 32), 256)},
	}

	for i, c := range testCases {
		navigator := NewHyperTreeNavigator(c.numBits)
		rootPos := navigator.Root()
		require.Equalf(t, c.expectedPos, rootPos, "The root position should match in test case %d", i)
	}

}

func TestIsLeaf(t *testing.T) {

	testCases := []struct {
		position navigator.Position
		ok       bool
	}{
		{NewPosition([]byte{0}, 0), true},
		{NewPosition([]byte{0}, 1), false},
		{NewPosition([]byte{0}, 7), false},
	}

	navigator := NewHyperTreeNavigator(8)
	for i, c := range testCases {
		result := navigator.IsLeaf(c.position)
		require.Equalf(t, c.ok, result, "The leaf checking should match for test case %d", i)
	}

}

func TestIsRoot(t *testing.T) {

	testCases := []struct {
		numBits  uint16
		position navigator.Position
		ok       bool
	}{
		{8, NewPosition([]byte{0}, 8), true},
		{8, NewPosition([]byte{0}, 1), false},
		{256, NewPosition([]byte{0}, 256), true},
		{256, NewPosition([]byte{0}, 56), false},
	}

	for i, c := range testCases {
		navigator := NewHyperTreeNavigator(c.numBits)
		result := navigator.IsRoot(c.position)
		require.Equalf(t, c.ok, result, "The root checking should match for test case %d", i)
	}
}

func TestGoToLeft(t *testing.T) {

	testCases := []struct {
		numBits      uint16
		position     navigator.Position
		expectedLeft navigator.Position
	}{
		{8, NewPosition([]byte{0}, 0), nil},
		{8, NewPosition([]byte{0}, 1), NewPosition([]byte{0}, 0)},
		{8, NewPosition([]byte{4}, 1), NewPosition([]byte{4}, 0)},
	}

	for i, c := range testCases {
		navigator := NewHyperTreeNavigator(c.numBits)
		left := navigator.GoToLeft(c.position)
		require.Equalf(t, c.expectedLeft, left, "The left positions should match for test case %d", i)
	}
}

func TestGoToRight(t *testing.T) {

	testCases := []struct {
		numBits       uint16
		position      navigator.Position
		expectedRight navigator.Position
	}{
		{8, NewPosition([]byte{0}, 0), nil},
		{8, NewPosition([]byte{0}, 1), NewPosition([]byte{1}, 0)},
	}

	for i, c := range testCases {
		navigator := NewHyperTreeNavigator(c.numBits)
		right := navigator.GoToRight(c.position)
		require.Equalf(t, c.expectedRight, right, "The right positions should match for test case %d", i)
	}
}

func TestDescendToFirst(t *testing.T) {

	testCases := []struct {
		numBits       uint16
		position      navigator.Position
		expectedFirst navigator.Position
	}{
		{8, NewPosition([]byte{0}, 0), NewPosition([]byte{0}, 0)},
		{8, NewPosition([]byte{4}, 4), NewPosition([]byte{4}, 0)},
	}

	for i, c := range testCases {
		navigator := NewHyperTreeNavigator(c.numBits)
		first := navigator.DescendToFirst(c.position)
		require.Equalf(t, c.expectedFirst, first, "The first positions should match for test case %d", i)
	}
}

func TestDescendToLast(t *testing.T) {

	testCases := []struct {
		numBits      uint16
		position     navigator.Position
		expectedLast navigator.Position
	}{
		{8, NewPosition([]byte{0}, 0), NewPosition([]byte{0}, 0)},
		{8, NewPosition([]byte{0}, 1), NewPosition([]byte{1}, 0)},
		{8, NewPosition([]byte{0}, 2), NewPosition([]byte{3}, 0)},
		{8, NewPosition([]byte{0}, 3), NewPosition([]byte{7}, 0)},
	}

	for i, c := range testCases {
		navigator := NewHyperTreeNavigator(c.numBits)
		last := navigator.DescendToLast(c.position)
		require.Equalf(t, c.expectedLast, last, "The last positions should match for test case %d", i)
	}
}
