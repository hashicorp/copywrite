import json
import os
import subprocess
import datetime

covered = 0
total = 0
with open("coverage.out", "r", encoding="utf-8") as f:
    next(f)  # skip mode line
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            _, num_stmts, num_executed = line.rsplit(" ", 2)
            num_stmts = int(num_stmts)
            num_executed = int(num_executed)
        except ValueError:
            continue
        total += num_stmts
        if num_executed > 0:
            covered += num_stmts

# Use go tool cover -func for line_rate so it matches the CI display value.
# It uses AST function boundaries and excludes non-function-level blocks,
# which is why it differs from the raw block count above.
out = subprocess.check_output(["go", "tool", "cover", "-func=coverage.out"], text=True)
line_rate = 0.0
for row in out.splitlines():
    if row.startswith("total:"):
        line_rate = float(row.split()[-1].rstrip("%"))
        break

# Derive lines_total so that lines_covered / lines_total == line_rate.
lines_total = round(covered / (line_rate / 100)) if line_rate > 0 else total

repo = os.environ["GITHUB_REPOSITORY"]
report = {
    "repo":          repo,
    "language":      "go",
    "date":          datetime.date.today().isoformat(),
    "commit":        os.environ.get("GITHUB_SHA", "")[:7],
    "branch":        os.environ.get("GITHUB_REF_NAME", "main"),
    "run_url":       f"https://github.com/{repo}/actions/runs/{os.environ['GITHUB_RUN_ID']}",
    "line_rate":     line_rate,
    "lines_covered": covered,
    "lines_total":   lines_total,
}

with open("coverage-report.json", "w", encoding="utf-8") as f:
    json.dump(report, f, indent=2)
