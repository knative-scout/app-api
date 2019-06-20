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


YAML_JSON=$(cat <<-END
          {{{yaml.file}}}
END
)

header_text "Installing Serverless App"

header_text "Setting Up Your Namespace"
read -p "Please Enter Namespace: " namespace
echo # new line

header_text "Setting Up Configuration"
SED_DATA="s/data/data/ "

{{{replacement.script}}}

echo # new line

header_text "Applying your Configuration"
data=$(sed "${SED_DATA}" <<< $YAML_JSON)


echo $data |kubectl -n $namespace apply -f -

