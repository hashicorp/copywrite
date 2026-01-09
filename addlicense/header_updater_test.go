// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package addlicense

import (
	"os"
	"testing"
)

func TestParseCopyrightHeader(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantOrg        string
		wantYear       string
		wantHolder     string
		wantAdditional string
		wantNil        bool
	}{
		{
			name:       "Hashicorp with (c) and year",
			input:      "Copyright (c) 2020 Hashicorp Inc.",
			wantOrg:    "Hashicorp",
			wantYear:   "2020",
			wantHolder: "Hashicorp",
		},
		{
			name:       "Hashicorp without (c)",
			input:      "Copyright 2020 Hashicorp Inc.",
			wantOrg:    "Hashicorp",
			wantYear:   "2020",
			wantHolder: "Hashicorp",
		},
		{
			name:       "Hashicorp without year",
			input:      "Copyright Hashicorp Inc.",
			wantOrg:    "Hashicorp",
			wantYear:   "",
			wantHolder: "Hashicorp",
		},
		{
			name:       "Hashicorp with (c) no year",
			input:      "Copyright (c) Hashicorp Inc.",
			wantOrg:    "Hashicorp",
			wantYear:   "",
			wantHolder: "Hashicorp",
		},
		{
			name:       "Hashicorp with comma",
			input:      "Copyright (c) Hashicorp, Inc.",
			wantOrg:    "Hashicorp",
			wantYear:   "",
			wantHolder: "Hashicorp",
		},
		{
			name:       "Hashicorp with comma and year",
			input:      "Copyright (c) 2020 Hashicorp, Inc.",
			wantOrg:    "Hashicorp",
			wantYear:   "2020",
			wantHolder: "Hashicorp",
		},
		{
			name:       "IBM Corp with year range",
			input:      "Copyright IBM Corp. 2020,2025",
			wantOrg:    "IBM",
			wantYear:   "2020,2025",
			wantHolder: "IBM Corp.",
		},
		{
			name:       "IBM Corp with (c) and year range",
			input:      "Copyright (c) IBM Corp. 2020, 2025",
			wantOrg:    "IBM",
			wantYear:   "2020,2025",
			wantHolder: "IBM Corp.",
		},
		{
			name:       "IBM Corp single year",
			input:      "Copyright IBM Corp. 2025",
			wantOrg:    "IBM",
			wantYear:   "2025",
			wantHolder: "IBM Corp.",
		},
		{
			name:           "Hashicorp with additional text",
			input:          "Copyright (c) Hashicorp Inc. All rights reserved.",
			wantOrg:        "Hashicorp",
			wantYear:       "",
			wantHolder:     "Hashicorp",
			wantAdditional: "All rights reserved.",
		},
		{
			name:           "IBM with additional text",
			input:          "Copyright IBM Corp. 2020,2025 All rights reserved.",
			wantOrg:        "IBM",
			wantYear:       "2020,2025",
			wantHolder:     "IBM Corp.",
			wantAdditional: "All rights reserved.",
		},
		{
			name:    "Other organization",
			input:   "Copyright (c) 2020 Google LLC",
			wantOrg: "Other",
			wantNil: false,
		},
		{
			name:    "Another organization",
			input:   "Copyright Microsoft Corporation",
			wantOrg: "Other",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseCopyrightHeader(tt.input)

			if tt.wantNil {
				if info != nil {
					t.Errorf("ParseCopyrightHeader() = %v, want nil", info)
				}
				return
			}

			if info == nil {
				t.Fatalf("ParseCopyrightHeader() = nil, want non-nil")
			}

			if info.Organization != tt.wantOrg {
				t.Errorf("Organization = %q, want %q", info.Organization, tt.wantOrg)
			}

			if tt.wantOrg != "Other" {
				if info.Year != tt.wantYear {
					t.Errorf("Year = %q, want %q", info.Year, tt.wantYear)
				}

				if info.Holder != tt.wantHolder {
					t.Errorf("Holder = %q, want %q", info.Holder, tt.wantHolder)
				}

				if info.AdditionalText != tt.wantAdditional {
					t.Errorf("AdditionalText = %q, want %q", info.AdditionalText, tt.wantAdditional)
				}
			}
		})
	}
}

func TestParseYearRange(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantStartYear int
		wantEndYear   int
	}{
		{
			name:          "Single year",
			input:         "2020",
			wantStartYear: 2020,
			wantEndYear:   2020,
		},
		{
			name:          "Year range with comma",
			input:         "2020,2025",
			wantStartYear: 2020,
			wantEndYear:   2025,
		},
		{
			name:          "Year range with comma and space",
			input:         "2020, 2025",
			wantStartYear: 2020,
			wantEndYear:   2025,
		},
		{
			name:          "Year range with dash",
			input:         "2020-2025",
			wantStartYear: 2020,
			wantEndYear:   2025,
		},
		{
			name:          "Empty string",
			input:         "",
			wantStartYear: 0,
			wantEndYear:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startYear, endYear := ParseYearRange(tt.input)
			if startYear != tt.wantStartYear {
				t.Errorf("startYear = %d, want %d", startYear, tt.wantStartYear)
			}
			if endYear != tt.wantEndYear {
				t.Errorf("endYear = %d, want %d", endYear, tt.wantEndYear)
			}
		})
	}
}

func TestFormatYearRange(t *testing.T) {
	tests := []struct {
		name      string
		startYear int
		endYear   int
		want      string
	}{
		{
			name:      "Same year",
			startYear: 2025,
			endYear:   2025,
			want:      "2025",
		},
		{
			name:      "Year range",
			startYear: 2020,
			endYear:   2025,
			want:      "2020, 2025",
		},
		{
			name:      "Start year only",
			startYear: 2020,
			endYear:   0,
			want:      "2020",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatYearRange(tt.startYear, tt.endYear)
			if got != tt.want {
				t.Errorf("FormatYearRange() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldUpdateHeader(t *testing.T) {
	currentYear := 2026

	tests := []struct {
		name            string
		headerInfo      *HeaderInfo
		fileModYear     int
		configStartYear int
		want            bool
	}{
		{
			name: "Hashicorp header - should update",
			headerInfo: &HeaderInfo{
				Organization: "Hashicorp",
				Year:         "2020",
			},
			fileModYear:     2025,
			configStartYear: 2014,
			want:            true,
		},
		{
			name: "IBM header - file modified in current year, outdated year",
			headerInfo: &HeaderInfo{
				Organization: "IBM",
				Year:         "2020,2025",
			},
			fileModYear:     currentYear,
			configStartYear: 2020,
			want:            true,
		},
		{
			name: "IBM header - file modified in current year, up-to-date",
			headerInfo: &HeaderInfo{
				Organization: "IBM",
				Year:         "2020,2026",
			},
			fileModYear:     currentYear,
			configStartYear: 2020,
			want:            false,
		},
		{
			name: "IBM header - file not modified in current year but wrong start year",
			headerInfo: &HeaderInfo{
				Organization: "IBM",
				Year:         "2013,2025",
			},
			fileModYear:     2025,
			configStartYear: 2014,
			want:            true,
		},
		{
			name: "IBM header - file not modified in current year, correct start year",
			headerInfo: &HeaderInfo{
				Organization: "IBM",
				Year:         "2014,2025",
			},
			fileModYear:     2025,
			configStartYear: 2014,
			want:            false,
		},
		{
			name: "Other organization - should not update",
			headerInfo: &HeaderInfo{
				Organization: "Other",
				Year:         "2020",
			},
			fileModYear:     currentYear,
			configStartYear: 2014,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUpdateHeader(tt.headerInfo, currentYear, tt.fileModYear, tt.configStartYear)
			if got != tt.want {
				t.Errorf("ShouldUpdateHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateUpdatedHeader(t *testing.T) {
	tests := []struct {
		name      string
		info      *HeaderInfo
		newHolder string
		newYear   string
		want      string
	}{
		{
			name: "Simple header",
			info: &HeaderInfo{
				AdditionalText: "",
			},
			newHolder: "IBM Corp.",
			newYear:   "2020, 2025",
			want:      "Copyright IBM Corp. 2020, 2025",
		},
		{
			name: "Header with additional text",
			info: &HeaderInfo{
				AdditionalText: "All rights reserved.",
			},
			newHolder: "IBM Corp.",
			newYear:   "2020, 2025",
			want:      "Copyright IBM Corp. 2020, 2025 All rights reserved.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUpdatedHeader(tt.info, tt.newHolder, tt.newYear, true)
			if got != tt.want {
				t.Errorf("GenerateUpdatedHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindAllCopyrightHeaders(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantOrgs  []string
	}{
		{
			name: "Single Hashicorp header",
			content: `// Copyright (c) 2020 Hashicorp Inc.
// Some other comment
package main`,
			wantCount: 1,
			wantOrgs:  []string{"Hashicorp"},
		},
		{
			name: "Multiple headers",
			content: `// Copyright (c) 2020 Hashicorp Inc.
// SPDX-License-Identifier: MPL-2.0

// Some code here
// Copyright IBM Corp. 2023, 2025

package main`,
			wantCount: 2,
			wantOrgs:  []string{"Hashicorp", "IBM"},
		},
		{
			name: "IBM header only",
			content: `// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			wantCount: 1,
			wantOrgs:  []string{"IBM"},
		},
		{
			name: "No headers",
			content: `package main

func main() {
}`,
			wantCount: 0,
			wantOrgs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := FindAllCopyrightHeaders([]byte(tt.content))

			if len(headers) != tt.wantCount {
				t.Errorf("FindAllCopyrightHeaders() found %d headers, want %d", len(headers), tt.wantCount)
			}

			for i, header := range headers {
				if i < len(tt.wantOrgs) && header.Organization != tt.wantOrgs[i] {
					t.Errorf("Header %d: Organization = %q, want %q", i, header.Organization, tt.wantOrgs[i])
				}
			}
		})
	}
}

func TestCheckIfHeaderNeedsUpdate(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		configYear     string
		configHolder   string
		expectedUpdate bool
		expectedError  bool
	}{
		{
			name: "Hashicorp header needs update",
			fileContent: `// Copyright (c) 2020 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:     "2014",
			configHolder:   "IBM Corp.",
			expectedUpdate: true,
			expectedError:  false,
		},
		{
			name: "IBM header up-to-date",
			fileContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:     "2014",
			configHolder:   "IBM Corp.",
			expectedUpdate: false,
			expectedError:  false,
		},
		{
			name: "IBM header needs year update (file modified in current year)",
			fileContent: `// Copyright IBM Corp. 2014, 2020
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:     "2014",
			configHolder:   "IBM Corp.",
			expectedUpdate: true,
			expectedError:  false,
		},
		{
			name: "IBM header needs config year update",
			fileContent: `// Copyright IBM Corp. 2013, 2020
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:     "2014",
			configHolder:   "IBM Corp.",
			expectedUpdate: true,
			expectedError:  false,
		},
		{
			name: "Other organization header - no update",
			fileContent: `// Copyright (c) 2020 Google LLC
// SPDX-License-Identifier: Apache-2.0

package main`,
			configYear:     "2014",
			configHolder:   "IBM Corp.",
			expectedUpdate: false,
			expectedError:  false,
		},
		{
			name: "No copyright header - no update needed (will be added)",
			fileContent: `// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:     "2014",
			configHolder:   "IBM Corp.",
			expectedUpdate: false,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := t.TempDir() + "/test.go"
			if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create LicenseData
			data := LicenseData{
				Year:   tt.configYear,
				Holder: tt.configHolder,
				SPDXID: "MPL-2.0",
			}

			needsUpdate, err := CheckIfHeaderNeedsUpdate(tmpFile, data)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if needsUpdate != tt.expectedUpdate {
				t.Errorf("CheckIfHeaderNeedsUpdate() = %v, want %v", needsUpdate, tt.expectedUpdate)
			}
		})
	}
}

func TestUpdateFileHeaders(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		configYear      string
		configHolder    string
		expectedUpdated bool
		expectedContent string
		expectedError   bool
	}{
		{
			name: "Update Hashicorp to IBM",
			fileContent: `// Copyright (c) 2020 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

func main() {}`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main

func main() {}`,
			expectedError: false,
		},
		{
			name: "Update IBM year range (file modified in current year)",
			fileContent: `// Copyright IBM Corp. 2014, 2020
// SPDX-License-Identifier: MPL-2.0

package main

func main() {}`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main

func main() {}`,
			expectedError: false,
		},
		{
			name: "Update IBM config year",
			fileContent: `// Copyright IBM Corp. 2013, 2020
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedError: false,
		},
		{
			name: "Preserve additional text",
			fileContent: `// Copyright IBM Corp. 2013, 2020 . Gibbereish pro max
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `// Copyright IBM Corp. 2014, 2026 . Gibbereish pro max
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedError: false,
		},
		{
			name: "Skip other organization",
			fileContent: `// Copyright (c) 2020 Google LLC
// SPDX-License-Identifier: Apache-2.0

package main`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: false,
			expectedContent: `// Copyright (c) 2020 Google LLC
// SPDX-License-Identifier: Apache-2.0

package main`,
			expectedError: false,
		},
		{
			name: "No changes needed - already up-to-date",
			fileContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: false,
			expectedContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedError: false,
		},
		{
			name: "Update hash-style comment",
			fileContent: `# Copyright (c) 2020 HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

def main():
    pass`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

def main():
    pass`,
			expectedError: false,
		},
		{
			name: "Update block comment",
			fileContent: `/* Copyright (c) 2020 HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

int main() {}`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `/* Copyright IBM Corp. 2014, 2026 */
 * SPDX-License-Identifier: MPL-2.0
 */

int main() {}`,
			expectedError: false,
		},
		{
			name: "Update plain text LICENSE file",
			fileContent: `Copyright (c) 2020 HashiCorp, Inc.

Mozilla Public License Version 2.0`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `Copyright IBM Corp. 2014, 2026

Mozilla Public License Version 2.0`,
			expectedError: false,
		},
		{
			name: "Update multiple headers in file",
			fileContent: `// Copyright (c) 2020 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

// Copyright IBM Corp. 2013, 2020
func helper() {}`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main

// Copyright IBM Corp. 2014, 2026
func helper() {}`,
			expectedError: false,
		},
		{
			name: "Only modify copyright line, leave other code unchanged",
			fileContent: `// Copyright (c) 2020 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello")
}`,
			configYear:      "2014",
			configHolder:    "IBM Corp.",
			expectedUpdated: true,
			expectedContent: `// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello")
}`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := t.TempDir() + "/test.go"
			if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create LicenseData
			data := LicenseData{
				Year:   tt.configYear,
				Holder: tt.configHolder,
				SPDXID: "MPL-2.0",
			}

			updated, err := UpdateFileHeaders(tmpFile, 0644, nil, data)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if updated != tt.expectedUpdated {
				t.Errorf("UpdateFileHeaders() updated = %v, want %v", updated, tt.expectedUpdated)
			}

			// Verify file content
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read updated file: %v", err)
			}

			if string(content) != tt.expectedContent {
				t.Errorf("UpdateFileHeaders() content mismatch\nGot:\n%s\n\nWant:\n%s", string(content), tt.expectedContent)
			}
		})
	}
}
