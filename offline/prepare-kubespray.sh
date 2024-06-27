#! /bin/bash
. .env
. common.sh
. check-env.sh


function create_kubespray() {
  check_docker
  docker_ps=$(docker ps | grep "$KUBESPRAY_NAME")
  if [ -n "$docker_ps" ]; then
      echo "Error: Docker $KUBESPRAY_NAME already exists"
      exit 0
  fi
  kubespray_image=$KUBESPRAY_NAME:v$KUBESPRAY_VERSION
  tar_name=$(echo ${kubespray_image##*/} | sed s/:/-/g).tar
  echo "===>loading kubespray_image"
  docker load -i $IMAGES_OUTPUT/$ARCH/$tar_name
  echo "===> starting kubespray"
  docker run --rm -it -v $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/:/kubespray/ -v /root/.ssh/id_rsa:/root/.ssh/id_rsa --name $KUBESPRAY_NAME $kubespray_image \
      ansible-playbook -i /kubespray/inventory/$CLUSTER_NAME/hosts.yaml --private-key /root/.ssh/id_rsa --become --become-user=root cluster.yml
  if [ $? -ne 0 ]; then
    echo "Error: Failed to run Docker kubespray container."
    exit 1
  else
    echo "Create kubespray done."
  fi
}


function configure_kubespray_containerd() {
  current_ip=$(get_main_ip)
  tee $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME/group_vars/all/containerd.yml >> /dev/null <<EOF
containerd_registries_mirrors:
 - prefix: $current_ip:$REGISTRY_PORT
   mirrors:
    - host: http://$current_ip:$REGISTRY_PORT
      capabilities: ["pull", "resolve", "push"]
      skip_verify: true
 - prefix: registry.dev.rdev.tech:18093
   mirrors:
    - host: http://registry.dev.rdev.tech:18093
      capabilities: ["pull", "resolve", "push"]
      skip_verify: true
EOF
}


function configure_nexus_hosts() {
  current_ip=$(get_main_ip)
  tee $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/update_hosts.yaml >> /dev/null <<EOF
---
- name: Update /etc/hosts on all nodes
  hosts: all
  become: yes
  tasks:
    - name: Append extra hosts entry to /etc/hosts
      lineinfile:
        path: /etc/hosts
        line: "$current_ip registry.dev.rdev.tech"
      delegate_to: "{{ inventory_hostname }}"
EOF
  check_docker
  docker_ps=$(docker ps | grep "$KUBESPRAY_NAME")
  if [ -n "$docker_ps" ]; then
      echo "Error: Docker $KUBESPRAY_NAME already exists"
      exit 0
  fi
  kubespray_image=$KUBESPRAY_NAME:v$KUBESPRAY_VERSION
  tar_name=$(echo ${kubespray_image##*/} | sed s/:/-/g).tar
  echo "===>loading kubespray_image"
  docker load -i $IMAGES_OUTPUT/$ARCH/$tar_name
  echo "===> starting kubespray"
  docker run --rm -it -v $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/:/kubespray/ -v /root/.ssh/id_rsa:/root/.ssh/id_rsa --name $KUBESPRAY_NAME $kubespray_image \
      ansible-playbook -i /kubespray/inventory/$CLUSTER_NAME/hosts.yaml --private-key /root/.ssh/id_rsa --become --become-user=root update_hosts.yaml
  if [ $? -ne 0 ]; then
    echo "Error: Failed to run Docker kubespray container."
    exit 1
  else
    echo "Create kubespray done."
  fi

}


function init_kubernetes() {
  cp -rf $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/sample $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME
  cp hosts.yaml $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME/hosts.yaml
  cp offline2.yaml $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME/group_vars/all/offline.yaml
  echo "===>Please add node information in the following file"
  echo "$KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME/hosts.yaml"
  echo "===>Please add image repository, file repository, and system source in the following file"
  echo "$KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME/group_vars/all/offline.yaml"
  echo "===>Please add cluster related configurations in the following file"
  echo "$KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME/group_vars/k8s_cluster/k8s-cluster.yml"
}

function install_kubernetes() {
  check_docker
  docker_ps=$(docker ps | grep "$KUBESPRAY_NAME")
  if [ -n "$docker_ps" ]; then
      echo "Error: Docker $KUBESPRAY_NAME already exists"
      exit 0
  fi
  kubespray_image=$KUBESPRAY_NAME:v$KUBESPRAY_VERSION
  tar_name=$(echo ${kubespray_image##*/} | sed s/:/-/g).tar
  echo "===>loading kubespray_image"
  docker load -i $IMAGES_OUTPUT/$ARCH/$tar_name
  echo "===> starting kubespray"
  docker run --rm -it -v $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/:/kubespray/ -v /root/.ssh/id_rsa:/root/.ssh/id_rsa --name $KUBESPRAY_NAME $kubespray_image \
      ansible-playbook -i /kubespray/inventory/$CLUSTER_NAME/hosts.yaml --private-key /root/.ssh/id_rsa --become --become-user=root cluster.yml
  if [ $? -ne 0 ]; then
    echo "Error: Failed to run Docker kubespray container."
    exit 1
  else
    echo "Create kubespray done."
  fi
}


function delete_kubernetes() {
  check_docker
  docker_ps=$(docker ps | grep "$KUBESPRAY_NAME")
  if [ -n "$docker_ps" ]; then
      echo "Error: Docker $KUBESPRAY_NAME already exists"
      exit 0
  fi
  kubespray_image=$KUBESPRAY_NAME:v$KUBESPRAY_VERSION
  tar_name=$(echo ${kubespray_image##*/} | sed s/:/-/g).tar
  echo "===>loading kubespray_image"
  docker load -i $IMAGES_OUTPUT/$ARCH/$tar_name
  echo "===> starting kubespray"
  docker run --rm -it -v $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/:/kubespray/ -v /root/.ssh/id_rsa:/root/.ssh/id_rsa --name $KUBESPRAY_NAME $kubespray_image \
      ansible-playbook -i /kubespray/inventory/$CLUSTER_NAME/hosts.yaml --private-key /root/.ssh/id_rsa --become --become-user=root reset.yml
  if [ $? -ne 0 ]; then
    echo "Error: Failed to delete kubernetes: $CLUSTER_NAME"
    exit 1
  else
    echo "Delete kubernetes: $CLUSTER_NAME done."
  fi
  rm -rf $KUBESPRAY_CACHE/kubespray-$KUBESPRAY_VERSION/inventory/$CLUSTER_NAME
}