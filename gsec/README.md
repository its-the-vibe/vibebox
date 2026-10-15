# gsec

Manage Google Cloud Secret Manager secrets with the `gcloud` CLI. Authenticate
with `gcloud` and select a project before use.

```sh
gsec view <secret-name> [--version <version>]
gsec put <secret-name> <key> <value>
gsec versions <secret-name>
gsec versions delete-oldest <secret-name>
```

`put` treats the secret payload as a JSON object and stores the supplied value
as a string in a new version. `versions delete-oldest` destroys the oldest
enabled version.
