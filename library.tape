Output library.gif

Set FontSize 12
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"

# Start the app and let cards load
Type "go run ./cmd/main.go"
Enter
Sleep 4s

# Browse through cards
Down
Down
Down
Sleep 200ms
Down
Down
Down
Sleep 200ms
Down
Down
Down
Sleep 500ms

# Text search — find "Banshee"
Type "t"
Sleep 300ms
Type "Banshee"
Sleep 500ms
Enter
Sleep 800ms

# Scroll through results
Down
Down
Sleep 400ms
Down
Down
Sleep 400ms

# Clear back to full list
Type "c"
Sleep 800ms

# Open filter palette → Filter by Color → select Blue + Red (multi-select)
Type "f"
Sleep 400ms
Down
Enter
Sleep 300ms
Down
Sleep 200ms
Space
Sleep 200ms
Down
Down
Sleep 200ms
Space
Sleep 200ms
Enter
Sleep 800ms

# Browse the blue+red filtered results
Down
Down
Sleep 300ms
Down
Down
Sleep 300ms

# Find pilot links for the selected unit
Type "l"
Sleep 800ms

# Navigate the pilot results
Down
Sleep 300ms
Down
Sleep 300ms

# Clear all filters
Type "c"
Sleep 800ms

# Quit
Type "q"
