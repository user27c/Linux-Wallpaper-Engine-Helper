package main

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var Dialog *gtk.Window = nil

// Whether a reloadWallpaperData() is required
var reloadRequired bool = false

// Whether a refreshWallpaperDisplay() is required
var refreshRequired bool = false

// Whether a filterWallpapersBySearch(SearchQuery) is required
var filterRequired bool = false

// Shows the options window as a "Modal".
//
// Sets the parent to be the MainWindow and ensures that you are not able to interact with the MainWindow while this is open.
func showOptionsDialog() {
	Dialog = gtk.NewWindow()
	Dialog.SetTitle("选项")
	Dialog.SetDefaultSize(600, 400)
	Dialog.SetHExpand(true)
	Dialog.SetVExpand(true)

	Dialog.Connect("close-request", func() bool {
		validateConfig()
		if reloadRequired || refreshRequired {
			if reloadRequired {
				if err := reloadWallpaperData(); err != nil {
					log.Printf("Error reloading wallpaper data: %v", err)
					showFrontError(err.Error())
				}
			}
			if refreshRequired {
				refreshWallpaperDisplay()
			}
		} else if filterRequired {
			filterWallpapersBySearch(SearchQuery)
		}
		return false
	})

	notebook := gtk.NewNotebook()

	notebook.AppendPage(createUIPage(), gtk.NewLabel("用户界面"))
	notebook.AppendPage(createConstantsPage(), gtk.NewLabel("常量设置"))
	notebook.AppendPage(createPostProcessingPage(), gtk.NewLabel("后处理"))

	closeButton := gtk.NewButtonWithLabel("关闭")
	closeButton.SetHAlign(gtk.AlignEnd)
	closeButton.SetMarginTop(10)
	closeButton.SetMarginBottom(10)
	closeButton.SetMarginEnd(10)
	closeButton.Connect("clicked", func() {
		Dialog.Close()
	})

	vBox := gtk.NewBox(gtk.OrientationVertical, 0)
	vBox.Append(notebook)
	vBox.Append(closeButton)

	Dialog.SetChild(vBox)
	Dialog.SetTransientFor(&MainWindow.Window)
	Dialog.SetModal(true)
	Dialog.SetDestroyWithParent(true)
	Dialog.SetVisible(true)
}

// Creates the UI settings page, containing options for Config.SavedUIState and some quick actions
func createUIPage() *gtk.Box {
	uiPage := gtk.NewBox(gtk.OrientationVertical, 0)
	uiPage.SetMarginTop(10)
	uiPage.SetMarginBottom(10)
	uiPage.SetMarginStart(10)
	uiPage.SetMarginEnd(10)
	uiPage.SetSpacing(10)
	uiPage.SetHExpand(true)
	uiPage.SetVExpand(true)
	uiPage.SetHAlign(gtk.AlignFill)

	uiPage.Append(addNewSectionLabel("开关"))

	hideBrokenToggle := gtk.NewCheckButtonWithLabel("隐藏已损坏的壁纸")
	hideBrokenToggle.SetHAlign(gtk.AlignStart)
	hideBrokenToggle.SetActive(Config.SavedUIState.HideBroken)
	hideBrokenToggle.Connect("toggled", func() {
		Config.SavedUIState.HideBroken = hideBrokenToggle.Active()
		filterRequired = true
	})
	uiPage.Append(hideBrokenToggle)

	uiPage.Append(addNewSectionLabel("快捷操作"))

	restoreButton := gtk.NewButtonWithLabel("恢复上次壁纸")
	restoreButton.SetHExpand(false)
	restoreButton.SetVExpand(false)
	restoreButton.SetHAlign(gtk.AlignStart)
	restoreButton.Connect("clicked", func() {
		log.Println("Restoring last set wallpaper...")
		go restoreWallpaper()
	})
	uiPage.Append(restoreButton)

	resetBrokenButton := gtk.NewButtonWithLabel("重置损坏标记（不可撤销！）")
	resetBrokenButton.SetHExpand(false)
	resetBrokenButton.SetVExpand(false)
	resetBrokenButton.SetHAlign(gtk.AlignStart)
	resetBrokenButton.Connect("clicked", func() {
		// i have to use NewMessageDialog even though its deprecated
		// the reason is cause NewAlertDialog does not exist
		// so I'm not sure how i would create gtk.AlertDialog effectively
		//
		// Not the only one with this issue, as i found the following issue
		// see https://github.com/diamondburned/gotk4/issues/165
		dialog := gtk.NewMessageDialog(Dialog, gtk.DialogModal, gtk.MessageWarning, gtk.ButtonsYesNo)
		dialog.SetTitle("确认重置")
		dialogMessage := gtk.NewLabel("确定要重置所有已损坏壁纸的标记吗？此操作不可撤销！")
		if dialogBox, ok := dialog.MessageArea().(*gtk.Box); ok {
			dialogBox.Append(dialogMessage)
		} else {
			log.Println("Failed to set message area for dialog")
			dialog.SetTitle("确定要重置所有已损坏壁纸的标记吗？此操作不可撤销！")
		}

		dialog.Connect("response", func(response gtk.ResponseType) {
			if response == gtk.ResponseYes {
				log.Println("Resetting broken wallpapers...")
				Config.SavedUIState.Broken = []string{}
				reloadRequired = true
				refreshRequired = true
			} else {
				log.Println("Reset broken wallpapers cancelled")
			}
			dialog.Destroy()
		})

		dialog.SetVisible(true)
	})
	uiPage.Append(resetBrokenButton)

	resetFavoritesButton := gtk.NewButtonWithLabel("重置收藏（不可撤销！）")
	resetFavoritesButton.SetHExpand(false)
	resetFavoritesButton.SetVExpand(false)
	resetFavoritesButton.SetHAlign(gtk.AlignStart)
	resetFavoritesButton.Connect("clicked", func() {
		// i have to use NewMessageDialog even though its deprecated
		// the reason is cause NewAlertDialog does not exist
		// so I'm not sure how i would create gtk.AlertDialog effectively
		//
		// Not the only one with this issue, as i found the following issue
		// see https://github.com/diamondburned/gotk4/issues/165
		dialog := gtk.NewMessageDialog(Dialog, gtk.DialogModal, gtk.MessageWarning, gtk.ButtonsYesNo)
		dialog.SetTitle("确认重置")
		dialogMessage := gtk.NewLabel("确定要重置所有收藏吗？此操作不可撤销！")
		if dialogBox, ok := dialog.MessageArea().(*gtk.Box); ok {
			dialogBox.Append(dialogMessage)
		} else {
			log.Println("Failed to set message area for dialog")
			dialog.SetTitle("确定要重置所有收藏吗？此操作不可撤销！")
		}

		dialog.Connect("response", func(response gtk.ResponseType) {
			if response == gtk.ResponseYes {
				log.Println("Resetting favorites...")
				Config.SavedUIState.Favorites = []string{}
				reloadRequired = true
				refreshRequired = true
			} else {
				log.Println("Reset favorites cancelled")
			}
			dialog.Destroy()
		})

		dialog.SetVisible(true)
	})
	uiPage.Append(resetFavoritesButton)

	return uiPage
}

// Creates the Constants page, containing options for Config.Constants
func createConstantsPage() *gtk.Box {
	constantsPage := gtk.NewBox(gtk.OrientationVertical, 0)
	constantsPage.SetMarginTop(10)
	constantsPage.SetMarginBottom(10)
	constantsPage.SetMarginStart(10)
	constantsPage.SetMarginEnd(10)
	constantsPage.SetSpacing(10)
	constantsPage.SetHExpand(true)
	constantsPage.SetVExpand(true)
	constantsPage.SetHAlign(gtk.AlignFill)

	constantsPage.Append(addNewSectionLabel("开关"))

	discardProcessLogsToggle := gtk.NewCheckButtonWithLabel("丢弃进程日志（stdout 重定向至 /dev/null）")
	discardProcessLogsToggle.SetHAlign(gtk.AlignStart)
	discardProcessLogsToggle.SetActive(Config.Constants.DiscardProcessLogs)
	discardProcessLogsToggle.Connect("toggled", func() {
		Config.Constants.DiscardProcessLogs = discardProcessLogsToggle.Active()
	})
	constantsPage.Append(discardProcessLogsToggle)

	constantsPage.Append(addNewSectionLabel("壁纸引擎可执行文件"))

	wallpaperEngineBinaryEntry := gtk.NewEntry()
	wallpaperEngineBinaryEntry.SetText(Config.Constants.LinuxWallpaperEngineBin)
	wallpaperEngineBinaryEntry.SetEditable(true)
	wallpaperEngineBinaryEntry.SetHExpand(true)
	wallpaperEngineBinaryEntry.SetHAlign(gtk.AlignFill)
	wallpaperEngineBinaryEntry.Connect("changed", func() {
		Config.Constants.LinuxWallpaperEngineBin = wallpaperEngineBinaryEntry.Text()
	})
	wallpaperEngineBinaryEntry.SetPlaceholderText("linux-wallpaperengine（须在 PATH 中）")
	constantsPage.Append(wallpaperEngineBinaryEntry)

	constantsPage.Append(addNewSectionLabel("屏幕输出"))

	screenRootEntry := gtk.NewEntry()
	screenRootEntry.SetText(Config.Constants.ScreenRoot)
	screenRootEntry.SetEditable(true)
	screenRootEntry.SetHExpand(true)
	screenRootEntry.SetHAlign(gtk.AlignFill)
	screenRootEntry.Connect("changed", func() {
		Config.Constants.ScreenRoot = screenRootEntry.Text()
	})
	screenRootEntry.SetPlaceholderText("留空自动检测（例如 eDP-1、HDMI-A-1）")
	constantsPage.Append(screenRootEntry)

	constantsPage.Append(addNewSectionLabel("壁纸引擎内容目录"))

	wallpaperEngineDirBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	wallpaperEngineDirBox.SetHExpand(true)
	wallpaperEngineDirBox.SetVExpand(false)
	constantsPage.Append(wallpaperEngineDirBox)

	wallpaperEngineDirEntry := gtk.NewEntry()
	wallpaperEngineDirEntry.SetText(Config.Constants.WallpaperEngineDir)
	wallpaperEngineDirEntry.SetEditable(false)
	wallpaperEngineDirEntry.SetHExpand(true)
	wallpaperEngineDirEntry.SetHAlign(gtk.AlignFill)

	wallpaperEngineDirButton := gtk.NewButtonFromIconName("folder-open")
	wallpaperEngineDirButton.SetHExpand(false)
	wallpaperEngineDirButton.SetVExpand(false)
	wallpaperEngineDirButton.SetHAlign(gtk.AlignStart)
	wallpaperEngineDirButton.SetSizeRequest(24, 24)
	wallpaperEngineDirButton.Connect("clicked", func() {
		filer := gio.NewFileForPath(Config.Constants.WallpaperEngineDir)

		// open a file dialog to select a new screenshot file
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("选择壁纸引擎壁纸加载路径")
		fileDialog.SetAcceptLabel("选择")
		fileDialog.SetModal(true)
		fileDialog.SetInitialFolder(filer)
		fileDialog.SelectFolder(context.TODO(), Dialog, func(result gio.AsyncResulter) {
			selectedFile, err := fileDialog.SelectFolderFinish(result)
			if err != nil {
				log.Printf("Failed to save screenshot file: %v", err)
				return
			}
			if selectedFile.Path() != "" {
				Config.Constants.WallpaperEngineDir = selectedFile.Path()
				wallpaperEngineDirEntry.SetText(Config.Constants.WallpaperEngineDir)
				reloadRequired = true
				refreshRequired = true
			}
		})
	})
	wallpaperEngineDirBox.Append(wallpaperEngineDirButton)
	wallpaperEngineDirBox.Append(wallpaperEngineDirEntry)

	constantsPage.Append(addNewSectionLabel("壁纸引擎资源目录"))

	wallpaperEngineAssetsBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	wallpaperEngineAssetsBox.SetHExpand(true)
	wallpaperEngineAssetsBox.SetVExpand(false)
	constantsPage.Append(wallpaperEngineAssetsBox)

	wallpaperEngineAssetsEntry := gtk.NewEntry()
	wallpaperEngineAssetsEntry.SetText(Config.Constants.WallpaperEngineAssets)
	wallpaperEngineAssetsEntry.SetEditable(false)
	wallpaperEngineAssetsEntry.SetHExpand(true)
	wallpaperEngineAssetsEntry.SetHAlign(gtk.AlignFill)

	wallpaperEngineAssetsButton := gtk.NewButtonFromIconName("folder-open")
	wallpaperEngineAssetsButton.SetHExpand(false)
	wallpaperEngineAssetsButton.SetVExpand(false)
	wallpaperEngineAssetsButton.SetHAlign(gtk.AlignStart)
	wallpaperEngineAssetsButton.SetSizeRequest(24, 24)
	wallpaperEngineAssetsButton.Connect("clicked", func() {
		filer := gio.NewFileForPath(Config.Constants.WallpaperEngineAssets)

		// open a file dialog to select a new screenshot file
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("选择壁纸引擎资源加载路径")
		fileDialog.SetAcceptLabel("选择")
		fileDialog.SetModal(true)
		fileDialog.SetInitialFolder(filer)
		fileDialog.SelectFolder(context.TODO(), Dialog, func(result gio.AsyncResulter) {
			selectedFile, err := fileDialog.SelectFolderFinish(result)
			if err != nil {
				log.Printf("Failed to save screenshot file: %v", err)
				return
			}
			if selectedFile.Path() != "" {
				Config.Constants.WallpaperEngineAssets = selectedFile.Path()
				wallpaperEngineAssetsEntry.SetText(Config.Constants.WallpaperEngineAssets)
			}
		})
	})
	wallpaperEngineAssetsBox.Append(wallpaperEngineAssetsButton)
	wallpaperEngineAssetsBox.Append(wallpaperEngineAssetsEntry)

	return constantsPage
}

// Creates the Post Processing page, containing options for Config.PostProcessing
func createPostProcessingPage() *gtk.Box {
	postProcessingPage := gtk.NewBox(gtk.OrientationVertical, 0)
	postProcessingPage.SetMarginTop(10)
	postProcessingPage.SetMarginBottom(10)
	postProcessingPage.SetMarginStart(10)
	postProcessingPage.SetMarginEnd(10)
	postProcessingPage.SetSpacing(10)
	postProcessingPage.SetHExpand(true)
	postProcessingPage.SetVExpand(true)
	postProcessingPage.SetHAlign(gtk.AlignFill)

	postProcessingPage.Append(addNewSectionLabel("开关"))

	postProcessingEnabled := gtk.NewCheckButtonWithLabel("启用后处理")
	postProcessingEnabled.SetHAlign(gtk.AlignStart)
	postProcessingEnabled.SetActive(Config.PostProcessing.Enabled)
	postProcessingEnabled.Connect("toggled", func() {
		Config.PostProcessing.Enabled = postProcessingEnabled.Active()
	})
	postProcessingPage.Append(postProcessingEnabled)

	setSWWWEnabled := gtk.NewCheckButtonWithLabel("将 swww 壁纸设为截图文件")
	setSWWWEnabled.SetHAlign(gtk.AlignStart)
	setSWWWEnabled.SetActive(Config.PostProcessing.SetSWWW)
	setSWWWEnabled.Connect("toggled", func() {
		Config.PostProcessing.SetSWWW = setSWWWEnabled.Active()
	})
	postProcessingPage.Append(setSWWWEnabled)

	postProcessingPage.Append(addNewSectionLabel("人工延迟"))

	artificialDelayBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	postProcessingPage.Append(artificialDelayBox)

	artificialDelayWarning := gtk.NewImageFromIconName("dialog-warning-symbolic")
	artificialDelayWarning.SetHExpand(false)
	artificialDelayWarning.SetVExpand(false)
	artificialDelayWarning.SetHAlign(gtk.AlignStart)
	artificialDelayWarning.SetSizeRequest(24, 24)
	artificialDelayWarning.SetVisible(false)

	artificialDelayEntry := gtk.NewEntry()
	artificialDelayEntry.SetText((time.Duration(Config.PostProcessing.ArtificialDelay) * time.Second).String())
	artificialDelayEntry.SetEditable(true)
	artificialDelayEntry.SetHExpand(true)
	artificialDelayEntry.SetHAlign(gtk.AlignFill)
	artificialDelayEntry.SetPlaceholderText("输入人工延迟时间（例如 2s、1m）（至少 1 秒）")
	artificialDelayEntry.Connect("changed", func() {
		delay, err := time.ParseDuration(artificialDelayEntry.Text())
		if err != nil {
			artificialDelayWarning.SetTooltipText("无效的时间格式")
			artificialDelayEntry.AddCSSClass("error")
			artificialDelayWarning.SetVisible(true)
			return
		}

		Config.PostProcessing.ArtificialDelay = int64(delay / time.Second)
		artificialDelayEntry.RemoveCSSClass("error")
		artificialDelayWarning.SetVisible(false)
	})
	artificialDelayBox.Append(artificialDelayEntry)
	artificialDelayBox.Append(artificialDelayWarning)

	postProcessingPage.Append(addNewSectionLabel("截图文件（清空即禁用）"))

	screenshotFileList := gtk.NewFlowBox()
	screenshotFileList.SetHAlign(gtk.AlignFill)
	screenshotFileList.SetOrientation(gtk.OrientationHorizontal)
	screenshotFileList.SetSelectionMode(gtk.SelectionNone)
	screenshotFileList.SetColumnSpacing(4)
	screenshotFileList.SetRowSpacing(4)
	screenshotFileList.SetMinChildrenPerLine(1)
	screenshotFileList.SetMaxChildrenPerLine(1)
	screenshotFileList.SetHomogeneous(true)
	screenshotFileList.SetHExpand(true)
	screenshotFileList.SetVExpand(false)
	refreshScreenshotFilesList(screenshotFileList)
	postProcessingPage.Append(screenshotFileList)

	postProcessingPage.Append(addNewSectionLabel("后处理命令"))

	postCommandEntry := gtk.NewEntry()
	postCommandEntry.SetText(Config.PostProcessing.PostCommand)
	postCommandEntry.SetEditable(true)
	postCommandEntry.SetHExpand(true)
	postCommandEntry.SetHAlign(gtk.AlignFill)
	postCommandEntry.Connect("changed", func() {
		Config.PostProcessing.PostCommand = postCommandEntry.Text()
	})
	postCommandEntry.SetPlaceholderText("输入截图后执行的命令，留空即禁用")
	postProcessingPage.Append(postCommandEntry)

	return postProcessingPage
}

// Helper function to create a label with the provided text to ensure uniform styles.
func addNewSectionLabel(text string) *gtk.Label {
	label := gtk.NewLabel(text)
	label.SetMarkup("<b>" + escapeMarkup(label.Text()) + "</b>")
	label.SetHExpand(true)
	label.SetHAlign(gtk.AlignStart)
	label.SetMarginTop(10)
	label.SetMarginBottom(10)
	return label
}

// Helper function to create the items for the screenshot files list.
//
// Each item has a button the change the current item's location, a text input showing the location, and a remove button to remove the item.
//
// Also adds an "Add" button to add more screenshot locations to the list, appending Config.PostProcessing.ScreenshotFiles and refreshing the list.
func refreshScreenshotFilesList(screenshotFileList *gtk.FlowBox) {
	screenshotFileList.RemoveAll()

	for i, file := range Config.PostProcessing.ScreenshotFiles {
		hBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
		hBox.SetHExpand(true)
		hBox.SetVExpand(false)

		button := gtk.NewButtonFromIconName("document-open")
		button.SetHExpand(false)
		button.SetVExpand(false)
		button.SetHAlign(gtk.AlignStart)
		button.SetSizeRequest(24, 24)
		button.Connect("clicked", func() {
			filer := gio.NewFileForPath(file)

			fileDialog := gtk.NewFileDialog()
			fileDialog.SetTitle("选择截图保存位置")
			fileDialog.SetAcceptLabel("保存")
			fileDialog.SetModal(true)
			fileDialog.SetInitialFile(filer)
			fileDialog.Save(context.TODO(), Dialog, func(result gio.AsyncResulter) {
				selectedFile, err := fileDialog.SaveFinish(result)
				if err != nil {
					log.Printf("Failed to save screenshot file: %v", err)
					return
				}
				if selectedFile.Path() != "" {
					Config.PostProcessing.ScreenshotFiles[i] = selectedFile.Path()
					refreshScreenshotFilesList(screenshotFileList)
				}
			})
		})
		hBox.Append(button)

		label := gtk.NewEntry()
		label.SetText(file)
		label.SetEditable(false)
		label.SetHExpand(true)
		label.SetHAlign(gtk.AlignFill)
		hBox.Append(label)

		removeButton := gtk.NewButtonFromIconName("edit-delete")
		removeButton.SetHExpand(false)
		removeButton.SetVExpand(false)
		removeButton.SetHAlign(gtk.AlignEnd)
		removeButton.SetSizeRequest(24, 24)
		removeButton.Connect("clicked", func() {
			Config.PostProcessing.ScreenshotFiles = append(Config.PostProcessing.ScreenshotFiles[:i], Config.PostProcessing.ScreenshotFiles[i+1:]...)
			refreshScreenshotFilesList(screenshotFileList)
		})
		hBox.Append(removeButton)

		screenshotFileList.Append(hBox)
	}

	addButton := gtk.NewButtonFromIconName("list-add")
	addButton.SetHExpand(true)
	addButton.SetVExpand(false)
	addButton.SetHAlign(gtk.AlignFill)
	addButton.SetSizeRequest(-1, 24)
	addButton.Connect("clicked", func() {
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("选择截图保存位置")
		fileDialog.SetAcceptLabel("保存")
		fileDialog.SetModal(true)
		fileDialog.Save(context.TODO(), Dialog, func(result gio.AsyncResulter) {
			selectedFile, err := fileDialog.SaveFinish(result)
			if err != nil {
				log.Printf("Failed to save screenshot file: %v", err)
				return
			}
			if selectedFile.Path() != "" {
				Config.PostProcessing.ScreenshotFiles = append(Config.PostProcessing.ScreenshotFiles, selectedFile.Path())
				refreshScreenshotFilesList(screenshotFileList)
			}
		})
	})

	screenshotFileList.Append(addButton)
}
