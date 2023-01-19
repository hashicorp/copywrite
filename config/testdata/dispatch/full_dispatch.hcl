schema_version = 78

dispatch {
  batch_id = "aZ0-9"

  branch = "main"

  ignored_repos = [
    "org/repo1",
    "org/repo2",
  ]

  sleep = 42

  max_attempts = 3

  workers = 12

  workflow_file_name = "repair-repo-headers.yml"
}
