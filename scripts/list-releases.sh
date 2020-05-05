 #!/bin/bash

if [ "$1" = "" ]; then
    echo "Error: please provide provider parameter"
    echo "./list-releases {provider} {release_active_tag(optional)}"
    exit 1
fi

opsctl list installations > list-of-installations.txt
gsctl list endpoints > list-of-endpoints.txt

provider=$1
active_release=$2
while read installation; do
  #echo $installation
  #inallation_data=$(echo $installation | tr -d " ")
  IFS=' ' read -ra inallation_data <<< "$installation"
  if [ "${inallation_data[1]}" == "$provider" ]
  then
		echo "-----${inallation_data[0]}-----"
		gsctl select endpoint "${inallation_data[0]}"
		
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
  	
  	#gsctl select endpoint api.g8s."${inallation_data[4]}"
  fi
done <list-of-installations.txt
rm -rf list-of-installations.txt
rm -rf list-of-releases.txt