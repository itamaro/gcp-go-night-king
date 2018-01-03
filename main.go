// -*- coding: utf-8 -*-
// Copyright 2018 Itamar Ostricher
// The Night King GCE instance resurrection service

package main

import (
	"log"
	"flag"
)

func main() {
	projectID := flag.String("project", "", "GCE Project ID")
	subscriptionName := flag.String("subscription-name", "night-king-preempt",
	                                "Name of Pub/Sub subscription name to listen to")
	flag.Parse()
	if *projectID == "" {
		log.Fatalf("Mandatory flag `-project` missing")
	}
	// Create and initialize NightKing service instance
	nk := &nightKing{*projectID, *subscriptionName, nil, nil}
	nk.Init()
	// Start listening and handling messages
	nk.HandleMessages()
}
