# DB startup troubleshooting

## What was fixed

`database is starting` meant the Go API process had not finished its first MySQL connection and migration. Because `guardDB` protects all non-health API routes, login and registration were also blocked while the DB handle was nil.

The app now returns DB readiness details from `/api/healthz` and from guarded API routes, including the latest connection error and check time. Cloud Run infrastructure is also configured to keep one minimum instance warm during demos.

## Why auth was not split into another DB

The `users` table is referenced by item sellers, likes, purchases, conversations, and messages. Moving auth to a separate database would remove local foreign-key guarantees and make item ownership harder to keep consistent. For the current MVP, the safer fix is to keep a single Cloud SQL database and reduce cold-start connection failures.

## gcloud repair command

Run this after `gcloud auth login` if Cloud Run starts returning `database is starting` again:

```bash
./scripts/stabilize-cloud-run-db.sh
```

Equivalent explicit command:

```bash
PROJECT_ID=term9-toshiie-shiomi
REGION=asia-northeast1
SERVICE=next-market
INSTANCE=next-market-mysql
CONNECTION_NAME="$(gcloud sql instances describe "$INSTANCE" --project "$PROJECT_ID" --format='value(connectionName)')"

gcloud run services update "$SERVICE" \
  --project "$PROJECT_ID" \
  --region "$REGION" \
  --min-instances 1 \
  --add-cloudsql-instances "$CONNECTION_NAME" \
  --set-env-vars "INSTANCE_UNIX_SOCKET=/cloudsql/$CONNECTION_NAME"
```

## Check current state

```bash
gcloud run services describe next-market \
  --project term9-toshiie-shiomi \
  --region asia-northeast1

gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="next-market" AND textPayload:"database init"' \
  --project term9-toshiie-shiomi \
  --limit=20 \
  --freshness=24h
```
