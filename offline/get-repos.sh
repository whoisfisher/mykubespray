#! /bin/bash
. .env
. /etc/os-release
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


function download_repo() {
  get_system_version
  if [ ! -e $REPO_OUTPUT/$VERSION_ID ]; then
    mkdir -p $REPO_OUTPUT/$VERSION_ID
  fi
  if [ "$VERSION_MAJOR" == "dnf" ]; then
    packages=$(cat dnf.list | grep -v "^#" | sort | uniq)
    repotrack -p $REPO_OUTPUT/$VERSION_ID $packages || {
          echo "Download error"
          exit 1
    }
  elif [ "$VERSION_MAJOR" == "apt" ]; then
    packages=$(cat apt.list | grep -v "^#" | sort | uniq)
    echo "===> Install Repository"
    sudo apt update
    sudo apt install -y apt-transport-https ca-certificates curl gnupg lsb-release apt-utils
    echo "===> Update apt cache"
    sudo apt update
    echo "===> Resolving dependencies"
    DEPS=$(apt-cache depends --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends $packages | grep "^\w" | sort | uniq)
    echo "===> Downloading packages: " $packages $DEPS
    cd $REPO_OUTPUT/$VERSION_ID && apt download $packages $DEPS
  fi
}

function remove_repo() {
  get_system_version
  rm -rf $REPO_OUTPUT/$VERSION_ID
}