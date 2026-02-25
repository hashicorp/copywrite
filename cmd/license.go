// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/copywrite/github"
	"github.com/hashicorp/copywrite/licensecheck"
	"github.com/spf13/cobra"
)

// Flag variables
var (
	dirPath string
)

// licenseCmd represents the license command
var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Validates that a LICENSE file is present and remediates any issues if found",
	Long: `Validates that a LICENSE file is present and remediates any issues if found:
- Check if any files appear to be licenses
- If no files are found, a license will be added
- If a file is found but it does not adhere to the "LICENSE" desired nomenclature, it will be renamed
- If a file is found that matches the desired naming scheme, it is left alone
- If multiple files are found, an error will be returned`,
	GroupID: "common", // Let's put this command in the common section of the help
	PreRun: func(cmd *cobra.Command, args []string) {
		// Map command flags to config keys
		mapping := map[string]string{
			`spdx`:             `project.license`,
			`year`:             `project.copyright_year`,
			`copyright-holder`: `project.copyright_holder`,
		}

		// update the running config with any command-line flags
		clobberWithDefaults := false
		err := conf.LoadCommandFlags(cmd.Flags(), mapping, clobberWithDefaults)
		cobra.CheckErr(err)

		// Input Validation
		if conf.Project.CopyrightYear == 0 {
			errYearNotFound := errors.New("unable to automatically determine copyright year: Please specify it manually in the config or via the --year flag")

			cliLogger.Info("Copyright year was not supplied via config or via the --year flag. Attempting to infer from the year the GitHub repo was created.")
			repo, err := github.DiscoverRepo()
			if err != nil {
				cobra.CheckErr(fmt.Errorf("%v: %w", errYearNotFound, err))
			}

			client := github.NewGHClient().Raw()
			year, err := github.GetRepoCreationYear(client, repo)
			if err != nil {
				cobra.CheckErr(fmt.Errorf("%v: %w", errYearNotFound, err))
			}
			conf.Project.CopyrightYear = year
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		cmd.Printf("Licensing under the following terms: %s\n", conf.Project.License)

		// Determine appropriate copyright years for LICENSE file
		licenseYears := determineLicenseCopyrightYears(dirPath)

		cmd.Printf("Using copyright years: %v\n", licenseYears)
		cmd.Printf("Using copyright holder: %v\n\n", conf.Project.CopyrightHolder)

		copyright := "Copyright " + conf.Project.CopyrightHolder + " " + licenseYears

		licenseFiles, err := licensecheck.FindLicenseFiles(dirPath)
		if err != nil {
			cliLogger.Error("Error when discovering license files", err)
		}
		cobra.CheckErr(err)

		var file string

		if len(licenseFiles) > 1 {
			err = fmt.Errorf("more than one license file exists: Please review the following files and manually ensure only one is present: %s", licenseFiles)
			cliLogger.Error(err.Error())
			cobra.CheckErr(err)
			return
		}

		if len(licenseFiles) == 0 {
			if plan {
				cobra.CheckErr("missing license file. Run without the --plan flag to fix this")
			}

			cmd.Println("No license file found, creating one.")
			path, err := licensecheck.AddLicenseFile(dirPath, conf.Project.License)
			if err != nil {
				cliLogger.Error("Error adding new license file", err)
			}
			cobra.CheckErr(err)
			file = path
		}

		if len(licenseFiles) == 1 {
			file = licenseFiles[0]
		}

		// Only a single license file is present beyond this point

		// Let's make sure the license file adheres to our naming standard
		if plan {
			dir, _ := filepath.Split(file)
			desiredPath := filepath.Join(dir, "LICENSE")
			if file != desiredPath {
				err := fmt.Errorf("license file is misnamed. Run without the --plan flag to fix this")
				cliLogger.Error(err.Error())
				cobra.CheckErr(err)
			} else {
				cmd.Println("License file is present and named properly!")
			}
		} else {
			file, err = licensecheck.EnsureCorrectName(file)
			if err != nil {
				cliLogger.Error("Problem correcting LICENSE filename", err)
			}
			cobra.CheckErr(err)
		}

		// TODO: make sure the LICENSE file contains the appropriate license text

		// Let's make sure it has a valid copyright header, too
		cmd.Println("Validating presence of license header")

		hasCopyright, err := licensecheck.HasCopyright(file)
		if err != nil {
			cliLogger.Error("Problem verifying a copyright statement", err)
		}
		cobra.CheckErr(err)

		hasValidCopyright, err := licensecheck.HasMatchingCopyright(file, copyright, true)
		if err != nil {
			cliLogger.Error("Problem matching copyright", err)
		}
		cobra.CheckErr(err)

		if hasCopyright {
			if hasValidCopyright {
				cmd.Println("Copyright statement is valid!")
			} else {
				err = fmt.Errorf("license file has a copyright statement, but it is malformed; Expected to find: \"%s\" Please resolve this manually", copyright)
				cliLogger.Error(err.Error())
				cobra.CheckErr(err)
			}
		} else {
			if plan {
				cobra.CheckErr("a LICENSE file exists, but the copyright statement is missing. Run without the --plan flag to fix this")
			}

			cmd.Println("Copyright statement is missing... attempting to add it")
			err = licensecheck.AddHeader(file, copyright)
			if err != nil {
				cliLogger.Error("Error adding header", err)
			}
			cobra.CheckErr(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(licenseCmd)

	// These flags are only locally relevant
	licenseCmd.Flags().StringVarP(&dirPath, "dirPath", "d", ".", "Path to the directory in which you wish to validate a LICENSE file in")
	licenseCmd.Flags().BoolVar(&plan, "plan", false, "Performs a dry-run and gives a non-zero return if improperly licensed")

	// These flags will get mapped to keys in the the global Config
	// TODO: eventually, the copyrightYear should be dynamically inferred from the repo
	licenseCmd.Flags().IntP("year", "y", 0, "Year that the copyright statement should include")
	licenseCmd.Flags().StringP("spdx", "s", "", "SPDX License Identifier indicating what the LICENSE file should represent")
	licenseCmd.Flags().StringP("copyright-holder", "c", "", "Copyright holder (default \"IBM Corp.\")")
}

// determineLicenseCopyrightYears determines the appropriate copyright year range for LICENSE file
// Uses git history to get the start year (first commit) and end year (last commit)
func determineLicenseCopyrightYears(dirPath string) string {
	currentYear := time.Now().Year()
	startYear := conf.Project.CopyrightYear

	// If no start year configured, try to auto-detect from git
	if startYear == 0 {
		if detectedYear, err := licensecheck.GetRepoFirstCommitYear(dirPath); err == nil && detectedYear > 0 {
			startYear = detectedYear
		} else {
			// Fallback to current year
			return strconv.Itoa(currentYear)
		}
	}

	// Determine end year from repository's last commit year
	endYear := currentYear // Default fallback
	if lastRepoCommitYear, err := licensecheck.GetRepoLastCommitYear(dirPath); err == nil && lastRepoCommitYear > 0 && lastRepoCommitYear <= currentYear {
		endYear = lastRepoCommitYear
	}

	// If start year equals end year, return single year
	if startYear == endYear {
		return strconv.Itoa(endYear)
	}

	// Return year range: "startYear, endYear"
	return fmt.Sprintf("%d, %d", startYear, endYear)
}
