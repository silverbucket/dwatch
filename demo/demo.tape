Output demo/demo.gif

Set FontSize 14
Set Width 960
Set Height 620
Set Theme "Catppuccin Mocha"
Set Shell "bash"
Set Padding 20

# Invisible setup: seed demo data, set a neutral prompt, alias dwatch, then clear
# Run from the project root: vhs demo/demo.tape
Hide
Type "mkdir -p /tmp/dwatch-demo && cp demo/data/*.json /tmp/dwatch-demo/"
Enter
Type "export PS1='$ '"
Enter
Type "alias dwatch='/usr/local/bin/dwatch --data-dir /tmp/dwatch-demo'"
Enter
Sleep 400ms
Ctrl+L
Sleep 200ms
Show

Sleep 600ms

# --- status: snapshot overview + largest dirs + growth + freed ---
Type "dwatch status"
Sleep 150ms
Enter
Sleep 2s
Sleep 2s

Ctrl+L
Sleep 400ms

# --- status --limit: show more entries ---
Type "dwatch status --limit 10"
Sleep 150ms
Enter
Sleep 2s
Sleep 2s

Ctrl+L
Sleep 400ms

# --- top: ranked growth over the past week ---
Type "dwatch top --since 1w"
Sleep 150ms
Enter
Sleep 2s
Sleep 1.5s

Ctrl+L
Sleep 400ms

# --- top --by pct: catch fast-growing small dirs ---
Type "dwatch top --since 1w --by pct"
Sleep 150ms
Enter
Sleep 2s
Sleep 2s
