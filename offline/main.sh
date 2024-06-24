#! /bin/bash

. get-docker.sh
. get-kubespray.sh
. get-images.sh
. get-files.sh
. get-repos.sh
. get-others-images.sh
. get-fileserver.sh
. prepare-docker.sh
. prepare-fileserver.sh
. prepare-images.sh
. prepare-repo.sh
. prepare-kubespray.sh

usage() {
    echo "Usage:"
    echo "  $0 <command> [<target>] [-r] [-h] [-cn <cluster_name>] [-cr <container_runtime>] [-np <network_plugin>]"
    echo
    echo "Commands:"
    echo "  get <target>          Retrieve specific targets:"
    echo "                          docker"
    echo "                          kubespray"
    echo "                          images"
    echo "                          files"
    echo "                          os-repositories"
    echo "                          other-images"
    echo "                          all"
    echo "  delete <target>       Delete specific targets:"
    echo "                          docker"
    echo "                          kubespray"
    echo "                          images"
    echo "                          files"
    echo "                          os-repositories"
    echo "                          all"
    echo "  make <target>         Make something:"
    echo "                          kubespray"
    echo "                          files-server"
    echo "  install <target>      Install something:"
    echo "                          docker"
    echo "                          cluster"
    echo "  remove <target>       Remove something:"
    echo "                          docker"
    echo "                          cluster"
    echo "  create <target>       Create something:"
    echo "                          docker-registry"
    echo "                          files-server"
    echo "  init <target>         Init something:"
    echo "                          cluster"
    echo "  Config <target>       Config something:"
    echo "                          docker"
    echo
    echo "Options:"
    echo "  -r                     Enable recursive mode (for applicable commands)."
    echo "  -h                     Display this help message."
    echo "  -cn <cluster_name>     Specify the cluster name (required for 'create cluster' and 'remove cluster')."
    echo "  -cr <container_runtime> Specify the container runtime."
    echo "  -np <network_plugin>   Specify the network plugin."
    echo
    echo "Examples:"
    echo "  $0 get docker -r -cn mycluster -cr docker"
    echo "  $0 delete kubespray -r"
    echo "  $0 make kubespray"
    echo "  $0 install docker"
    echo "  $0 remove cluster docker -cn mycluster"
    echo "  $0 create docker-registry -cn production -cr docker"
    echo
    echo "Notes:"
    echo "  - Commands 'create' and 'remove' with 'cluster' target require '-cn <cluster_name>'."
    echo "  - Commands 'install' and 'remove' with 'cluster' target can specify '-cr <container_runtime>'."
    echo "  - Commands 'create' can specify '-cr <container_runtime>' and '-np <network_plugin>'."
    exit 1
}


if [ $# -lt 1 ]; then
  echo "Error: You must specify a command."
  usage
fi


command="$1"
shift

case "$command" in
    get)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            docker|kubespray|images|files|os-repositories|other-images|all)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'docker', 'kubespray', 'images', 'files', 'os-repositories', 'other-images', 'all'."
                usage
                ;;
        esac
        ;;
    delete)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            docker|kubespray|images|files|os-repositories|all)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'docker', 'kubespray', 'images', 'files', 'os-repositories', 'all'."
                usage
                ;;
        esac
        ;;
    make)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            make_kubespray|files-server)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'make_kubespray', 'files-server'."
                usage
                ;;
        esac
        ;;
    install|remove)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            docker|cluster)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'docker', 'cluster'."
                usage
                ;;
        esac
        if [ "$command" = "create" ] || [ "$command" = "remove" ] || [ "$command" = "init" ] ; then
            if [ "$target" = "cluster" ]; then
                if [ -z "$2" ]; then
                    echo "Error: You must specify a cluster name with '-cn' for 'create cluster' or 'remove cluster' or 'init cluster'."
                    usage
                fi
                cluster_name="$2"
                shift 2
            fi
        fi
        ;;
    create)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            docker-registry|files-server|os-repositories)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'docker-registry', 'files-server', 'os-repositories'."
                usage
                ;;
        esac
        if [ "$command" = "create" ] || [ "$command" = "remove" ] || [ "$command" = "init" ] ; then
            if [ "$target" = "cluster" ]; then
                if [ -z "$2" ]; then
                    echo "Error: You must specify a cluster name with '-cn' for 'create cluster' or 'remove cluster' or 'init cluster'."
                    usage
                fi
                cluster_name="$2"
                shift 2
            fi
        fi
        ;;
    init)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            cluster)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'cluster'."
                usage
                ;;
        esac

        if [ "$command" = "create" ] || [ "$command" = "remove" ] || [ "$command" = "init" ] ; then
            if [ "$target" = "cluster" ]; then
                if [ -z "$2" ]; then
                    echo "Error: You must specify a cluster name with '-cn' for 'create cluster' or 'remove cluster' or 'init cluster'."
                    usage
                fi
                cluster_name="$2"
                shift 2
            fi
        fi
        ;;
    config)
        if [ $# -lt 1 ]; then
            echo "Error: You must specify a target for '$command'."
            usage
        fi

        case "$1" in
            docker)
                target="$1"
                shift
                ;;
            *)
                echo "Error: Invalid target '$1' for command '$command'. Must be one of 'docker'."
                usage
                ;;
        esac
        ;;
    *)
        echo "Error: Invalid command '$command'. Must be one of 'get', 'delete', 'make', 'install', 'remove', 'create', 'init'."
        usage
        ;;
esac

recursive=false
cluster_name=""
container_runtime=""
network_plugin=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -r)
            recursive=true
            shift
            ;;
        -h)
            usage
            ;;
        -cn)
            if [ -z "$2" ]; then
                echo "Error: Cluster name cannot be empty."
                usage
            fi
            cluster_name="$2"
            shift 2
            ;;
        -cr)
            if [ -z "$2" ]; then
                echo "Error: Container runtime cannot be empty."
                usage
            fi
            container_runtime="$2"
            shift 2
            ;;
        -np)
            if [ -z "$2" ]; then
                echo "Error: Network plugin cannot be empty."
                usage
            fi
            network_plugin="$2"
            shift 2
            ;;
        *)
            echo "Error: Unknown option '$1'."
            usage
            ;;
    esac
done

echo "Command: $command"
if [ -n "$target" ]; then
    echo "Target: $target"
fi
echo "Recursive mode: $recursive"
if [ -n "$cluster_name" ]; then
    echo "Cluster Name: $cluster_name"
    export CLUSTER_NAME=$cluster_name
fi
if [ -n "$container_runtime" ]; then
    echo "Container Runtime: $container_runtime"
fi
if [ -n "$network_plugin" ]; then
    echo "Network Plugin: $network_plugin"
fi

case "$command" in
    get)
        echo "Performing 'get' operation on '$target'."
        case "$target" in
        docker)
          get_docker
          ;;
        kubespray)
          download_kubespray
          ;;
        images)
          download_images
          ;;
        files)
          download_files
          ;;
        os-repositories)
          download_repo
          ;;
        other-images)
          get_other_images
          ;;
        all)
          download_kubespray
          download_files
          get_docker
          install_docker
          download_images
          download_repo
          get_other_images
          make_kubespray
          make_file_server
          ;;
        esac
        ;;
    delete)
        echo "Performing 'delete' operation on '$target'."
        case "$target" in
        docker)
          remove_docker
          ;;
        kubespray)
          remove_kubespray
          ;;
        images)
          remove_images
          ;;
        files)
          remove_files
          ;;
        os-repositories)
          remove_repo
          ;;
        all)
          remove_kubespray
          remove_images
          remove_files
          remove_repo
          remove_docker
          ;;
        esac
        ;;
    make)
        echo "Performing 'make' operation on '$target'."
        case "$target" in
        kubespray)
          make_kubespray
          ;;
        files-server)
          make_file_server
          ;;
        esac
        ;;
    install)
        echo "Performing 'install' operation on '$target'."
        case "$target" in
        docker)
          install_docker
          ;;
        cluster)
          create_registry
          configure_docker
          pushed_images
          create_file_server
          create_repo
          configure_repo
          configure_kubespray_containerd
          install_kubernetes
          ;;
        esac
        ;;
    remove)
        echo "Performing 'remove' operation on '$target'."
        case "$target" in
        docker)
          remove_docker
          ;;
        cluster)
          remove_docker
          remove_file_server
          remove_repo
          delete_kubernetes
          ;;
        esac
        ;;
    create)
        echo "Performing 'create' operation on '$target'."
        case "$target" in
        docker-registry)
          create_registry
          configure_docker
          pushed_images
          ;;
        files-server)
          create_file_server
          ;;
        os-repositories)
          create_repo
          configure_repo
          ;;
        esac
        ;;
    init)
        echo "Performing 'init' operation on '$target'."
        case "$target" in
        cluster)
          init_kubernetes
          ;;
        esac
        ;;
    config)
        echo "Performing 'init' operation on '$target'."
        case "$target" in
        docker)
          configure_proxy
          ;;
        esac
        ;;
esac