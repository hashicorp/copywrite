# License Check

This module provides helper functions to validating and remediating any problems with a `LICENSE` file.

## Entry

The `Entry(dirPath string)` function takes in a directory path and will do the following:

- Check if any files appear to be licenses
- If no files are found, a stubbed out `addLicenseFile` function is called
- If a file is found but it does not adhere to the `LICENSE` desired nomenclature, it will be renamed
- If a file is found that matches the desired naming scheme, it is left alone
- If multiple files are found, an error will be returned

## License File Criteria

Potential LICENSE files are found by searching all files in a directory to find matching files with the name `LICENSE`
with or without `.txt` or `.md` extensions in a case-insensitive manner. As an example, the following all qualify:

- `LICENSE`
- `LICENSE.txt`
- `LICENSE.md`
- `license.TXT`
- `LiCeNsE` (for those who woke up and chose chaos)

## Testing

Due to the nature of mutating the filesystem, some functions in this module are not suited to being tested with a more
common `testdata` paradigm. Instead, `testing/TempDir()` is used to generate an ephemeral testing directory for each
sub-test.
