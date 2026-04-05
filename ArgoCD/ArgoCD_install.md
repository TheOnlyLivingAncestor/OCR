# ArgoCD installálása

Bármely kubernetes környezet megfelel, a projekt során Minikube cluster használatos. Telepítés a következőkkel:
```
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'

kubectl -n argocd patch secret argocd-secret \
    -p '{"stringData": {"admin.password": "$2a$10$mivhwttXM0U5eBrZGtAG8.VSRL1l9cZNAmaSaqotIzXRBRwID1NT.",
        "admin.passwordMtime": "'$(date +%FT%T)'"
    }}'
```

Az ArgoCD konzol URL címét a következő paranccsal lehet elérni, majd admin/admin párossal belépni:
```
minikube service -n argocd argocd-server
```