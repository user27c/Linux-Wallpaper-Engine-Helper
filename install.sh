#!/usr/bin/env bash
set -euo pipefail

APP_NAME="linux-wallpaperengine-helper"
APP_DISPLAY_NAME="Wallpaper Engine Helper"
APP_COMMENT="A GUI helper for linux-wallpaperengine"
APP_ID="dev._6gh.linux-wallpaperengine-helper"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ICON_SRC="${SCRIPT_DIR}/image.png"

INSTALL_DIR="${HOME}/.local/bin"
ICON_DIR="${HOME}/.local/share/icons/hicolor/256x256/apps"
DESKTOP_DIR="${HOME}/.local/share/applications"

INSTALL_BIN="${INSTALL_DIR}/${APP_NAME}"
INSTALL_ICON="${ICON_DIR}/${APP_NAME}.png"
INSTALL_DESKTOP="${DESKTOP_DIR}/${APP_NAME}.desktop"

# ── Build ──────────────────────────────────────────────
echo "🔨 正在构建 ${APP_NAME}..."
cd "${SCRIPT_DIR}"
go build -o "${APP_NAME}" ./...
echo "✅ 构建成功: ${SCRIPT_DIR}/${APP_NAME}"

# ── Install binary ─────────────────────────────────────
mkdir -p "${INSTALL_DIR}"
cp -f "${SCRIPT_DIR}/${APP_NAME}" "${INSTALL_BIN}"
chmod +x "${INSTALL_BIN}"
echo "📦 已安装二进制文件: ${INSTALL_BIN}"

# ── Install icon ───────────────────────────────────────
if [ ! -f "${ICON_SRC}" ]; then
    echo "⚠️  图标文件不存在: ${ICON_SRC}"
    echo "   请将 image.png 放到项目根目录"
    exit 1
fi

mkdir -p "${ICON_DIR}"
cp -f "${ICON_SRC}" "${INSTALL_ICON}"
echo "🎨 已安装图标: ${INSTALL_ICON}"

# ── Generate .desktop file ─────────────────────────────
mkdir -p "${DESKTOP_DIR}"
cat > "${INSTALL_DESKTOP}" << EOF
[Desktop Entry]
Type=Application
Name=${APP_DISPLAY_NAME}
Comment=${APP_COMMENT}
Exec=${INSTALL_BIN}
Icon=${INSTALL_ICON}
Terminal=false
Categories=Utility;Settings;
StartupWMClass=${APP_ID}
StartupNotify=true
EOF
echo "🖥️  已生成 desktop 文件: ${INSTALL_DESKTOP}"

# ── Update desktop database ───────────────────────────
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "${DESKTOP_DIR}" 2>/dev/null || true
fi
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "${HOME}/.local/share/icons/hicolor" 2>/dev/null || true
fi

echo ""
echo "🎉 安装完成！"
echo "   二进制: ${INSTALL_BIN}"
echo "   图标:   ${INSTALL_ICON}"
echo "   桌面:   ${INSTALL_DESKTOP}"
echo ""
echo "   可以从应用启动器找到「${APP_DISPLAY_NAME}」启动"
echo "   或直接运行: ${APP_NAME}"
