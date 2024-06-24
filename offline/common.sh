#! /bin/bash
. .env
. /etc/os-release

function generate_list() {
    LANG=C /bin/bash  $GENERATE_LIST || exit 1
}

function expand_image_repo() {
    local repo="$1"

    if [[ "$repo" =~ ^[a-zA-Z0-9]+: ]]; then
        repo="docker.io/library/$repo"
    elif [[ "$repo" =~ ^[a-zA-Z0-9]+\/ ]]; then
            repo="docker.io/$repo"
    fi
    echo "$repo"
}


function get_image() {
  if [ ! -e $IMAGES_OUTPUT ]; then
    mkdir -p $IMAGES_OUTPUT
  fi
  image=$1
  tar_name=$(echo ${image##*/} | sed s/:/-/g).tar
  if [ ! -e $IMAGES_OUTPUT/tar_name ]; then
    echo "===> Pull $image"
    docker pull $image || exit 1
    docker save -o $IMAGES_OUTPUT/$tar_name $image || exit 1
  else
    echo "==> Skip $image"
  fi
}



function get_main_ip() {
    local main_ip
    if command -v hostname &> /dev/null && hostname -I &> /dev/null; then
        main_ip=$(hostname -I | awk '{print $1}')
    fi

    if [[ -z "$main_ip" ]]; then
        local default_iface
        default_iface=$(ip route | awk '/default/ {print $5}')
        main_ip=$(ip addr show dev "$default_iface" | awk '/inet / {print $2}' | cut -d'/' -f1)
    fi

    if [[ -z "$main_ip" ]]; then
        main_ip=$(ifconfig | awk '/inet / && !/127.0.0.1/ {print $2}' | head -n1)
    fi

    echo "$main_ip"
}

function get_system_version() {
  VERSION_MAJOR=$VERSION_ID
  case "${VERSION_MAJOR}" in
      7*)
          VERSION_MAJOR="dnf"
          ;;
      8*)
          VERSION_MAJOR="dnf"
          ;;
      9*)
          VERSION_MAJOR="dnf"
          ;;
      22*)
          VERSION_MAJOR="apt"
          ;;
      24*)
          VERSION_MAJOR="apt"
          ;;
      kylin*)
          VERSION_MAJOR="dnf"
          ;;
      uos*)
          VERSION_MAJOR="apt"
          ;;
      *)
          echo "Unsupported version: $VERSION_MAJOR"
          ;;
  esac
}