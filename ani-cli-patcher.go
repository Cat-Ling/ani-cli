package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

const newLauncherFunc = `launcher() {
    # This is the fzf replacement for a-shell.
    # It presents a numbered list for selection.
    list=$(cat -)
    item_count=$(echo "$list" | wc -l | tr -d ' ')

    if [ "$item_count" -eq 0 ]; then
        echo "No items to select from." >&2
        return 1
    fi

    echo "$list" | nl -w 2 -s '. '

    while true; do
        printf "%s (1-%d): " "$2" "$item_count"
        read -r selection

        if [ "$selection" -ge 1 ] && [ "$selection" -le "$item_count" ] 2>/dev/null; then
            echo "$list" | sed -n "${selection}p"
            return 0
        else
            echo "Invalid selection. Please try again." >&2
        fi
    done
}`

const newDownloadFunc = `download() {
    # download subtitle if it's set
    [ -n "$subtitle" ] && curl -s "$subtitle" -o "$download_dir/$2.vtt"
    case $1 in
        *m3u8*)
            if command -v "yt-dlp" >/dev/null; then
                yt-dlp --referer "$m3u8_refr" "$1" --no-skip-unavailable-fragments --fragment-retries infinite -N 16 -o "$download_dir/$2.mp4"
            else
                ffmpeg -extension_picky 0 -referer "$m3u8_refr" -loglevel error -stats -i "$1" -c copy "$download_dir/$2.mp4"
            fi
            ;;
        *)
            # Patched to use curl instead of aria2c
            echo "Downloading with curl..."
            curl -L -e "$allanime_refr" -o "$download_dir/$2.mp4" "$1"
            ;;
    esac
}`

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path_to_ani_cli_script>\n", os.Args[0])
		os.Exit(1)
	}

	scriptPath := os.Args[1]
	content, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	scriptContent := string(content)

	// --- Patch 1: Replace UI functions ---
	// Remove external_menu and replace launcher with our numbered list version.
	scriptContent = regexp.MustCompile(`(?s)external_menu\(\) \{.*?^}\n`).ReplaceAllString(scriptContent, "")
	scriptContent = regexp.MustCompile(`(?s)launcher\(\) \{.*?^}\n`).ReplaceAllString(scriptContent, newLauncherFunc+"\n")

	// --- Patch 2: Fix the history path ---
	scriptContent = regexp.MustCompile(`\${XDG_STATE_HOME:-\$HOME/\.local/state}`).ReplaceAllString(scriptContent, `\$HOME/Documents`)

	// --- Patch 3: Replace the download function ---
	scriptContent = regexp.MustCompile(`(?s)download\(\) \{.*?esac\n}`).ReplaceAllString(scriptContent, newDownloadFunc)

	// --- Patch 4: Add a-shell player support ---
	// Add 'ashell-vlc' to the player detection case statement
	playerDetectRegex := regexp.MustCompile(`(\*ish\*\) player_function="\${ANI_CLI_PLAYER:-iSH}" ;;)`)
	scriptContent = playerDetectRegex.ReplaceAllString(scriptContent, "$1\n    *a-shell*) player_function=\"ashell-vlc\" ;;")

	// Add a case for 'ashell-vlc' in the play_episode function
	playEpisodeRegex := regexp.MustCompile(`(catt\) nohup catt cast.*?\n\s*;;
)`)
	scriptContent = playEpisodeRegex.ReplaceAllString(scriptContent, "$1        iSH)\n            printf \"\\e]8;;vlc://%s\\a~~~~~~~~~~~~~~~~~~~~\\n~ Tap to open VLC ~\\n~~~~~~~~~~~~~~~~~~~~\\e]8;;\\a\\n\" \"$episode\"\n            sleep 5\n            ;;\n        ashell-vlc)\n            open \"vlc://${episode}\" ;;\n")

	// --- Patch 5: Remove fzf dependency check ---
	scriptContent = regexp.MustCompile(`dep_ch "fzf" \|\| true\n`).ReplaceAllString(scriptContent, "# fzf dependency removed by patcher\n")

	fmt.Print(scriptContent)
}
