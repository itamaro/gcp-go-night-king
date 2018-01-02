#!/bin/bash
# -*- coding: utf-8 -*-
# Shutdown script for preemptible instances - ask Night King to resurrect on preemption

META_URL="http://metadata.google.internal/computeMetadata/v1/instance"
TOPIC="night-king-preempt"

get_meta() {
  curl -s "$META_URL/$1" -H "Metadata-Flavor: Google"
}

IS_PREEMPTED="$( get_meta preempted )"
if [ "$IS_PREEMPTED" == "TRUE" ]; then
  NAME="$( get_meta name )"
  ZONE="$( get_meta zone | cut -d '/' -f 4 )"
  gcloud pubsub topics publish "$TOPIC" --message '{"name": "'${NAME}'", "zone": "'${ZONE}'"}'
fi
