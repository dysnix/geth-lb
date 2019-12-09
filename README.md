

## Redis
helm install -n redis --name redis --set master.disableCommands='' --set usePassword=false stable/redis

