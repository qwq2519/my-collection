@echo off
REM 下载 ffmpeg 到项目本地 persist\bin\ 目录
REM 用法: scripts\install-ffmpeg.bat

setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."
set "BIN_DIR=%PROJECT_ROOT%\persist\bin"
set "TMP_ZIP=%TEMP%\ffmpeg-download.zip"
set "TMP_EXTRACT=%TEMP%\ffmpeg-extract"

echo ==^> 正在下载 ffmpeg ...

if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

curl -fSL "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip" -o "%TMP_ZIP%"
if errorlevel 1 (
    echo 错误: 下载失败，请检查网络连接
    exit /b 1
)

echo ==^> 正在解压 ...
if exist "%TMP_EXTRACT%" rd /s /q "%TMP_EXTRACT%"
tar -xf "%TMP_ZIP%" -C "%TEMP%" 2>nul
if errorlevel 1 (
    powershell -Command "Expand-Archive -Path '%TMP_ZIP%' -DestinationPath '%TMP_EXTRACT%' -Force"
)

REM 查找解压后的 ffmpeg.exe 并复制到目标目录
for /r "%TEMP%\ffmpeg-master-latest-win64-gpl" %%f in (ffmpeg.exe) do (
    copy /y "%%f" "%BIN_DIR%\ffmpeg.exe" >nul 2>&1
    goto :verify
)
for /r "%TMP_EXTRACT%" %%f in (ffmpeg.exe) do (
    copy /y "%%f" "%BIN_DIR%\ffmpeg.exe" >nul 2>&1
    goto :verify
)

echo 错误: 解压后未找到 ffmpeg.exe
exit /b 1

:verify
del /f /q "%TMP_ZIP%" 2>nul
rd /s /q "%TEMP%\ffmpeg-master-latest-win64-gpl" 2>nul
rd /s /q "%TMP_EXTRACT%" 2>nul

"%BIN_DIR%\ffmpeg.exe" -version >nul 2>&1
if errorlevel 1 (
    echo 错误: 安装后验证失败
    exit /b 1
)

for /f "tokens=3" %%v in ('"%BIN_DIR%\ffmpeg.exe" -version ^| findstr /b "ffmpeg version"') do set "VERSION=%%v"
echo ==^> 安装成功! ffmpeg %VERSION%
echo     路径: %BIN_DIR%\ffmpeg.exe
echo     请在应用设置页点击「重新检测」

endlocal
