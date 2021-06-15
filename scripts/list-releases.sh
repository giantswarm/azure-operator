 #!/bin/bash

if [ "$1" = "" ]; then
    echo "Error: please provide provider parameter"
    echo "./list-releases {provider} {release_active_tag(optional)}"
    exit 1
fi


provider=$1
active_release=$2
for installation in $(opsctl list installations --provider=$provider --short); do
		gsctl select endpoint "${installation[0]}"
		if [ "$active_release" = "" ]	
		then
			gsctl list releases
		else
			gsctl list releases > list-of-releases.txt
			while read releases; do
				IFS=' ' read -ra release_data <<< "$releases"
				if [ "${release_data[0]}" == "$active_release" ]
			    then
			    	echo "Release $active_release is ${release_data[1]}"
			    fi
		done <list-of-releases.txt
  		fi
done
rm -rf list-of-releases.txt
