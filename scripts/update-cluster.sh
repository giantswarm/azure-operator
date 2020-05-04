 #!/bin/bash

if [ "$1" = "" ] || [ "$2" = "" ] || [ "$3" = "" ]; then
    echo "Error: not enough arguments."
    echo "./update-cluster {installation} {cluster_id} {version}"
    exit 1
fi

INSTALLATION=$1
CLUSTER=$2
VERSION=$3
gsctl select endpoint ${INSTALLATION}


# refresh token
gsctl list releases >/dev/null 2>/dev/null
API=`gsctl info -v | grep "API endpoint:" | awk '{print $3}'`
TOKEN=`gsctl info -v | grep "Auth token:" | awk '{print $3}'`

curl -s ${API}/v4/clusters/${CLUSTER}/ -H "Authorization: Bearer ${TOKEN}" -X PATCH -d "{ \"release_version\": \"${VERSION}\" }"
