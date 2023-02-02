schema_version = 12

project {
  copyright_year   = 9001
  copyright_holder = "Dummy Corporation"
  license          = "NOT_A_VALID_SPDX"

  header_ignore = [
    "asdf.go",
    "*.css",
    "**/vendors/**.go",
  ]

  upstream = "hashicorp/super-secret-private-repo"
}
