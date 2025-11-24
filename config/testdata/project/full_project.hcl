schema_version = 12

project {
  copyright_year   = 9001
  copyright_year1  = 6001
  copyright_year2  = 7001
  copyright_holder = "Dummy Corporation"
  license          = "NOT_A_VALID_SPDX"

  header_ignore = [
    "asdf.go",
    "*.css",
    "**/vendor/**.go",
  ]

  upstream = "hashicorp/super-secret-private-repo"
}
