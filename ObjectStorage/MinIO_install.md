# Object Storage (MinIO) installálása

Bármely kubernetes környezet megfelel, a projekt során Minikube cluster használatos. Telepítés a következőkkel:
```
kubectl create ns minio
helm repo add minio https://charts.min.io/
helm install minio minio/minio -n minio -f minio_values.yaml
```

A MinIO konzolt a következő paranccsal lehet Minikube-n elérni (megnyitja a böngészőt):
```
minikube service -n minio minio-console  
```

Belépést követően (a felhasználónév és a jelszó a values.yaml fájlban vannak) a "Create bucket" gombbal lehet bucketet kreálni.
