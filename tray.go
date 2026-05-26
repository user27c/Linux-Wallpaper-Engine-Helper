package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path"

	"fyne.io/systray"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/disintegration/imaging"
)

// Whether the app is actually quitting (vs just hiding)
var appQuitting bool = false

// Stores the volume before muting so it can be restored
var previousVolume int64 = 100

func onTrayReady() {
	systray.SetTitle("WPE")
	systray.SetTooltip("Linux Wallpaper Engine Helper")
	systray.SetIcon(createTrayIcon())

	mShow := systray.AddMenuItem("显示主窗口", "显示主窗口")
	mRandom := systray.AddMenuItem("随机壁纸", "应用随机壁纸")
	mMute := systray.AddMenuItem("一键静音", "静音/取消静音")
	mStop := systray.AddMenuItem("关闭动态壁纸", "停止当前壁纸引擎")
	mRestart := systray.AddMenuItem("一键重启底层修复", "杀死卡死的壁纸引擎并重新应用当前壁纸")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出应用")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				glib.IdleAdd(func() {
					showMainWindow()
				})
			case <-mRandom.ClickedCh:
				go applyRandomWallpaper()
			case <-mMute.ClickedCh:
				if Config.SavedUIState.Volume > 0 {
					previousVolume = Config.SavedUIState.Volume
					Config.SavedUIState.Volume = 0
					mMute.SetTitle("取消静音")
					setPipeWireMute(true)
					updateVolumeUI(0)
				} else {
					Config.SavedUIState.Volume = previousVolume
					mMute.SetTitle("一键静音")
					setPipeWireMute(false)
					setPipeWireVolume(previousVolume)
					updateVolumeUI(previousVolume)
				}
			case <-mStop.ClickedCh:
				log.Println("Stopping wallpaper engine from tray...")
				if err := tryKillProcesses("linux-wallpaperengine"); err != nil {
					log.Printf("Error stopping wallpaper engine: %v", err)
				} else {
					log.Println("Wallpaper engine stopped")
				}
			case <-mRestart.ClickedCh:
				log.Println("Restarting wallpaper engine to fix issues from tray...")
				if err := tryKillProcesses("linux-wallpaperengine"); err != nil {
					log.Printf("Error stopping wallpaper engine: %v", err)
				}
				go restoreWallpaper()
			case <-mQuit.ClickedCh:
				glib.IdleAdd(func() {
					quitApp()
				})
			}
		}
	}()
}

func onTrayExit() {
	log.Println("Tray exited")
}

// Shows the main window from the tray.
func showMainWindow() {
	if MainWindow != nil {
		MainWindow.SetVisible(true)
		MainWindow.Present()
	}
}

// Actually quits the application.
func quitApp() {
	appQuitting = true
	saveConfig()
	log.Println("Stopping wallpaper engine on helper exit...")
	if err := tryKillProcesses("linux-wallpaperengine"); err != nil {
		log.Printf("Error stopping wallpaper engine on exit: %v", err)
	}
	if MainWindow != nil {
		app := MainWindow.Application()
		if app != nil {
			app.Release() // Balance the Hold() call
			app.Quit()
		}
	}
	systray.Quit()
}

// Loads the tray icon from image.png and resizes to 48x48 for the tray.
// Falls back to a simple generated icon if file not found.
func createTrayIcon() []byte {
	// Search paths for the icon
	searchPaths := []string{}

	if exePath, err := os.Executable(); err == nil {
		searchPaths = append(searchPaths, path.Join(path.Dir(exePath), "image.png"))
	}

	homeDir := os.Getenv("HOME")
	searchPaths = append(searchPaths,
		path.Join(homeDir, ".local", "share", "icons", "hicolor", "256x256", "apps", "linux-wallpaperengine-helper.png"),
	)

	for _, iconPath := range searchPaths {
		img, err := imaging.Open(iconPath)
		if err != nil {
			continue
		}
		// Resize to 48x48 for tray
		resized := imaging.Resize(img, 48, 48, imaging.Lanczos)
		var buf bytes.Buffer
		if err := png.Encode(&buf, resized); err == nil {
			log.Printf("Loaded tray icon from: %s (resized to 48x48)", iconPath)
			return buf.Bytes()
		}
	}

	// Fallback: generate a simple icon
	log.Println("Icon file not found, using fallback icon")
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	c := color.RGBA{R: 80, G: 140, B: 255, A: 255}
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
