// -*- coding: utf-8 -*-
// Copyright 2018 Itamar Ostricher

package main

import "testing"

func TestGoodMessageParsing(t *testing.T) {
	nk := new(nightKing)
	parsed, err := nk.parseMessage([]byte(`
		{
			"name": "foo",
			"zone": "bar"
		}
	`))
	if err != nil {
		t.Fatal("Unexpected parsing error:", err)
	}
	if parsed.Name != "foo" {
		t.Error("Expected name foo, Got", parsed.Name)
	}
	if parsed.Zone != "bar" {
		t.Error("Expected zone bar, Got", parsed.Zone)
	}
}

func TestBadMessageParsing(t *testing.T) {
	nk := new(nightKing)
	_, err := nk.parseMessage([]byte(`
		{
			"foo": "bar"
		}
	`))
	if err == nil {
		t.Fatal("Expected parsing to fail, but it was successful")
	}
}

func TestBrokenMessageParsing(t *testing.T) {
	nk := new(nightKing)
	_, err := nk.parseMessage([]byte("foo"))
	if err == nil {
		t.Fatal("Expected parsing to fail, but it was successful")
	}
}
