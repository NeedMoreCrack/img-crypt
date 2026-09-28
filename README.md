[English](#english) | [繁體中文](#繁體中文)

# ImgCrypt

ImgCrypt is a Go CLI that converts images into transferable Base64 or Base85 text chunks. Encryption is optional. It can process multiple images with one set of settings and restore the original filenames.

ImgCrypt 是 Go 開發的 CLI 工具，可將圖片轉成適合文字傳輸的 Base64 或 Base85 分段。加密可選擇開啟；程式可用同一組設定批次處理圖片，並還原原始檔名。

## English

### Features

- Process one image, multiple specified files, or the images in a directory; the directory scan is not recursive.
- By default, read images beside the executable and write one TXT file per image beside the executable.
- Use one password, JPEG quality, PNG compression mode, text encoding, and chunk length for the whole batch.
- Enter a password for AES-256-GCM encryption with a new random salt and nonce per image. Leave the password blank for **conversion without encryption**; the resulting text is readable by anyone who has it.
- Ask for JPEG quality only if the batch contains JPEG; ask for PNG compression only if it contains PNG.
- Decode a pasted set of chunks or restore one or more TXT files. Chunks from one image can be pasted in any order.
- Check chunk count, duplicate chunks, encoding, and message ID; encrypted files also receive AES-GCM authentication.
- Preserve the original image filename and extension in newly created TXT files. Older ImgCrypt v1/v2 TXT files can still be decoded.

### Image formats

| Format | Processing |
| --- | --- |
| JPG/JPEG | Re-encoded with configurable JPEG quality (default 50; lossy). |
| PNG | Re-encoded with lossless PNG compression (default: Best Compression). |
| WebP and APNG | Original bytes preserved; APNG animation is not re-encoded. |
| Other listed image extensions | Original bytes preserved, without transcoding or size reduction. |

The directory scan recognizes GIF, BMP, TIFF, HEIC/HEIF, AVIF, ICO, ICNS, SVG, JFIF, JPE, JXL, JP2, PSD, RAW, DNG, CR2, NEF, ARW, QOI, EXR, TGA, PBM, PGM, PPM, plus JPG/JPEG, PNG, WebP, and APNG. For these other formats, the program selects files by extension and preserves their bytes; it does **not** validate, decode, or compress every format. Unsupported extensions are skipped during directory scanning. JPEG, PNG, and WebP are identified by their file headers.

> JPEG recompression changes image bytes and can reduce quality. PNG recompression is lossless for the image but may change the file bytes. Base64/Base85 encoding is reversible and does not compress an image.

### Menu

```text
1. Convert images -> TXT files (optional encryption, batch)
2. Decrypt pasted text chunks -> Image
3. Restore TXT files -> Images (batch)
0. Exit
```

### Convert images to TXT

Choose **1**. At `Images / directory [all in executable directory]:`, press Enter to select the recognized images beside the executable. You can also enter a file path, a directory path, or multiple paths separated by `;`. Relative paths are resolved from the **executable's directory**, not necessarily the shell's working directory.

```text
Images / directory [all in executable directory]: D:\Pictures\one.png;D:\Pictures\two.jpg
Password (blank = convert without encryption):
JPEG Quality [default 50]:       # prompted because two.jpg is JPEG
PNG Compression:                 # prompted because one.png is PNG
Encoding [default 2]:            # 1 = Base64; 2 = Base85
Maximum characters per message [default 500]:
```

All images in this batch use the same password and relevant settings. A complete chunk, including its `IMGCRYPT|...|` prefix, stays within the chosen character limit (minimum 80). The default is 500. Encoding to Base64 increases binary size by about one third; Base85 is usually shorter but contains more special characters. If a message service changes Base85 characters, use Base64.

A chunk looks like `IMGCRYPT|B85|263c44efcf9d1cb7|1/88|...`: encoding, message ID, part number, total parts, and payload. Keep all parts for an image together; the message ID helps detect mixed chunks.

Output is saved **beside the executable**, even when source images are in another directory:

| Source | TXT output |
| --- | --- |
| `photo.png` | `photo-imgcrypt.txt` |
| `photo.jpg` with the same stem as an existing TXT | `photo.jpg-imgcrypt.txt` (collision fallback) |

An existing TXT is never overwritten. If both candidate names already exist, that image reports an error and processing continues with the rest. Move or rename an old TXT before converting it again. The TXT contains all chunks for that one image, one per line; it is not one combined TXT for the entire batch.

**Password choice:** A nonempty password produces AES-256-GCM encrypted data (v3). A blank password produces an unencrypted conversion (v4). This mode has no default password and provides **no confidentiality**. A message ID helps detect accidental mixing or alteration, but it is not authentication for passwordless TXT files. Send a password through a separate channel when confidentiality matters.

### Restore images

- **Option 2 — pasted text:** paste the chunks for **one image**, then enter an empty line. Encrypted text prompts for its password; passwordless text restores without a password prompt.
- **Option 3 — TXT files:** press Enter at the TXT path prompt to read all `*-imgcrypt.txt` (and legacy `*.imgcrypt.txt`) files beside the executable. Alternatively enter a TXT file, a directory, or `;`-separated paths. A directory scan is not recursive. Enter one shared password for encrypted files in the batch; leave it blank if the selected files are passwordless. Passwordless files also restore when a password is entered for a mixed batch. Encrypted files fail individually if no password is supplied.

For new TXT files, restored images use their **original filename and extension** and are written beside the executable. An existing image is **not overwritten**: move/rename it first, or restore in a separate directory containing the executable. Legacy v1/v2 TXT files use the filename inferred from the TXT name when available, otherwise `decrypted` plus the stored image extension. Wrong passwords, missing chunks, or corrupted encrypted content cause an error.

### Downloads and execution

Download the matching executable from [Releases](https://github.com/NeedMoreCrack/img-crypt/releases):

| Platform | Release asset | Run |
| --- | --- | --- |
| Windows x64 | `ImgCrypt-windows-amd64.exe` | `./ImgCrypt-windows-amd64.exe` in PowerShell |
| macOS Intel | `ImgCrypt-macos-amd64` | `chmod +x ImgCrypt-macos-amd64 && ./ImgCrypt-macos-amd64` |
| macOS Apple Silicon | `ImgCrypt-macos-arm64` | `chmod +x ImgCrypt-macos-arm64 && ./ImgCrypt-macos-arm64` |
| Linux x64 | `ImgCrypt-linux-amd64` | `chmod +x ImgCrypt-linux-amd64 && ./ImgCrypt-linux-amd64` |
| Linux ARM64 | `ImgCrypt-linux-arm64` | `chmod +x ImgCrypt-linux-arm64 && ./ImgCrypt-linux-arm64` |

**Android / Termux:** `ImgCrypt-linux-arm64` targets Linux ARM64 and is not guaranteed to execute on Android. Build and test an Android/ARM64 executable separately in Termux or for Android before offering it as a Termux release asset. If you build in Termux, keep the executable and the images/TXT files together for convenient default paths. Running with `go run .` creates a temporary executable; to use the directory default, build a binary and run that binary instead.

### Build from source

Go and the dependencies from `go.mod` are required for building. After building, the standalone executable does not require a Go runtime.

```bash
git clone git@github.com:NeedMoreCrack/img-crypt.git
cd img-crypt
go mod download
go build -o ImgCrypt .
```

Cross-compilation examples from bash/zsh:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/ImgCrypt-windows-amd64.exe .
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o dist/ImgCrypt-macos-amd64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o dist/ImgCrypt-macos-arm64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/ImgCrypt-linux-amd64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o dist/ImgCrypt-linux-arm64 .
```

Create `dist` before using these examples (`mkdir -p dist`), or use the repository's existing build script. Test each build on its target system before publishing it. On Windows PowerShell, set `$env:GOOS`, `$env:GOARCH`, and `$env:CGO_ENABLED` instead of using bash's `VAR=value command` syntax.

The repository also includes `build.ps1` and `build.sh`. Compiled `dist/` files are typically ignored by Git and distributed as GitHub Release assets.

### Disclaimer

This project is intended for educational and personal use. Follow the rules of the communication platform you use. Keep an independent copy of important images: the author is not responsible for data loss caused by lost passwords, corrupted data, missing chunks, or incorrect use.

---

## 繁體中文

### 主要功能

- 可處理單一圖片、指定多個檔案，或掃描資料夾內的圖片；資料夾掃描不會遞迴讀取子資料夾。
- 輸入路徑留白時，預設讀取**執行檔所在資料夾**的圖片；每張圖片各輸出一份 TXT，也放在執行檔旁。
- 同一批圖片共用密碼、JPEG 品質、PNG 壓縮模式、文字編碼方式與每段字數設定。
- 有輸入密碼才使用 AES-256-GCM 加密；密碼留白只做圖片處理與文字轉換，**不會加密**。
- 只有選到 JPEG 才詢問 JPEG 品質；只有選到 PNG 才詢問 PNG 壓縮模式。
- 可貼上單張圖片的文字片段還原，也可批次讀取 TXT 還原。
- 檢查缺段、重複段、編碼、Message ID；加密檔另外使用 AES-GCM 驗證。
- 新產生的 TXT 保存原始圖片檔名與副檔名；仍可解讀舊版 ImgCrypt v1/v2 的 TXT。

### 支援圖片格式與壓縮

| 格式 | 處理方式 |
| --- | --- |
| JPG/JPEG | 依指定品質重新編碼，預設 50；屬於有損壓縮。 |
| PNG | 以無損 PNG 壓縮重新編碼，預設「Best Compression」。 |
| WebP、APNG | 保留原始位元組；APNG 動畫不重新編碼。 |
| 其他列出的圖片副檔名 | 保留原始位元組，不轉檔，也不保證能縮小。 |

掃描資料夾時可辨識 GIF、BMP、TIFF、HEIC/HEIF、AVIF、ICO、ICNS、SVG、JFIF、JPE、JXL、JP2、PSD、RAW、DNG、CR2、NEF、ARW、QOI、EXR、TGA、PBM、PGM、PPM，以及 JPG/JPEG、PNG、WebP、APNG。**其他格式只是依副檔名選檔並原樣保存，程式並沒有完整解碼或驗證每種圖片格式。**未列出的副檔名不會被資料夾掃描選入；JPEG、PNG、WebP 則會檢查檔案表頭。

> JPEG 重新編碼會改變內容且可能降低畫質。PNG 重新編碼可維持圖片畫質，但檔案位元組可能不同。Base64／Base85 只是可逆編碼，並不會壓縮圖片。

### 操作選單

```text
1. Convert images -> TXT files (optional encryption, batch)
2. Decrypt pasted text chunks -> Image
3. Restore TXT files -> Images (batch)
0. Exit
```

### 圖片轉 TXT

選 **1**。在 `Images / directory [all in executable directory]:` 直接按 Enter，會選取執行檔旁可辨識的圖片。也可輸入圖片路徑、資料夾路徑，或使用 `;` 分隔多個路徑。**相對路徑以執行檔所在資料夾為基準**，不一定是終端機當下的位置。

```text
Images / directory [all in executable directory]: D:\Pictures\one.png;D:\Pictures\two.jpg
Password (blank = convert without encryption):
JPEG Quality [default 50]:      # 有選到 two.jpg 才顯示
PNG Compression:                # 有選到 one.png 才顯示
Encoding [default 2]:           # 1 = Base64；2 = Base85
Maximum characters per message [default 500]:
```

整批圖片使用同一組密碼及相關格式設定。每段文字連同 `IMGCRYPT|...|` 前綴，長度都不超過指定上限；預設 500 字，最小 80 字。Base64 文字通常會比原始二進位資料多約三分之一；Base85 通常較短，但特殊字元較多。若聊天平台會修改 Base85 的特殊字元，可改選 Base64。

片段格式例如 `IMGCRYPT|B85|263c44efcf9d1cb7|1/88|...`，依序包含編碼、Message ID、目前段數、總段數和資料。請保留同張圖片的所有片段；Message ID 可幫助發現不同圖片的片段被混用。

**TXT 永遠輸出至執行檔所在資料夾**，即使來源圖片位於其他資料夾：

| 來源圖片 | 輸出檔名 |
| --- | --- |
| `photo.png` | `photo-imgcrypt.txt` |
| 同名 TXT 已存在時的 `photo.jpg` | `photo.jpg-imgcrypt.txt`（檔名衝突時備用） |

已存在的 TXT 不會被覆寫。若兩種候選檔名都已存在，該圖片會報錯，程式仍繼續處理其他圖片。再次處理前請先移走或重新命名舊 TXT。**每張圖片各有一份 TXT**，不會把整批圖片合併成同一份 TXT。

**密碼留白：** 不使用任何預設密碼，也不進行加密（v4 格式）。任何取得 TXT 的人都能還原內容。Message ID 可幫助發現意外混段或異動，但不是無密碼檔案的安全驗證。**有輸入密碼：** 每張圖片以新的隨機 salt／nonce 搭配 scrypt 與 AES-256-GCM 加密（v3 格式）。需要保密時請使用強密碼，並從其他管道傳送密碼。

### 還原圖片

- **選項 2，貼上片段：** 一次貼上**一張圖片**的所有 Chunk，最後輸入空白行。順序可打亂；加密檔會再詢問密碼，無密碼檔可直接還原。
- **選項 3，讀取 TXT：** TXT 路徑直接按 Enter，會掃描執行檔旁的 `*-imgcrypt.txt`（也讀取舊版 `*.imgcrypt.txt`）。也可以指定 TXT 檔案、資料夾或用 `;` 分隔多個路徑。資料夾不遞迴掃描。加密檔使用整批共用密碼；若只有無密碼 TXT，可以直接在密碼提示按 Enter。混合處理時，即使輸入了密碼，無密碼 TXT 也可以還原；若留白密碼，加密 TXT 會各自報錯。

新版本 TXT 會還原**原始檔名與副檔名**；還原後的圖片放在執行檔旁。若同名圖片已存在，程式不會覆寫，請先移走或改名，或把執行檔和 TXT 放進沒有同名圖片的資料夾再還原。舊版 v1/v2 TXT 會盡可能由 TXT 檔名推回圖片檔名；無法推得時使用 `decrypted` 加上圖片副檔名。密碼錯誤、缺少片段或加密資料損壞都會報錯。

### 下載與執行

到 [Releases](https://github.com/NeedMoreCrack/img-crypt/releases) 下載對應平台的執行檔：

| 平台 | 檔名 | 執行方式 |
| --- | --- | --- |
| Windows x64 | `ImgCrypt-windows-amd64.exe` | PowerShell：`./ImgCrypt-windows-amd64.exe` |
| macOS Intel | `ImgCrypt-macos-amd64` | `chmod +x ImgCrypt-macos-amd64 && ./ImgCrypt-macos-amd64` |
| macOS Apple Silicon | `ImgCrypt-macos-arm64` | `chmod +x ImgCrypt-macos-arm64 && ./ImgCrypt-macos-arm64` |
| Linux x64 | `ImgCrypt-linux-amd64` | `chmod +x ImgCrypt-linux-amd64 && ./ImgCrypt-linux-amd64` |
| Linux ARM64 | `ImgCrypt-linux-arm64` | `chmod +x ImgCrypt-linux-arm64 && ./ImgCrypt-linux-arm64` |

**Android／Termux：** `ImgCrypt-linux-arm64` 是 Linux ARM64 的執行檔，不能保證直接在 Android 執行。若要提供 Termux 版本，請在 Termux 或針對 Android/ARM64 另外編譯並在手機上測試，再作為獨立的 Release 附件。將執行檔、圖片及 TXT 放在同一資料夾，使用預設路徑會比較方便。`go run .` 會產生暫存執行檔，因此要使用「執行檔旁」作為預設路徑時，請先編譯再執行。

### 從原始碼編譯

編譯時需要 Go 及 `go.mod` 列出的相依套件；編譯完成後的執行檔不需要另裝 Go Runtime。

```bash
git clone git@github.com:NeedMoreCrack/img-crypt.git
cd img-crypt
go mod download
go build -o ImgCrypt .
```

以下是 bash／zsh 跨平台編譯範例：

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/ImgCrypt-windows-amd64.exe .
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o dist/ImgCrypt-macos-amd64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o dist/ImgCrypt-macos-arm64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/ImgCrypt-linux-amd64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o dist/ImgCrypt-linux-arm64 .
```

先執行 `mkdir -p dist`，或使用專案既有的編譯腳本。發布前請在目標平台測試產出的執行檔。若使用 Windows PowerShell，環境變數應設定為 `$env:GOOS`、`$env:GOARCH`、`$env:CGO_ENABLED`；以上 `VAR=value command` 是 bash／zsh 寫法。

專案也包含 `build.ps1` 和 `build.sh`。編譯後的 `dist/` 通常加入 `.gitignore`，透過 GitHub Release 附件提供下載。

### 安全與相容性

- 不要把密碼和加密 TXT 放在同一個傳輸管道；密碼遺失後無法由程式恢復。
- TXT 必須保留所有 Chunk；單一 TXT 對應一張圖片。傳送圖片前請確認聊天平台允許此用途。
- 新產生的加密檔為 v3，無密碼轉換檔為 v4；舊版 v1/v2 仍支援讀取。對於加密檔，錯誤密碼或內容遭修改時 AES-GCM 會拒絕還原。
- `dist/` 一般加入 `.gitignore`；發佈時可將編譯檔上傳到 GitHub Releases。

### 免責聲明

本專案主要供學習與個人使用，請遵守使用的平台規範。重要圖片請另存備份；若因密碼遺失、資料損壞、Chunk 缺失或操作不當而無法還原，作者不負相關責任。

## License

MIT License.
