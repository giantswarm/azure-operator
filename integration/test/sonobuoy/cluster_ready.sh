#!/usr/bin/env bash

echo "Waiting for tenant cluster..."
sleep 900

count=0
while [ $count -lt 2 ]
do
  count=$(kubectl --context="giantswarm-${CLUSTER_ID}" get no|grep worker|grep  -v "NotReady"|wc -l)
  if [ "${count}" -lt 2 ]
  then
    sleep 5
    echo "Found $count ready nodes"
  fi
done
