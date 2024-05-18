#!/bin/bash

cp ../../cmd/data/makai/sudoku.db .

while IFS="|" read -r K V; do
   command="DELETE FROM puzzles WHERE pid IN \
    (SELECT pid FROM puzzles WHERE puzzleset='$K' LIMIT -1 OFFSET $V)"
   echo "Running $command"
	sqlite3 "sudoku.db" "$command"
done <<EOF
17 Clue Pack 1|5
17 Clue Pack 2|6
17 Clue Pack 3|7
Hard Pack 1|8
Medium Pack 1|9
Super Easy Pack 1|10
EOF

sqlite3 "sudoku.db" "delete from users; delete from completions; delete from inprogress; VACUUM;"
