#!/bin/bash

#
# Example usage:
#   flowchart.sh azure-operator-master --tenant-cluster xyz99 --open-browser
#
# where 'azure-operator-master' is App CR name and 'xyz99' is tenant cluster ID
# and '--open-browser' flag indicates that a default browser will open the
# generated flowchart.
#

# check if required CLI tools are installed
for required in kubectl jq
do
  if [ ! -x "$(command -v $required)" ]
  then
    echo "[err] The required command $required was not found in your system. Aborting."
    exit 1
  fi
done

args=()

while [[ $# -gt 0 ]]; do
    flag="$1"
    case $flag in
        -b|--open-browser)
            flag_open_browser=1
            shift
            ;;
        -t|--tenant-cluster)
            tenant_cluster_id="$2"
            shift
            shift
            ;;
        *)
            args+=("$1")
            shift
            ;;
    esac
done

# restore positional arguments
set -- "${args[@]}"
azure_operator_app=$1

if [ -z "$azure_operator_app" ]; then
    echo [err] azure-operator CR name must be specified
    exit 1
fi

# get logs
logfile="/tmp/${azure_operator_app}.logs"
if ! kubectl -n giantswarm logs deployment/${azure_operator_app} > "${logfile}"; then
    echo "[err] azure-operator app '$azure_operator_app' not found"
    exit 1
fi

# filter by event message, get only state change events
query='. | select(.message | test("state changed")) '

# filter by tenant cluster ID
if ! [ -z "$tenant_cluster_id" ]; then
    query+="| select(.object | endswith(\"/azureconfigs/$tenant_cluster_id\")) "
fi

# echo state transition in format 'stateX --> stateY'
query+='| "    " +
    (if (.oldState | length) > 0 then .oldState else "DeploymentUninitialized" end) +
    " --> " +
    (if (.newState | length) > 0 then .newState else "DeploymentUninitialized" end)'

# idented transition lines: "    stateX --> stateY"
transitions=$(cat "${logfile}" \
    | jq -r "$query" \
    | sort \
    | uniq)

mermaid="graph TD
${transitions}"

script_dir="$( cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd )"
template=$(cat "${script_dir}/flowchart.template.html")
generated_flowchart="${azure_operator_app}-flowchart.html"

# generate flowchart
echo "${template/_FLOWCHART_DATA_/$mermaid}" > "${generated_flowchart}"
echo "Generated flowchart in file '${generated_flowchart}'."

# open default browser
if [ "$flag_open_browser" == 1 ]; then
    if which xdg-open > /dev/null; then
        xdg-open "${generated_flowchart}"
    elif which gnome-open > /dev/null; then
        gnome-open "${generated_flowchart}"
    fi
fi
