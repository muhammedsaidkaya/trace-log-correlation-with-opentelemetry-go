
SERVICE_NAME=$1
EXPOSE_PORT=$2
INTERNAL_PORT=$3

kubectl port-forward pod/$(kubectl get po | grep -i "service1" | awk '{ print $1 }') ${EXPOSE_PORT}:${INTERNAL_PORT}