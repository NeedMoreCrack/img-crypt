[English](#english) | [繁體中文](#繁體中文)

# ImgCrypt

A lightweight cross-platform CLI tool for image compression, encryption, and text-based transmission.

輕量級跨平台 CLI 圖片壓縮、加密與純文字傳輸工具。

---

# English

## Overview

ImgCrypt is a CLI tool written in Go.

It can:

- Compress images
- Encrypt image data with AES-256-GCM
- Convert encrypted binary data into Base64 or Base85 text
- Split the encoded text into multiple chunks
- Reassemble chunks and restore the image
- Read encrypted chunks directly from a TXT file

It is designed for situations where files cannot be transferred directly and only text messages can be sent.

---

## Features

- Cross-platform CLI
- No Go runtime required after compilation
- Windows support
- macOS Intel support
- macOS Apple Silicon support
- Linux support
- Android support through Termux
- JPG / JPEG support
- PNG support
- WebP support
- JPEG quality adjustment
- PNG compression
- AES-256-GCM encryption
- scrypt password-based key derivation
- Base64 encoding
- Base85 encoding
- User-configurable maximum message length
- Automatic chunk splitting
- Automatic chunk sorting and merging
- Missing / duplicate chunk detection
- Message ID verification
- TXT file decryption
- Automatic output image extension restoration

---

## How It Works

Encryption flow:

```text
Image
  ↓
Image Compression
  ↓
AES-256-GCM Encryption
  ↓
Base64 / Base85 Encoding
  ↓
Text Chunk Splitting
  ↓
Transfer as Text
```

Decryption flow:

```text
Text Chunks / TXT File
  ↓
Chunk Validation
  ↓
Chunk Sorting
  ↓
Chunk Merging
  ↓
Base64 / Base85 Decoding
  ↓
AES-256-GCM Decryption
  ↓
Image
```

---

## Supported Formats

### JPEG / JPG

JPEG images can be recompressed with a configurable quality value.

Default:

```text
Quality = 50
```

Example:

```text
Original JPG
958 KB

↓

JPEG Quality 50

↓

Compressed JPG
578 KB
```

Actual compression ratio depends on the image.

> JPEG recompression is lossy.
> The decrypted result is the compressed JPEG, not the original byte-for-byte JPEG file.

---

### PNG

PNG uses lossless compression.

Available modes:

```text
Default
Best Speed
Best Compression
```

PNG preserves image quality, but the compression ratio may be lower than JPEG.

---

### WebP

WebP files are supported.

The current implementation preserves the existing WebP data instead of performing quality-based recompression.

```text
WebP
  ↓
AES-256-GCM
  ↓
Base64 / Base85
```

---

## Encryption

ImgCrypt uses:

```text
AES-256-GCM
```

AES-GCM provides:

```text
Encryption
+
Integrity Verification
```

If the password is incorrect or the encrypted data has been modified, decryption will fail.

---

## Password Key Derivation

ImgCrypt uses `scrypt` to derive a 256-bit AES key from the password.

```text
Password
+
Random Salt
  ↓
scrypt
  ↓
256-bit AES Key
```

A new random salt and nonce are generated for every encryption operation.

---

## Base64 and Base85

### Base64

Advantages:

- High compatibility
- Commonly supported
- Safer on systems that modify special characters

Disadvantage:

- Binary data becomes approximately 33% larger

Example:

```text
600 KB binary
↓
Base64
↓
~800 KB text
```

---

### Base85

Advantages:

- Shorter than Base64
- Better when message length is limited

Example:

```text
600 KB binary
↓
Base85
↓
~750 KB text
```

Base85 is recommended when minimizing character count is more important.

> Some platforms may modify special characters.
> If that happens, use Base64 instead.

---

## Text Chunking

Many messaging platforms impose a maximum character limit per message.

ImgCrypt allows users to define the maximum message length.

Example:

```text
Maximum characters per message [default 500]: 500
```

The program dynamically calculates the available payload size.

The complete chunk, including metadata, will stay within the configured limit.

Example:

```text
IMGCRYPT|B85|263c44efcf9d1cb7|1/88|......
IMGCRYPT|B85|263c44efcf9d1cb7|2/88|......
IMGCRYPT|B85|263c44efcf9d1cb7|3/88|......
```

Each chunk contains:

```text
IMGCRYPT
Encoding Type
Message ID
Chunk Number
Total Chunk Count
Payload
```

---

## Message ID

Each encrypted payload receives a Message ID.

Example:

```text
263c44efcf9d1cb7
```

The Message ID helps prevent chunks from different images from being accidentally mixed.

ImgCrypt checks:

- Missing chunks
- Duplicate chunks
- Mixed Message IDs
- Invalid chunk count
- Invalid encoding
- Corrupted encrypted data

---

## TXT File Support

Chunks can be saved into a TXT file.

Example:

```text
image.jpg.imgcrypt.txt
```

The receiver can decrypt directly from the TXT file.

```text
TXT File
↓
Read Chunks
↓
Validate
↓
Sort
↓
Merge
↓
Decode
↓
Decrypt
↓
Image
```

This is useful when manually pasting dozens of chunks would be inconvenient.

---

## Menu

```text
==============================================
                  ImgCrypt
 Image Compress + AES + Base64/Base85 Chunks
==============================================

Please select options:

1. Encrypt image -> Base64/Base85 text chunks
2. Decrypt pasted text chunks -> Image
3. Decrypt TXT file -> Image
0. Exit
```

---

## Usage

### Encrypt an Image

Select:

```text
1
```

Example:

```text
Image path: D:\Images\example.jpg
Password: myPassword123

Text encoding:
1. Base64
2. Base85

Encoding [default 2]: 2

Maximum characters per message [default 500]: 500

JPEG Quality [default 50]: 50
```

---

### Decrypt Pasted Chunks

Select:

```text
2
```

Paste all chunks:

```text
IMGCRYPT|B85|263c44efcf9d1cb7|1/88|...
IMGCRYPT|B85|263c44efcf9d1cb7|2/88|...
IMGCRYPT|B85|263c44efcf9d1cb7|3/88|...
```

The chunks do not need to be pasted in order.

ImgCrypt will automatically sort and merge them.

---

### Decrypt From TXT

Select:

```text
3
```

Example:

```text
TXT file path:
D:\Images\example.jpg.imgcrypt.txt
```

Enter the correct password and ImgCrypt will restore the image.

---

## Downloads

Precompiled binaries are available from GitHub Releases.

Possible release files:

```text
ImgCrypt-windows-amd64.exe
ImgCrypt-macos-amd64
ImgCrypt-macos-arm64
ImgCrypt-linux-amd64
ImgCrypt-linux-arm64
```

---

## Windows

```powershell
.\ImgCrypt-windows-amd64.exe
```

---

## macOS Apple Silicon

For M1 / M2 / M3 / M4 and newer Apple Silicon Macs:

```bash
chmod +x ImgCrypt-macos-arm64
./ImgCrypt-macos-arm64
```

---

## macOS Intel

```bash
chmod +x ImgCrypt-macos-amd64
./ImgCrypt-macos-amd64
```

---

## Linux

x64:

```bash
chmod +x ImgCrypt-linux-amd64
./ImgCrypt-linux-amd64
```

ARM64:

```bash
chmod +x ImgCrypt-linux-arm64
./ImgCrypt-linux-arm64
```

---

## Android / Termux

Most modern Android devices use ARM64.

```bash
chmod +x ImgCrypt-linux-arm64
./ImgCrypt-linux-arm64
```

---

## Build From Source

Requirements:

```text
Go
```

Clone:

```bash
git clone git@github.com:NeedMoreCrack/img-crypt.git
cd img-crypt
```

Install dependencies:

```bash
go mod tidy
```

Run:

```bash
go run .
```

---

## Build Targets

Windows x64:

```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"

go build -o ImgCrypt-windows-amd64.exe .
```

macOS Apple Silicon:

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
go build -o ImgCrypt-macos-arm64 .
```

macOS Intel:

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
go build -o ImgCrypt-macos-amd64 .
```

Linux x64:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -o ImgCrypt-linux-amd64 .
```

Linux ARM64:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
go build -o ImgCrypt-linux-arm64 .
```

---

## Security Notes

- Use a strong password
- Do not send the password together with the encrypted chunks
- Prefer sending the password through a separate communication channel
- Keep all chunks intact
- Do not manually modify encrypted chunk content
- Lost passwords cannot be recovered by ImgCrypt

---

## Important Notes

JPEG recompression is lossy.

```text
Original JPEG
↓
JPEG Quality 50
↓
Compressed JPEG
```

AES encryption is reversible, but JPEG compression is not.

Therefore:

```text
Decrypt(Encrypt(Compressed JPG))
=
Compressed JPG
```

but not necessarily:

```text
Compressed JPG
=
Original JPG
```

byte-for-byte.

---

# 繁體中文

## 專案介紹

ImgCrypt 是一款使用 Go 開發的輕量級跨平台 CLI 工具。

主要功能包含：

- 圖片壓縮
- AES-256-GCM 加密
- 將加密後的二進位資料轉成 Base64 或 Base85 文字
- 自動將長字串切割成多段
- 接收端自動排序、合併與還原
- 支援從 TXT 檔案直接讀取並解密

主要用途是：

> 當環境無法直接傳送圖片或檔案，只允許傳送純文字時，可以將圖片轉換成加密文字後進行傳輸。

---

## 功能

- 跨平台 CLI
- 編譯後不需要安裝 Go Runtime
- 支援 Windows
- 支援 macOS Intel
- 支援 macOS Apple Silicon
- 支援 Linux
- 支援 Android Termux
- 支援 JPG / JPEG
- 支援 PNG
- 支援 WebP
- JPEG Quality 可調整
- PNG 壓縮
- AES-256-GCM 加密
- 使用 scrypt 將密碼轉換成 AES Key
- Base64 編碼
- Base85 編碼
- 使用者可自行設定每段最大字數
- 自動切割文字
- 自動排序與合併 Chunk
- 偵測缺少 Chunk
- 偵測重複 Chunk
- Message ID 驗證
- 支援從 TXT 解密
- 自動還原圖片副檔名

---

## 工作流程

加密：

```text
圖片
  ↓
圖片壓縮
  ↓
AES-256-GCM 加密
  ↓
Base64 / Base85 編碼
  ↓
文字自動分段
  ↓
以純文字方式傳輸
```

解密：

```text
文字片段 / TXT
  ↓
驗證 Chunk
  ↓
自動排序
  ↓
合併
  ↓
Base64 / Base85 Decode
  ↓
AES-256-GCM 解密
  ↓
圖片
```

---

## 支援圖片格式

### JPG / JPEG

JPEG 可以重新指定品質進行壓縮。

預設：

```text
Quality = 50
```

例如：

```text
原始 JPG
958 KB

↓

JPEG Quality 50

↓

壓縮後 JPG
578 KB
```

實際壓縮率會依圖片內容不同而有所差異。

> JPEG 屬於有損壓縮。
> 解密後得到的是「壓縮後的 JPEG」，並不是最初原始 JPEG 的 byte-for-byte 完整還原。

---

### PNG

PNG 使用無損壓縮。

支援：

```text
Default
Best Speed
Best Compression
```

PNG 可以保留畫質，但檔案縮小幅度通常不會像 JPEG 那麼明顯。

---

### WebP

支援 WebP。

目前版本會保留既有 WebP 的內容，不會重新用 Quality 進行有損壓縮。

```text
WebP
↓
AES-256-GCM
↓
Base64 / Base85
```

---

## 加密方式

ImgCrypt 使用：

```text
AES-256-GCM
```

AES-GCM 同時提供：

```text
資料加密
+
完整性驗證
```

如果：

- 密碼錯誤
- 加密資料被修改
- 資料損壞

解密會失敗。

---

## 密碼轉換

ImgCrypt 不會直接把使用者輸入的密碼當成 AES Key。

而是使用：

```text
scrypt
```

流程：

```text
Password
+
Random Salt
↓
scrypt
↓
256-bit AES Key
```

每次加密都會產生新的：

```text
Salt
Nonce
```

---

## Base64 / Base85

### Base64

優點：

- 相容性高
- 幾乎所有系統都支援
- 特殊字元較少

缺點：

- 大約會比原始 binary 增加 33%

例如：

```text
600 KB
↓
Base64
↓
約 800 KB 文字
```

---

### Base85

優點：

- 比 Base64 短
- 適合有字數限制的平台

例如：

```text
600 KB
↓
Base85
↓
約 750 KB 文字
```

如果你的主要需求是減少字數，建議優先使用 Base85。

但 Base85 會包含較多特殊符號。

如果聊天平台會修改特殊符號，建議改用 Base64。

---

## 自動分段

許多聊天平台都有單則訊息字數限制。

ImgCrypt 可以讓使用者自行設定：

```text
Maximum characters per message [default 500]: 500
```

程式會自行計算：

```text
Metadata
+
Payload
<=
使用者指定的最大字數
```

例如：

```text
IMGCRYPT|B85|263c44efcf9d1cb7|1/88|......
IMGCRYPT|B85|263c44efcf9d1cb7|2/88|......
IMGCRYPT|B85|263c44efcf9d1cb7|3/88|......
```

每一段包含：

```text
IMGCRYPT
編碼方式
Message ID
目前段數
總段數
Payload
```

---

## Message ID

每份加密資料都會產生 Message ID。

例如：

```text
263c44efcf9d1cb7
```

用途是避免不同圖片的 Chunk 被混在一起。

ImgCrypt 會檢查：

- 缺少 Chunk
- 重複 Chunk
- 不同 Message ID
- Chunk 數量異常
- 編碼方式錯誤
- 加密資料損壞

---

## TXT 解密

加密後的 Chunk 可以存成 TXT。

例如：

```text
image.jpg.imgcrypt.txt
```

接收端可以直接：

```text
TXT
↓
讀取每一行
↓
驗證
↓
排序
↓
合併
↓
Decode
↓
AES 解密
↓
圖片
```

如果圖片被切成數十段甚至數百段，使用 TXT 會比手動貼到 CLI 更方便。

---

## 選單

```text
==============================================
                  ImgCrypt
 Image Compress + AES + Base64/Base85 Chunks
==============================================

Please select options:

1. Encrypt image -> Base64/Base85 text chunks
2. Decrypt pasted text chunks -> Image
3. Decrypt TXT file -> Image
0. Exit
```

---

## 使用方式

### 加密圖片

選：

```text
1
```

例如：

```text
Image path: D:\Images\example.jpg
Password: myPassword123

Text encoding:
1. Base64
2. Base85

Encoding [default 2]: 2

Maximum characters per message [default 500]: 500

JPEG Quality [default 50]: 50
```

---

### 貼上 Chunk 解密

選：

```text
2
```

貼入：

```text
IMGCRYPT|B85|263c44efcf9d1cb7|1/88|...
IMGCRYPT|B85|263c44efcf9d1cb7|2/88|...
IMGCRYPT|B85|263c44efcf9d1cb7|3/88|...
```

Chunk 不需要按照順序。

ImgCrypt 會自動排序。

全部貼完後輸入空白行即可。

---

### 從 TXT 解密

選：

```text
3
```

例如：

```text
TXT file path:
D:\Images\example.jpg.imgcrypt.txt
```

輸入正確密碼後即可還原圖片。

---

## 下載

可以從 GitHub Releases 下載已經編譯好的版本。

例如：

```text
ImgCrypt-windows-amd64.exe
ImgCrypt-macos-amd64
ImgCrypt-macos-arm64
ImgCrypt-linux-amd64
ImgCrypt-linux-arm64
```

---

## Windows

```powershell
.\ImgCrypt-windows-amd64.exe
```

---

## macOS Apple Silicon

適用：

```text
M1
M2
M3
M4
...
```

執行：

```bash
chmod +x ImgCrypt-macos-arm64
./ImgCrypt-macos-arm64
```

---

## macOS Intel

```bash
chmod +x ImgCrypt-macos-amd64
./ImgCrypt-macos-amd64
```

---

## Linux

x64：

```bash
chmod +x ImgCrypt-linux-amd64
./ImgCrypt-linux-amd64
```

ARM64：

```bash
chmod +x ImgCrypt-linux-arm64
./ImgCrypt-linux-arm64
```

---

## Android / Termux

目前多數 Android 手機為 ARM64。

可以使用：

```bash
chmod +x ImgCrypt-linux-arm64
./ImgCrypt-linux-arm64
```

---

## 從原始碼執行

需要：

```text
Go
```

Clone：

```bash
git clone git@github.com:NeedMoreCrack/img-crypt.git
cd img-crypt
```

下載 dependency：

```bash
go mod tidy
```

執行：

```bash
go run .
```

---

## 編譯

Windows x64：

```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"

go build -o ImgCrypt-windows-amd64.exe .
```

macOS Apple Silicon：

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
go build -o ImgCrypt-macos-arm64 .
```

macOS Intel：

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
go build -o ImgCrypt-macos-amd64 .
```

Linux x64：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -o ImgCrypt-linux-amd64 .
```

Linux ARM64：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
go build -o ImgCrypt-linux-arm64 .
```

---

## 安全注意事項

建議：

- 使用強密碼
- 不要把密碼和加密資料放在同一則訊息
- 密碼最好透過不同管道傳送
- 不要修改 Chunk 內容
- 確保所有 Chunk 都有完整收到
- 密碼遺失後無法由 ImgCrypt 還原

---

## JPEG 注意事項

JPEG 屬於有損壓縮。

```text
原始 JPEG
↓
Quality 50
↓
壓縮 JPEG
```

AES 加密本身可以完整還原。

但是 JPEG 有損壓縮無法還原被捨棄的圖片資訊。

因此：

```text
Decrypt(Encrypt(Compressed JPG))
=
Compressed JPG
```

但：

```text
Compressed JPG
!=
Original JPG
```

不一定 byte-for-byte 相同。

---

## 專案結構

```text
img-crypt/
├── .gitignore
├── README.md
├── build.ps1
├── build.sh
├── go.mod
├── go.sum
├── main.go
└── dist/
```

其中：

```text
dist/
```

通常會加入 `.gitignore`，避免編譯產物直接 commit 進 Git。

---

## Disclaimer / 免責聲明

This project is intended for educational and personal use.

Please follow the rules and terms of the communication platform you use.

The author is not responsible for data loss caused by lost passwords, corrupted data, missing chunks, or improper usage.

本專案主要用途為學習與個人使用。

請遵守所使用平台的規範與服務條款。

若因密碼遺失、資料損壞、Chunk 缺失或操作錯誤造成資料無法還原，作者不負相關責任。

---

## License

MIT License
