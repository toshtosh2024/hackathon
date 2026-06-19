# GCP Deploy

This Terraform deploys the app to Cloud Run with Cloud SQL for MySQL.

## Prerequisites

- GCP project: `term9-toshiie-shiomi`
- Billing enabled on the project
- Active gcloud account: `taishi14ki@gmail.com`

## Deploy

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
terraform init
GOOGLE_OAUTH_ACCESS_TOKEN="$(gcloud auth print-access-token)" terraform apply \
  -target=google_project_service.services \
  -target=google_artifact_registry_repository.app
```

Create the OpenAI API key secret before the full Terraform apply:

```bash
printf '%s' 'YOUR_OPENAI_API_KEY' | gcloud secrets create next-market-openai-api-key \
  --project term9-toshiie-shiomi \
  --replication-policy=automatic \
  --data-file=-
```

Build and push the container image:

```bash
cd ../..
gcloud builds submit --project term9-toshiie-shiomi --config cloudbuild.yaml .
```

`cloudbuild.yaml` also deploys the pushed image to Cloud Run. Automatic deploys on GitHub `main` pushes use the Cloud Build GitHub connection:

Every deployment explicitly re-applies the Terraform-managed runtime service account, uploads bucket, and Cloud SQL connection. Image uploads use Cloud Run Application Default Credentials; no GCS private key is stored in Secret Manager.

- Connection: `tosh`
- Connection region: `us-central1`
- Provider account: `Tk-visionary`
- Repository: `utaishi-utoshiie/hackathon`

The Cloud Build GitHub App installation must have access to `utaishi-utoshiie/hackathon`. If Terraform reports that the repository is not accessible to the GitHub App, update the GitHub App repository access for installation `140825418`, then enable the trigger with:

```hcl
enable_github_trigger = true
```

Deploy the remaining resources:

```bash
cd infra/terraform
GOOGLE_OAUTH_ACCESS_TOKEN="$(gcloud auth print-access-token)" terraform apply
terraform output cloud_run_url
```

If the service was previously deployed without the storage settings, repair the live revision once with:

```bash
cd ../..
./scripts/stabilize-cloud-run-storage.sh
```

The script verifies that the bucket exists, grants the runtime service account object creation permission, preserves public product-image reads, and updates `GCS_BUCKET` on Cloud Run.

To rotate the OpenAI key after deployment:

```bash
printf '%s' 'YOUR_OPENAI_API_KEY' | gcloud secrets versions add next-market-openai-api-key \
  --project term9-toshiie-shiomi \
  --data-file=-
```
