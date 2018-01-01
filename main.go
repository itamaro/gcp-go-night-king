// -*- coding: utf-8 -*-
// Copyright 2018 Itamar Ostricher
// The Night King GCE instance resurrection service

package main

import (
	"log"
	"cloud.google.com/go/pubsub"
	"encoding/json"
	"errors"
	"flag"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"time"
)

type gceInstanceInfo struct {
	Name string
	Zone string
}

func main() {
	projectID := flag.String("project", "", "GCE Project ID")
	subscriptionName := flag.String("subscription-name", "night-king-preempt",
	                                "Name of Pub/Sub subscription name to listen to")
	flag.Parse()
	if *projectID == "" {
		log.Fatalf("Mandatory flag `-project` missing")
	}
	ctx := context.Background()
	// Create a Pub/Sub client and get a subscription reference.
	pubsubClient, err := pubsub.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatalf("Failed to create pubsub client: %v", err)
	}
	sub := pubsubClient.Subscription(*subscriptionName)
	// Create GCE compute client
	computeClient, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		log.Fatalf("Failed to create compute client: %v", err)
	}
	computeService, err := compute.New(computeClient)
	if err != nil {
		log.Fatalf("Failed to create compute service: %v", err)
	}
	// Start listening and handling messages
	handleNightKingMessages(sub, *projectID, computeService)
}

func handleNightKingMessages(sub *pubsub.Subscription, projectID string,
	                         computeService *compute.Service) {
	log.Printf("Listening for messages on subscription: %s", sub.String())
	err := sub.Receive(context.Background(), func(ctx context.Context, m *pubsub.Message) {
		handleNightKingMessage(m.Data, sub.String(), projectID, computeService)
		log.Printf("ACKing meesage\n%s", m.Data)
		m.Ack()
	})
	if err != nil {
		log.Fatalf("Failed to start listening on subscription: %v", err)
	}
}

func handleNightKingMessage(message []byte, subscriptionName string,
	                        projectID string, computeService *compute.Service) {
	log.Printf("Handling message from subscription '%s'", subscriptionName)
	instanceInfo, err := parseNightKingMessage(message)
	if err != nil {
		log.Printf("Failed parsing message - ignoring:\n%s", message)
		return
	}
	resurrectInstance(instanceInfo, projectID, computeService)
}

// parseNightKingMessage parses a JSON-formatted message and returns the parsed
// gceInstanceInfo struct if the message is valid (syntax & structure).
func parseNightKingMessage(message []byte) (parsed gceInstanceInfo, err error) {
	err = json.Unmarshal(message, &parsed)
	if parsed.Name == "" {
		err = errors.New("Missing mandatory field: name")
	}
	if parsed.Zone == "" {
		err = errors.New("Missing mandatory field: zone")
	}
	return
}

func resurrectInstance(instanceInfo gceInstanceInfo, projectID string,
	                   computeService *compute.Service) {
	zone, instance := instanceInfo.Zone, instanceInfo.Name
	keepTrying := true
	for keepTrying {
		instanceStatus, err := getInstanceStatus(projectID, zone, instance, computeService)
		if err != nil {
			log.Printf("No instance '%s' in zone '%s'", instance, zone)
			return
		}
		keepTrying = false
		switch instanceStatus {
		case "STOPPING":
			log.Printf("Instance %s/%s is being stopped - waiting...", zone, instance)
			keepTrying = true
			time.Sleep(30 * time.Second)
		case "TERMINATED":
			log.Printf("Attemping to start instance %s/%s", zone, instance)
			startInstance(projectID, zone, instance, computeService)
		default:
			log.Printf("Instance %s/%s not terminated - ignoring", zone, instance)
		}
	}
}

func getInstanceStatus(projectID string, zone string, instance string,
	                   computeService *compute.Service) (string, error) {
	resp, err := computeService.Instances.Get(projectID, zone, instance).Do()
	if err != nil {
		return "", err
	}
	return resp.Status, nil
}

func startInstance(projectID string, zone string, instance string,
	               computeService *compute.Service) error {
	_, err := computeService.Instances.Start(projectID, zone, instance).Do()
	return err
}
