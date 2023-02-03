schema_version = 78

dispatch {
  batch_id = "aZ0-9"

  branch = "main"

  github_org_to_audit = "hashicorp-forge"

  ignored_repos = [
    "org/repo1",
    "org/repo2",
  ]

  sleep = 42

  max_attempts = 3

  workers = 12

  workflow_file_name = "repair-repo-headers.yml"
}
