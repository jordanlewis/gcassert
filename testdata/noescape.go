// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package gcassert

type foo struct {
	a int
	b int
}

func returnsStackVarPtr() *foo {
	// this should fail
	//gcassert:noescape
	foo := foo{a: 1, b:2}
	return &foo
}

func returnsStackVar() foo {
	// this should succeed
	//gcassert:noescape
	foo := foo{a: 1, b:2}
	return foo
}
