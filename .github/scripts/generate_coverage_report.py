import json
import os
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

line_rate = round(covered / total * 100, 1) if total > 0 else 0.0

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
    "lines_total":   total,
}

with open("coverage-report.json", "w", encoding="utf-8") as f:
    json.dump(report, f, indent=2)
