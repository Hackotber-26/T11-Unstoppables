# Migration Checker

`check.py` statically checks six-digit ordered migration filenames, contiguous versions, non-empty SQL, and common destructive statements. Destructive operations require a `-- destructive-reviewed: <reference>` SQL comment.

This command does not apply or reverse schema changes. The Go migration runner owns application; destructive migrations still require code review, backup, and a tested recovery plan.