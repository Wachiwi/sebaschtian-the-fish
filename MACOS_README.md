# Running Fish on macOS

This is a stripped-down version of the fish that works on macOS without GPIO support.

## Code Structure

The codebase now uses Go build tags to support both Linux and macOS:

- **`cmd/fish/common.go`** - Shared logic for both platforms (phrases, playlist, fish cycle)
- **`cmd/fish/main.go`** - Linux-specific entry point (`//go:build linux`)
- **`cmd/fish/main_darwin.go`** - macOS-specific entry point (`//go:build darwin`)
- **`pkg/fish/fish.go`** - Linux implementation with GPIO
- **`pkg/fish/fish_darwin.go`** - macOS implementation with mock motors

## What works on macOS:
- ✅ Audio playback (WAV and MP3 files)
- ✅ Text-to-speech (if you have a Piper server running)
- ✅ Playlist management
- ✅ Cron-based scheduling
- ✅ Mock motor controls (logs actions instead of using GPIO)

## What's disabled:
- ❌ GPIO motor controls (replaced with logging)

## Setup:

1. **Create the sound-data directory**:
   ```bash
   mkdir -p sound-data
   ```

2. **Add some audio files** to the `sound-data` directory:
   ```bash
   cp /path/to/your/music/*.mp3 sound-data/
   # or *.wav files
   ```

3. **Build and run**:
   ```bash
   # Option 1: Build then run
   go build -o fish-mac ./cmd/fish
   ./fish-mac
   
   # Option 2: Run directly (without building)
   go run ./cmd/fish
   ```
   
   **Note:** Use `go run ./cmd/fish` (directory), NOT `go run cmd/fish/main.go` (single file).
   The directory approach ensures all files (common.go, main_darwin.go) are compiled together.

## Optional: Running with Text-to-Speech

By default, TTS is **disabled** on macOS. To enable it:

### Option 1: Enable TTS in the code
Edit `cmd/fish/main_darwin.go` and change:
```go
enableTTS := false // Set to true if you have piper running
```
to:
```go
enableTTS := true
```

Then rebuild:
```bash
go build -o fish-mac ./cmd/fish
```

### Option 2: Run a local Piper server
You'll need to set up and run the Piper server locally on port 5000 for TTS to work.

## How it works:

The fish will:
- Run every minute (via cron)
- Either play a random audio file from `sound-data/` OR say a random phrase
- Keep track of what it's played recently to avoid repetition
- Log all motor movements (but not actually move anything since there's no GPIO)

## Testing without waiting for cron:

You can modify the cron schedule in `main_darwin.go` to run more frequently for testing:
```go
// Change from "* * * * *" (every minute) to "*/10 * * * * *" (every 10 seconds)
c.AddFunc("*/10 * * * * *", func() {
```

## Notes:

- The playlist files will be created in `./sound-data/played.json` and `./sound-data/queue.json`
- All motor actions are logged with `[MOCK]` prefix so you can see what would happen
- Audio will play through your Mac's default audio output
- The same `go build ./cmd/fish` command works on both Linux and macOS thanks to build tags!
