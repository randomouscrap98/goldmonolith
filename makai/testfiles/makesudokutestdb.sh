#!/bin/bash

cp ../../cmd/data/makai/sudoku.db .

while IFS="|" read -r K V; do
	sqlite3 "sudoku.db" "delete from puzzles where pid not in \
    (select pid from puzzles where puzzleset='$K' limit -1 offset $V)"
done <<EOF
"17 Clue Pack 1" | 5
"17 Clue Pack 2" | 6
"17 Clue Pack 3" | 7
"Hard Pack 1" | 8
"Medium Pack 1" | 9
"Super Easy Pack 1" | 10
EOF

sqlite3 "sudoku.db" "delete from users; delete from completions; delete from inprogress; VACUUM;"
