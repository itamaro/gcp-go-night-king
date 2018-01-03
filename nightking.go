// -*- coding: utf-8 -*-
// Copyright 2018 Itamar Ostricher
// The Night King GCE instance resurrection service

package main

import (
	"log"
	"cloud.google.com/go/pubsub"
	"encoding/json"
	"errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"time"
)

type gceInstanceInfo struct {
	Name string
	Zone string
}

type nightKing struct {
	ProjectID string
	SubscriptionName string
	Subscription *pubsub.Subscription
	ComputeService *compute.Service
}

// Init initializes a NightKing service instance by creating a PubSub subscription
// and a GCE Compute API service for the instance ProjectID and SubscriptionName.
func (nk *nightKing) Init() {
	ctx := context.Background()
	// Create a Pub/Sub client and get a subscription reference.
	pubsubClient, err := pubsub.NewClient(ctx, nk.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create pubsub client: %v", err)
	}
	nk.Subscription = pubsubClient.Subscription(nk.SubscriptionName)
	// Create GCE compute client & API service
	computeClient, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		log.Fatalf("Failed to create compute client: %v", err)
	}
	nk.ComputeService, err = compute.New(computeClient)
	if err != nil {
		log.Fatalf("Failed to create compute service: %v", err)
	}
}

// HandleMessages goes into PubSub subscription listening mode,
// processing incoming messages from the subscription, forever.
func (nk nightKing) HandleMessages() {
	log.Printf("Listening for messages on subscription: %s", nk.Subscription.String())
	err := nk.Subscription.Receive(
			context.Background(), func(ctx context.Context, m *pubsub.Message) {
		nk.handleMessage(m.Data)
		log.Printf("ACKing meesage\n%s", m.Data)
		m.Ack()
	})
	if err != nil {
		log.Fatalf("Failed to start listening on subscription: %v", err)
	}
}

func (nk nightKing) handleMessage(message []byte) {
	log.Printf("Handling message from subscription '%s'", nk.SubscriptionName)
	instanceInfo, err := nk.parseMessage(message)
	if err != nil {
		log.Printf("Failed parsing message - ignoring:\n%s", message)
		return
	}
	if nk.waitForInstanceTermination(instanceInfo.Zone, instanceInfo.Name) {
		nk.resurrectInstance(instanceInfo.Zone, instanceInfo.Name)
	}
}

// parsenightKingMessage parses a JSON-formatted message and returns the parsed
// gceInstanceInfo struct if the message is valid (syntax & structure).
func (nk nightKing) parseMessage(message []byte) (parsed gceInstanceInfo, err error) {
	err = json.Unmarshal(message, &parsed)
	if parsed.Name == "" {
		err = errors.New("Missing mandatory field: name")
	}
	if parsed.Zone == "" {
		err = errors.New("Missing mandatory field: zone")
	}
	return
}

func (nk nightKing) waitForInstanceTermination(zone string, instance string) bool {
	stillRunningCount := 0
	for {
		instanceStatus, err := nk.getInstanceStatus(zone, instance)
		if err != nil {
			log.Printf("No instance '%s' in zone '%s'", instance, zone)
			return false
		}
		switch instanceStatus {
		case "TERMINATED":
			log.Printf("Instance %s/%s terminated", zone, instance)
			return true
		case "STOPPING":
			log.Printf("Instance %s/%s is stopping - waiting for termination", zone, instance)
			time.Sleep(10 * time.Second)
		case "RUNNING":
			stillRunningCount++
			if stillRunningCount > 6 {
				log.Printf("Instance %s/%s has been running for the last 3 minutes - " +
					"assuming it's not about to terminate", zone, instance)
				return false
			}
			log.Printf("Instance %s/%s still running - waiting for its termination", zone, instance)
			time.Sleep(30 * time.Second)
		default:
			log.Printf("Not sure how to handle instance %s/%s status: %s -- ignoring",
				zone, instance, instanceStatus)
		}
	}
}

func (nk nightKing) resurrectInstance(zone string, instance string) {
	log.Printf("Attemping to start instance %s/%s", zone, instance)
	err := nk.startInstance(zone, instance)
	if (err != nil) {
		log.Printf("Error in start instance %s/%s API call: %v", zone, instance, err)
	}
}

func (nk nightKing) getInstanceStatus(zone string, instance string) (string, error) {
	resp, err := nk.ComputeService.Instances.Get(nk.ProjectID, zone, instance).Do()
	if err != nil {
		return "", err
	}
	return resp.Status, nil
}

func (nk nightKing) startInstance(zone string, instance string) error {
	_, err := nk.ComputeService.Instances.Start(nk.ProjectID, zone, instance).Do()
	return err
}
