#! /bin/bash
. .env
. common.sh

function get_files() {
    if [ ! -e $FILES_OUTPUT ]; then
      mkdir -p $FILES_OUTPUT
    fi
    url=$1
    file_name=$(echo ${url##*/})
    if [ ! -e $FILES_OUTPUT/$file_name ]; then
      echo "==> Download $url"
      for i in {1..3}; do
          curl --location --show-error --fail --output $FILES_OUTPUT/$file_name $url && return
          echo "curl failed. Attempt=$i"
      done
      echo "Download failed, exit : $url"
      exit 1
    else
      echo "==> Skip $url"
    fi
}

function download_files() {
  if [ ! -e $FILE_LIST ]; then
    generate_list
  fi
  files=$(cat $FILE_LIST)
  for file in $files;do
    get_files $file
  done
}

function remove_files() {
  rm -rf $FILES_OUTPUT
}