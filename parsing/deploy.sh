#!/usr/bin/env bash

set -e

# Turn colors in this script off by setting the NO_COLOR variable in your
# environment to any value:

NO_COLOR=${NO_COLOR:-""}
if [ -z "$NO_COLOR" ]; then
  header=$'\e[1;33m'
  reset=$'\e[0m'
else
  header=''
  reset=''
fi

function header_text {
  echo "$header$*$reset"
}


YAML_URL="https://api.kscout.io/apps/{{app.id}}/deploy/parameterized-deployment.yaml"
CONFIG_URL="https://api.kscout.io/apps/{{app.id}}/deploy/parameters"

header_text "Installing Serverless App"

header_text "Setting Up Your Namespace"
read -p "Please Enter Namespace: " namespace
echo # new line

header_text "Setting Up Configuration"
SED_DATA="s/data/data/ "

while read -r line <&9
do
    ID=$(echo $line | awk '{print $1}')
    KEY=$(echo $line | awk '{print $2}')
    DFLT=$(echo $line | awk '{print $3}')
    BASE64=$(echo $line | awk '{print $4}')

    echo  #new line
    echo "Default Value for $KEY is '$DFLT'"
    read -p "Do you want to change it ? (y/n): " choice

    case "$choice" in
      y|Y|yes|YES|Yes )
        read -p "Enter new value for $KEY : " value
        if [[ "$BASE64" == "Y" ]]
        then
            value=$(echo "${value}" | base64)
        else
            value="${value}"
        fi
        SED_DATA="$SED_DATA ; s/$ID/$value/" ;;
      n|N|no|NO|No )
        if [[ "$BASE64" == "Y" ]]
        then
            DFLT="${value}"
        else
            DFLT=$(echo "${DFLT}" | base64 -d)
        fi
        SED_DATA="$SED_DATA ; s/$ID/$DFLT/";;
      * ) echo "invalid input, Please run the script again";;
    esac
done 9<<< "$(curl '$CONFIG_URL')"

echo # new line

header_text "Downloading data and Applying your Configuration"

curl -L "${YAML_URL}" \
      | sed "${SED_DATA}" \
      | kubectl -n $namespace apply -f -

