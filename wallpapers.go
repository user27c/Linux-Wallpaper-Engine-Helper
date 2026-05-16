package main

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/image/bmp"
)

type WallpaperItem struct {
	WallpaperID   string
	WallpaperPath string
	CachedPath    string
	IsFavorite    bool
	IsBroken      bool
	ModTime       time.Time

	// get from project.json
	projectJson ProjectJSON
}

type ProjectJSON struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
	PreviewImage string   `json:"preview"`
}

var WallpaperItems []WallpaperItem = []WallpaperItem{}
var settingWallpaper bool = false

// Detects the primary screen output name using xrandr.
// Returns the first connected monitor name found, or "eDP-1" as a fallback.
func detectScreenOutput() string {
	cmd := exec.Command("xrandr", "--listmonitors")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to detect screen output via xrandr: %v, falling back to eDP-1", err)
		return "eDP-1"
	}

	// xrandr --listmonitors output format:
	// Monitors: 1
	//  0: +*eDP-1 2560/340x1600/210+0+0  eDP-1
	// We want the last field on lines starting with " 0:"
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "Monitors:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			screenName := fields[len(fields)-1]
			log.Printf("Auto-detected screen output: %s", screenName)
			return screenName
		}
	}

	log.Println("Could not parse xrandr output, falling back to eDP-1")
	return "eDP-1"
}

// Creates the command string to run linux-wallpaperengine with the given wallpaper path and volume.
// Also returns the path to the screenshot file that will be created by the command as the second return value.
func createWallpaperCommand(wallpaperPath string, volume float64) (string, string) {
	screenRoot := Config.Constants.ScreenRoot
	if screenRoot == "" {
		screenRoot = detectScreenOutput()
	}

	// Build options first, wallpaper path goes as positional argument at the end
	cmd := Config.Constants.LinuxWallpaperEngineBin + " --screen-root " + screenRoot

	if volume <= 1 {
		cmd += " --silent"
	} else {
		cmd += " --volume " + strconv.FormatFloat(volume, 'f', 0, 64)
	}

	cacheScreenshot := ""
	if Config.PostProcessing.Enabled {
		cacheScreenshot = path.Join(CacheDir, "screenshot.png")

		cmd += " --screenshot " + cacheScreenshot
	}

	if Config.Constants.WallpaperEngineAssets != "" {
		cmd += " --assets-dir " + Config.Constants.WallpaperEngineAssets
	}

	// Wallpaper path as positional argument (last)
	cmd += " " + wallpaperPath

	return cmd, cacheScreenshot
}

// Applies the wallpaper from the given wallpaperPath, with the specified volume.
//
// Returns nil if the wallpaper was successfully applied, an error otherwise.
func applyWallpaper(wallpaperPath string, volume float64) error {
	if settingWallpaper {
		return fmt.Errorf("another wallpaper is currently being set. Please wait before setting another wallpaper")
	}
	updateGUIStatusText("正在启动 linux-wallpaperengine...")
	settingWallpaper = true

	defer func() {
		updateGUIStatusText("双击壁纸即可应用。")
		settingWallpaper = false
	}()

	cmd, cacheScreenshot := createWallpaperCommand(wallpaperPath, volume)

	err := tryKillProcesses("linux-wallpaperengine")
	if err != nil {
		return fmt.Errorf("error trying to kill existing processes: %v", err)
	}

	log.Println("Executing command:", cmd)
	pid, err := runDetachedProcess("sh", "-c", cmd)
	if err != nil {
		// exit if we cannot start the command, to prevent multiple instances taking up resources
		return fmt.Errorf("error starting wallpaper command '%s': %v", cmd, err)
	} else {
		log.Printf("Successfully started detached wallpaper command (PID: %d): %s", pid, cmd)
	}

	// Wait briefly and check if the process crashed
	updateGUIStatusText("正在检测壁纸引擎状态...")
	time.Sleep(2 * time.Second)

	wallpaperId := path.Base(wallpaperPath)
	if err := syscall.Kill(pid, 0); err != nil {
		log.Printf("Wallpaper engine process (PID: %d) died shortly after starting", pid)
		updateGUIStatusText("⚠ 壁纸引擎崩溃，已自动标记为损坏: " + wallpaperId)

		// Auto-mark as broken
		if !slices.Contains(Config.SavedUIState.Broken, wallpaperId) {
			Config.SavedUIState.Broken = append(Config.SavedUIState.Broken, wallpaperId)
			for i := range WallpaperItems {
				if WallpaperItems[i].WallpaperID == wallpaperId {
					WallpaperItems[i].IsBroken = true
					break
				}
			}
			log.Printf("Auto-marked wallpaper %s as broken", wallpaperId)
		}

		return fmt.Errorf("wallpaper engine crashed for wallpaper %s", wallpaperId)
	}

	if Config.PostProcessing.Enabled {
		log.Println("Post-processing enabled, running post-processing...")

		if Config.PostProcessing.ArtificialDelay > 0 {
			updateGUIStatusText("正在等待后处理延迟...")
			log.Printf("Waiting for %d seconds before running post-processing...", Config.PostProcessing.ArtificialDelay)
			time.Sleep(time.Duration(Config.PostProcessing.ArtificialDelay) * time.Second)
		}
		updateGUIStatusText("正在执行后处理...")

		if len(Config.PostProcessing.ScreenshotFiles) > 0 && len(Config.PostProcessing.ScreenshotFiles[0]) > 0 {
			for _, filePath := range Config.PostProcessing.ScreenshotFiles {
				if path.Ext(filePath) == "" {
					filePath += ".png" // ensure the file has a .png extension
				}

				if !slices.Contains([]string{".png", ".jpg", ".jpeg", ".bmp"}, path.Ext(filePath)) {
					log.Printf("Unsupported file format for post-processing: %s", filePath)
					continue // skip unsupported formats
				}

				source, err := os.Open(cacheScreenshot)
				if err != nil {
					log.Printf("Error opening screenshot file for post-processing: %v", err)
					continue
				}
				defer source.Close()

				dest, err := os.Create(filePath)
				if err != nil {
					log.Printf("Error creating destination file for post-processing: %v", err)
					continue
				}
				defer dest.Close()

				if path.Ext(filePath) == ".png" {
					// no need for transcoding, just copy the file
					log.Printf("Copying screenshot to: %s", filePath)

					if _, err := io.Copy(dest, source); err != nil {
						log.Printf("Error copying screenshot file: %v", err)
						continue
					}

					log.Printf("Copied screenshot to: %s", filePath)
					continue
				} else {
					// transcoding the screenshot to the specified file format
					log.Printf("Transcoding screenshot to: %s", filePath)

					img, err := png.Decode(source)
					if err != nil {
						log.Printf("Error decoding PNG for transcoding: %v", err)
						continue
					}

					switch ext := path.Ext(filePath); ext {
					case ".jpg", ".jpeg":
						err = jpeg.Encode(dest, img, nil)
						if err != nil {
							log.Printf("Error encoding JPEG for transcoding: %v", err)
						}
						log.Printf("Successfully transcoded screenshot to: %s", filePath)
					case ".bmp":
						err = bmp.Encode(dest, img)
						if err != nil {
							log.Printf("Error encoding BMP for transcoding: %v", err)
						}
						log.Printf("Successfully transcoded screenshot to: %s", filePath)
					default:
						log.Printf("Unsupported file format: %s", ext)
					}
				}
			}
		}

		if Config.PostProcessing.PostCommand != "" {
			postCmdStr := replaceVariablesInString(Config.PostProcessing.PostCommand, map[string]string{
				"screenshot":    cacheScreenshot,
				"wallpaperPath": wallpaperPath,
				"wallpaperId":   path.Base(wallpaperPath),
				"volume":        strconv.FormatFloat(volume, 'f', 0, 64),
				"pid":           strconv.Itoa(pid),
			})

			log.Printf("Post-processing command: %s", postCmdStr)

			pid, err := runDetachedProcess("sh", "-c", postCmdStr)
			if err != nil {
				log.Printf("Error starting post-processing command '%s': %v", postCmdStr, err)
			} else {
				log.Printf("Successfully started post-processing command (PID: %d): %s", pid, postCmdStr)
			}
		}

		// set swww wallpaper if enabled
		if Config.PostProcessing.SetSWWW {
			setSWWW(cacheScreenshot)
		}
	}

	// Save the last set wallpaper ID
	Config.SavedUIState.LastSetId = path.Base(wallpaperPath)
	return nil
}

// Runs `swww img <screenshotPath>` to set swww's wallpaper.
// Returns true if the command was successfully started, false otherwise.
//
// Also ensures the swww-daemon is running.
func setSWWW(screenshotPath string) error {
	runningDaemons, err := getRunningProcessPids("swww-daemon")

	if err != nil {
		return fmt.Errorf("failed to check for swww-daemon running: %v", err)
	}

	if len(runningDaemons) < 1 {
		log.Println("swww-daemon not running, starting swww-daemon")
		pid, err := runDetachedProcess("swww-daemon")
		if err != nil {
			return fmt.Errorf("failed to start swww-daemon: %v", err)
		}

		log.Printf("Started swww-daemon [PID: %v]", pid)
	}

	pid, err := runDetachedProcess("swww", "img", screenshotPath)
	if err != nil {
		return fmt.Errorf("failed to start swww command: %v", err)
	}

	log.Printf("Started swww command [PID: %v]", pid)
	return nil
}

// Restores the last set wallpaper provided from Config.SavedUIState.LastSetId
//
// Returns nil if the wallpaper was successfully restored, an error otherwise.
func restoreWallpaper() error {
	if Config.SavedUIState.LastSetId == "" {
		return fmt.Errorf("no last set wallpaper ID found")
	}

	wallpaperPath, err := resolvePath(path.Join(Config.Constants.WallpaperEngineDir, Config.SavedUIState.LastSetId))
	if err != nil {
		return fmt.Errorf("failed to resolve wallpaper path: %v", err)
	}

	log.Printf("Restoring last set wallpaper: %s", wallpaperPath)
	return applyWallpaper(wallpaperPath, float64(Config.SavedUIState.Volume))
}

// Applies a random wallpaper from the available wallpapers.
//
// Requires WallpaperItems to be populated with available wallpapers.
//
// Returns nil if a random wallpaper was successfully applied, an error otherwise.
func applyRandomWallpaper() error {
	// Reload to pick up any new/deleted wallpapers
	if err := reloadWallpaperData(); err != nil {
		log.Printf("Error reloading wallpaper data for random: %v", err)
	}

	if len(WallpaperItems) == 0 {
		return fmt.Errorf("no wallpapers available to apply")
	}

	nonBrokenWallpapers := make([]WallpaperItem, 0)
	for _, item := range WallpaperItems {
		if !item.IsBroken {
			nonBrokenWallpapers = append(nonBrokenWallpapers, item)
		}
	}
	if len(nonBrokenWallpapers) == 0 {
		return fmt.Errorf("no non-broken wallpapers available to apply")
	}

	randomIndex := rand.Intn(len(nonBrokenWallpapers))
	wallpaper := nonBrokenWallpapers[randomIndex]

	log.Printf("Applying random wallpaper: %s", wallpaper.WallpaperID)
	return applyWallpaper(wallpaper.WallpaperPath, float64(Config.SavedUIState.Volume))
}
