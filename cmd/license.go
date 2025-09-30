// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

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
			`year1`:            `project.copyright_year1`,
			`year2`:            `project.copyright_year2`,
			`copyright-holder`: `project.copyright_holder`,
		}

		// update the running config with any command-line flags
		clobberWithDefaults := false
		err := conf.LoadCommandFlags(cmd.Flags(), mapping, clobberWithDefaults)
		cobra.CheckErr(err)

		// Input Validation
		// Check if we have year information from new year1/year2 flags or legacy year flag
		hasYearInfo := conf.Project.CopyrightYear > 0 || conf.Project.CopyrightYear1 > 0 || conf.Project.CopyrightYear2 > 0

		if !hasYearInfo {
			errYearNotFound := errors.New("unable to automatically determine copyright year: Please specify it manually in the config or via the --year, --year1, or --year2 flag")

			cliLogger.Info("Copyright year was not supplied via config or via the --year/--year1/--year2 flags. Attempting to infer from the year the GitHub repo was created.")
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

		// Construct the year range similar to headers command
		yearRange := ""
		if conf.Project.CopyrightYear1 > 0 && conf.Project.CopyrightYear2 > 0 {
			if conf.Project.CopyrightYear1 == conf.Project.CopyrightYear2 {
				yearRange = fmt.Sprintf("%d", conf.Project.CopyrightYear1)
			} else {
				yearRange = fmt.Sprintf("%d, %d", conf.Project.CopyrightYear1, conf.Project.CopyrightYear2)
			}
		} else if conf.Project.CopyrightYear1 > 0 {
			yearRange = fmt.Sprintf("%d", conf.Project.CopyrightYear1)
		} else if conf.Project.CopyrightYear2 > 0 {
			yearRange = fmt.Sprintf("%d", conf.Project.CopyrightYear2)
		} else if conf.Project.CopyrightYear > 0 {
			// Fallback to legacy single year for backward compatibility
			yearRange = fmt.Sprintf("%d", conf.Project.CopyrightYear)
		}

		if yearRange != "" {
			cmd.Printf("Using year of initial copyright: %v\n", yearRange)
		}
		cmd.Printf("Using copyright holder: %v\n\n", conf.Project.CopyrightHolder)

		// Use the same format as headers command: "Copyright [HOLDER] [YEAR_RANGE]"
		var copyright string
		if yearRange != "" {
			copyright = "Copyright " + conf.Project.CopyrightHolder + " " + yearRange
		} else {
			copyright = "Copyright " + conf.Project.CopyrightHolder
		}

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
	licenseCmd.Flags().IntP("year1", "", 0, "Start year for copyright range (e.g., 2020)")
	licenseCmd.Flags().IntP("year2", "", 0, "End year for copyright range (e.g., 2025)")
	licenseCmd.Flags().StringP("spdx", "s", "", "SPDX License Identifier indicating what the LICENSE file should represent")
	licenseCmd.Flags().StringP("copyright-holder", "c", "", "Copyright holder (default \"IBM Corp.\")")
}
