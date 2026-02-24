 infra/terraform/staging-k8s && terraform apply \
    -var="registry_username=robot\$zenith-stage+ci-push" \
    -var="registry_password=flzHu8gJwQJL3eKgpNh8girixpnEBktR" \
    -var="jwt_secret=6AHcdLGHXqpJ6SOiWktf+Wzs6ZreaESdXpkJ7W+V4eKnaHbOIJ1cU+oSE3rv22c+" \
    -var="admin_email=admin@freezenith.com" \
    -var="admin_password=8i3wIotgaZEgxVnXMEpA"