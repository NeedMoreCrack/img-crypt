package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/ascii85"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	magicHeader = "IMGCRYPT"
	versionV1   = byte(1)
	versionV2   = byte(2)
	versionV3   = byte(3)
	versionV4   = byte(4) // Text conversion only; no encryption.

	defaultJPEGQuality     = 50
	defaultMaxMessageChars = 500

	saltSize = 16
	keySize  = 32

	scryptN = 32768
	scryptR = 8
	scryptP = 1

	formatJPEG  = byte(1)
	formatPNG   = byte(2)
	formatWebP  = byte(3)
	formatOther = byte(4)
)

var reader = bufio.NewReader(os.Stdin)

type ImageInfo struct {
	FormatCode   byte
	FormatName   string
	Extension    string
	Setting      byte
	OriginalName string
}

type Chunk struct {
	Encoding  string
	MessageID string
	Index     int
	Total     int
	Payload   string
}

func main() {
	fmt.Println("==============================================")
	fmt.Println("                  ImgCrypt")
	fmt.Println(" Image Compress + AES + Base64/Base85 Chunks")
	fmt.Println("==============================================")
	fmt.Println()

	for {
		showMenu()
		option := readLine("Option: ")
		fmt.Println()

		var err error

		switch option {
		case "1":
			err = encryptImageToChunks()
		case "2":
			err = decryptPastedChunksToImage()
		case "3":
			err = decryptChunksFromTXT()
		case "0":
			fmt.Println("Bye.")
			return
		default:
			fmt.Println("[ERROR] Invalid option.")
			continue
		}

		if err != nil {
			fmt.Println("[ERROR]", err)
		}

		fmt.Println()
		fmt.Println("----------------------------------------------")
		fmt.Println()
	}
}

func showMenu() {
	fmt.Println("Please select options:")
	fmt.Println("1. Convert images -> TXT files (optional encryption, batch)")
	fmt.Println("2. Decrypt pasted text chunks -> Image")
	fmt.Println("3. Restore TXT files -> Images (batch)")
	fmt.Println("0. Exit")
	fmt.Println()
}

// =========================================================
// Encrypt flow
// =========================================================

func encryptImageToChunks() error {
	fmt.Println("=== Encrypt Images ===")
	baseDir, err := executableDir()
	if err != nil {
		return err
	}
	fmt.Println("Executable directory:", baseDir)
	fmt.Println("Enter a file or directory path (multiple paths: separate with ;).")
	fmt.Println("Leave blank to process all image files in the executable directory.")
	paths, err := collectInputs(cleanPath(readLine("Images / directory [all in executable directory]: ")), baseDir, false)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return errors.New("no image files found")
	}
	// Inspect the selected files before asking for format-specific settings.
	imageInfoByPath := make(map[string]ImageInfo, len(paths))
	hasJPEG, hasPNG := false, false
	for _, path := range paths {
		info, err := detectImageFormat(path)
		if err != nil {
			fmt.Printf("[ERROR] %s: %v\n", path, err)
			continue
		}
		imageInfoByPath[path] = info
		if info.FormatCode == formatJPEG {
			hasJPEG = true
		}
		if info.FormatCode == formatPNG {
			hasPNG = true
		}
	}
	if len(imageInfoByPath) == 0 {
		return errors.New("no readable image files found")
	}
	fmt.Printf("[INFO] Selected %d images\n", len(imageInfoByPath))
	password := readLine("Password (blank = convert without encryption): ")
	if password == "" {
		fmt.Println("[INFO] No password: TXT content will NOT be encrypted.")
	}
	quality := defaultJPEGQuality
	if hasJPEG {
		quality = readJPEGQuality()
	}
	pngMode := byte(3)
	if hasPNG {
		pngMode = readPNGCompressionMode()
	}
	encodingType := readEncodingType()
	maxChars := readMaxMessageChars()
	fmt.Println("[INFO] Output directory:", baseDir)
	succeeded := 0
	for _, inputPath := range paths {
		info, ok := imageInfoByPath[inputPath]
		if !ok {
			continue
		}
		if err := encryptOne(inputPath, info, baseDir, password, quality, pngMode, encodingType, maxChars); err != nil {
			fmt.Printf("[ERROR] %s: %v\n", inputPath, err)
		} else {
			succeeded++
		}
	}
	fmt.Printf("[INFO] Processed %d/%d readable images\n", succeeded, len(imageInfoByPath))
	if succeeded == 0 {
		return errors.New("no images processed")
	}
	return nil
}

func encryptOne(inputPath string, info ImageInfo, baseDir, password string, quality int, pngMode byte, encodingType string, maxChars int) error {
	info.OriginalName = filepath.Base(inputPath)
	data, originalSize, processedInfo, err := processImageForEncryption(inputPath, info, quality, pngMode)
	if err != nil {
		return err
	}
	var encrypted []byte
	if password == "" {
		encrypted, err = convertDataV4(data, processedInfo)
	} else {
		encrypted, err = encryptDataV3(data, password, processedInfo)
	}
	if err != nil {
		return err
	}
	encoded, err := encodeText(encrypted, encodingType)
	if err != nil {
		return err
	}
	chunks, err := createChunksWithMaxLength(encoded, encodingType, generateMessageID(encrypted), maxChars)
	if err != nil {
		return err
	}
	stem := strings.TrimSuffix(info.OriginalName, filepath.Ext(info.OriginalName))
	outputPath := filepath.Join(baseDir, stem+"-imgcrypt.txt")
	// Two images with the same stem but different extensions must not overwrite each other.
	if fileExists(outputPath) {
		outputPath = filepath.Join(baseDir, info.OriginalName+"-imgcrypt.txt")
	}
	if fileExists(outputPath) {
		return fmt.Errorf("output already exists: %s", outputPath)
	}
	if err := writeNewFile(outputPath, []byte(strings.Join(chunks, "\n")+"\n")); err != nil {
		return err
	}
	mode := "encrypted"
	if password == "" {
		mode = "converted, not encrypted"
	}
	fmt.Printf("[OK] %s (%s -> %s, %s) -> %s (%d chunks, %s)\n", info.OriginalName, formatBytes(originalSize), formatBytes(int64(len(data))), processedInfo.FormatName, outputPath, len(chunks), mode)
	return nil
}

// =========================================================
// Decrypt pasted chunks
// =========================================================

func decryptPastedChunksToImage() error {
	fmt.Println("=== Decrypt Pasted Text Chunks ===")
	fmt.Println()
	fmt.Println("Paste all ImgCrypt chunks.")
	fmt.Println("Order does NOT matter.")
	fmt.Println("After pasting all chunks, enter an empty line.")
	fmt.Println()

	lines, err := readMultipleLines()
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		return errors.New("no ImgCrypt chunks provided")
	}

	baseDir, err := executableDir()
	if err != nil {
		return err
	}
	return decryptChunkLines(lines, "", "", baseDir)
}

// =========================================================
// Decrypt TXT file
// =========================================================

func decryptChunksFromTXT() error {
	baseDir, err := executableDir()
	if err != nil {
		return err
	}
	fmt.Println("TXT input: file, directory, or paths separated by ; (blank: all *-imgcrypt.txt beside executable)")
	paths, err := collectInputs(cleanPath(readLine("TXT files [all beside executable]: ")), baseDir, true)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return errors.New("no ImgCrypt TXT files found")
	}
	password := readLine("Password for encrypted files (blank = passwordless files only): ")
	succeeded := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			err = decryptChunkLines(splitLines(string(data)), path, password, baseDir)
		}
		if err != nil {
			fmt.Printf("[ERROR] %s: %v\n", path, err)
		} else {
			succeeded++
		}
	}
	fmt.Printf("[INFO] Decrypted %d/%d files\n", succeeded, len(paths))
	if succeeded == 0 {
		return errors.New("no files decrypted")
	}
	return nil
}

func splitLines(content string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func decryptChunkLines(lines []string, sourcePath, password, baseDir string) error {
	encodingType, messageID, encodedData, err := parseAndMergeChunks(lines)
	if err != nil {
		return err
	}
	encryptedData, err := decodeText(encodedData, encodingType)
	if err != nil {
		return err
	}
	if generateMessageID(encryptedData) != messageID {
		return errors.New("message ID verification failed")
	}
	if len(encryptedData) <= len(magicHeader) || string(encryptedData[:len(magicHeader)]) != magicHeader {
		return errors.New("invalid ImgCrypt header")
	}
	if encryptedData[len(magicHeader)] != versionV4 && password == "" {
		if sourcePath == "" {
			password = readLine("Password: ")
		}
		if password == "" {
			return errors.New("this TXT is encrypted; a password is required")
		}
	}
	decryptedData, info, err := decryptData(encryptedData, password)
	if err != nil {
		return err
	}
	name := info.OriginalName
	if name == "" {
		name = legacyOutputName(sourcePath, info.Extension)
	}
	outputPath := filepath.Join(baseDir, name)
	if fileExists(outputPath) {
		return fmt.Errorf("output already exists (move or rename it first): %s", outputPath)
	}
	if err := writeNewFile(outputPath, decryptedData); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}
	fmt.Printf("[OK] Restored %s (%s, %s)\n", outputPath, info.FormatName, formatBytes(int64(len(decryptedData))))
	return nil
}

func legacyOutputName(sourcePath, ext string) string {
	if sourcePath == "" {
		return "decrypted" + ext
	}
	name := filepath.Base(sourcePath)
	if strings.HasSuffix(strings.ToLower(name), ".imgcrypt.txt") {
		name = name[:len(name)-len(".imgcrypt.txt")]
	} else if strings.HasSuffix(strings.ToLower(name), "-imgcrypt.txt") {
		name = name[:len(name)-len("-imgcrypt.txt")]
	} else {
		return "decrypted" + ext
	}
	if filepath.Ext(name) == "" {
		name += ext
	}
	return name
}

// =========================================================
// Image format / compression
// =========================================================

func detectImageFormat(path string) (ImageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return ImageInfo{}, err
	}
	defer file.Close()
	head := make([]byte, 32)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return ImageInfo{}, err
	}
	head = head[:n]
	if n == 0 {
		return ImageInfo{}, errors.New("empty image")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".apng" {
		return ImageInfo{FormatCode: formatOther, FormatName: "APNG (original bytes)", Extension: ext}, nil
	}
	switch {
	case len(head) >= 3 && head[0] == 0xff && head[1] == 0xd8 && head[2] == 0xff:
		if ext != ".jpeg" {
			ext = ".jpg"
		}
		return ImageInfo{FormatCode: formatJPEG, FormatName: "JPEG", Extension: ext}, nil
	case len(head) >= 8 && bytes.Equal(head[:8], []byte{0x89, 'P', 'N', 'G', 13, 10, 26, 10}):
		return ImageInfo{FormatCode: formatPNG, FormatName: "PNG", Extension: ".png"}, nil
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return ImageInfo{FormatCode: formatWebP, FormatName: "WebP", Extension: ".webp"}, nil
	default:
		// Other image formats are preserved byte for byte; no decoder is required.
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			return ImageInfo{}, errors.New("image content does not match its extension")
		}
		if !isImageExtension(ext) {
			return ImageInfo{}, fmt.Errorf("unrecognized image extension %q", ext)
		}
		return ImageInfo{FormatCode: formatOther, FormatName: strings.ToUpper(strings.TrimPrefix(ext, ".")) + " (original bytes)", Extension: ext}, nil
	}
}

var imageExtensions = map[string]bool{
	".gif": true, ".bmp": true, ".tif": true, ".tiff": true, ".heic": true, ".heif": true,
	".avif": true, ".ico": true, ".icns": true, ".svg": true, ".apng": true, ".jfif": true,
	".jpe": true, ".jxl": true, ".jp2": true, ".psd": true, ".raw": true, ".dng": true,
	".cr2": true, ".nef": true, ".arw": true, ".qoi": true, ".exr": true, ".tga": true,
	".pbm": true, ".pgm": true, ".ppm": true, ".webp": true, ".jpg": true, ".jpeg": true, ".png": true,
}

func isImageExtension(ext string) bool { return imageExtensions[strings.ToLower(ext)] }

func processImageForEncryption(inputPath string, info ImageInfo, quality int, mode byte) ([]byte, int64, ImageInfo, error) {
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, 0, ImageInfo{}, err
	}
	var data []byte
	switch info.FormatCode {
	case formatJPEG:
		data, err = compressJPEG(inputPath, quality)
		info.Setting = byte(quality)
	case formatPNG:
		data, err = compressPNG(inputPath, mode)
		info.Setting = mode
	default:
		data, err = os.ReadFile(inputPath)
	}
	if err != nil {
		return nil, 0, ImageInfo{}, err
	}
	return data, fileInfo.Size(), info, nil
}

func compressJPEG(inputPath string, quality int) ([]byte, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JPEG: %w", err)
	}
	defer inputFile.Close()

	img, err := jpeg.Decode(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}

	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buffer.Bytes(), nil
}

func compressPNG(inputPath string, mode byte) ([]byte, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PNG: %w", err)
	}
	defer inputFile.Close()

	img, err := png.Decode(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	switch mode {
	case 1:
		encoder.CompressionLevel = png.DefaultCompression
	case 2:
		encoder.CompressionLevel = png.BestSpeed
	case 3:
		encoder.CompressionLevel = png.BestCompression
	default:
		return nil, errors.New("invalid PNG compression mode")
	}

	var buffer bytes.Buffer
	if err := encoder.Encode(&buffer, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buffer.Bytes(), nil
}

// =========================================================
// AES encryption / decryption
// =========================================================

func namedPayload(data []byte, name string) ([]byte, error) {
	if !validImageName(name) || len([]byte(name)) > 65535 {
		return nil, errors.New("invalid image filename")
	}
	nameBytes := []byte(name)
	result := make([]byte, 2+len(nameBytes)+len(data))
	binary.BigEndian.PutUint16(result[:2], uint16(len(nameBytes)))
	copy(result[2:], nameBytes)
	copy(result[2+len(nameBytes):], data)
	return result, nil
}

// V4 contains processed image bytes and the filename as plain data.
// It must never be mistaken for an encrypted V3 file.
func convertDataV4(data []byte, info ImageInfo) ([]byte, error) {
	payload, err := namedPayload(data, info.OriginalName)
	if err != nil {
		return nil, err
	}
	result := append([]byte(magicHeader), versionV4, info.FormatCode, info.Setting)
	return append(result, payload...), nil
}

func encryptDataV3(data []byte, password string, info ImageInfo) ([]byte, error) {
	plaintext, err := namedPayload(data, info.OriginalName)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	result := append([]byte(magicHeader), versionV3, info.FormatCode, info.Setting)
	result = append(result, salt...)
	result = append(result, byte(len(nonce)))
	result = append(result, nonce...)
	return append(result, gcm.Seal(nil, nonce, plaintext, nil)...), nil
}

func decryptData(data []byte, password string) ([]byte, ImageInfo, error) {
	if len(data) < len(magicHeader)+1 {
		return nil, ImageInfo{}, errors.New("invalid ImgCrypt data")
	}

	offset := 0
	header := string(data[offset : offset+len(magicHeader)])
	if header != magicHeader {
		return nil, ImageInfo{}, errors.New("invalid ImgCrypt header")
	}
	offset += len(magicHeader)

	fileVersion := data[offset]
	offset++
	if fileVersion == versionV4 {
		if len(data) < offset+2 {
			return nil, ImageInfo{}, errors.New("invalid passwordless ImgCrypt data")
		}
		info, err := imageInfoFromCode(data[offset], data[offset+1])
		if err != nil {
			return nil, ImageInfo{}, err
		}
		image, name, err := unpackNamedPayload(data[offset+2:])
		if err != nil {
			return nil, ImageInfo{}, err
		}
		info.OriginalName = name
		info.Extension = filepath.Ext(name)
		return image, info, nil
	}
	minimumV1 := len(magicHeader) + 1 + 1 + saltSize + 1
	if len(data) < minimumV1 {
		return nil, ImageInfo{}, errors.New("invalid ImgCrypt data")
	}

	var info ImageInfo

	switch fileVersion {
	case versionV1:
		// Backward compatibility with the earlier JPEG-only format.
		if len(data) < offset+1+saltSize+1 {
			return nil, ImageInfo{}, errors.New("invalid ImgCrypt v1 data")
		}
		info = ImageInfo{FormatCode: formatJPEG, FormatName: "JPEG", Extension: ".jpg", Setting: data[offset]}
		offset++

	case versionV2, versionV3:
		if len(data) < offset+2+saltSize+1 {
			return nil, ImageInfo{}, errors.New("invalid ImgCrypt v2 data")
		}
		formatCode := data[offset]
		offset++
		setting := data[offset]
		offset++

		var err error
		info, err = imageInfoFromCode(formatCode, setting)
		if err != nil {
			return nil, ImageInfo{}, err
		}

	default:
		return nil, ImageInfo{}, fmt.Errorf("unsupported ImgCrypt version: %d", fileVersion)
	}

	if len(data) < offset+saltSize+1 {
		return nil, ImageInfo{}, errors.New("invalid ImgCrypt data")
	}

	salt := data[offset : offset+saltSize]
	offset += saltSize

	nonceSize := int(data[offset])
	offset++
	if nonceSize <= 0 || len(data) < offset+nonceSize {
		return nil, ImageInfo{}, errors.New("invalid nonce")
	}

	nonce := data[offset : offset+nonceSize]
	offset += nonceSize
	ciphertext := data[offset:]
	if len(ciphertext) == 0 {
		return nil, ImageInfo{}, errors.New("missing ciphertext")
	}

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, ImageInfo{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ImageInfo{}, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ImageInfo{}, fmt.Errorf("failed to create GCM: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, ImageInfo{}, errors.New("invalid GCM nonce size")
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ImageInfo{}, errors.New("decryption failed: wrong password or corrupted data")
	}

	if fileVersion == versionV3 {
		var name string
		plaintext, name, err = unpackNamedPayload(plaintext)
		if err != nil {
			return nil, ImageInfo{}, err
		}
		info.OriginalName = name
		info.Extension = filepath.Ext(name)
	}
	return plaintext, info, nil
}

func unpackNamedPayload(payload []byte) ([]byte, string, error) {
	if len(payload) < 2 {
		return nil, "", errors.New("invalid filename metadata")
	}
	nameSize := int(binary.BigEndian.Uint16(payload[:2]))
	if nameSize == 0 || len(payload) < 2+nameSize {
		return nil, "", errors.New("invalid filename metadata")
	}
	name := string(payload[2 : 2+nameSize])
	if !validImageName(name) {
		return nil, "", errors.New("unsafe filename in data")
	}
	return payload[2+nameSize:], name, nil
}

func deriveKey(password string, salt []byte) ([]byte, error) {
	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}
	return key, nil
}

func imageInfoFromCode(formatCode byte, setting byte) (ImageInfo, error) {
	switch formatCode {
	case formatJPEG:
		return ImageInfo{FormatCode: formatJPEG, FormatName: "JPEG", Extension: ".jpg", Setting: setting}, nil
	case formatPNG:
		return ImageInfo{FormatCode: formatPNG, FormatName: "PNG", Extension: ".png", Setting: setting}, nil
	case formatWebP:
		return ImageInfo{FormatCode: formatWebP, FormatName: "WebP", Extension: ".webp", Setting: setting}, nil
	case formatOther:
		return ImageInfo{FormatCode: formatOther, FormatName: "Image (original bytes)", Setting: setting}, nil
	default:
		return ImageInfo{}, fmt.Errorf("unsupported stored image format: %d", formatCode)
	}
}

// =========================================================
// Base64 / Base85
// =========================================================

func encodeText(data []byte, encodingType string) (string, error) {
	switch encodingType {
	case "B64":
		return base64.StdEncoding.EncodeToString(data), nil
	case "B85":
		dst := make([]byte, ascii85.MaxEncodedLen(len(data)))
		n := ascii85.Encode(dst, data)
		return string(dst[:n]), nil
	default:
		return "", fmt.Errorf("unsupported encoding type: %s", encodingType)
	}
}

func decodeText(input string, encodingType string) ([]byte, error) {
	switch encodingType {
	case "B64":
		data, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return nil, fmt.Errorf("invalid Base64 data: %w", err)
		}
		return data, nil

	case "B85":
		src := []byte(input)
		dst := make([]byte, (len(src)*4)/5+8)
		n, _, err := ascii85.Decode(dst, src, true)
		if err != nil {
			return nil, fmt.Errorf("invalid Base85 data: %w", err)
		}
		return dst[:n], nil

	default:
		return nil, fmt.Errorf("unsupported encoding type: %s", encodingType)
	}
}

// =========================================================
// Dynamic chunking: each complete message <= user limit
// =========================================================

func createChunksWithMaxLength(encodedData string, encodingType string, messageID string, maxChars int) ([]string, error) {
	if maxChars < 80 {
		return nil, errors.New("maximum message length is too small; use at least 80 characters")
	}
	if encodedData == "" {
		return nil, errors.New("encoded data is empty")
	}

	// The total chunk count is part of the prefix, so its digit count affects
	// how much payload each chunk can hold. Iterate until the total stabilizes.
	totalGuess := 1

	for attempt := 0; attempt < 20; attempt++ {
		chunks, err := splitWithKnownTotal(encodedData, encodingType, messageID, maxChars, totalGuess)
		if err != nil {
			return nil, err
		}

		actualTotal := len(chunks)
		if actualTotal == totalGuess {
			return chunks, nil
		}
		totalGuess = actualTotal
	}

	return nil, errors.New("failed to stabilize chunk count")
}

func splitWithKnownTotal(encodedData string, encodingType string, messageID string, maxChars int, total int) ([]string, error) {
	chunks := make([]string, 0)
	start := 0
	index := 1

	for start < len(encodedData) {
		prefix := fmt.Sprintf("IMGCRYPT|%s|%s|%d/%d|", encodingType, messageID, index, total)
		available := maxChars - len(prefix)
		if available <= 0 {
			return nil, fmt.Errorf("maximum message length %d is too small for metadata prefix", maxChars)
		}

		end := start + available
		if end > len(encodedData) {
			end = len(encodedData)
		}

		chunk := prefix + encodedData[start:end]
		if len(chunk) > maxChars {
			return nil, errors.New("internal chunk size calculation error")
		}

		chunks = append(chunks, chunk)
		start = end
		index++
	}

	return chunks, nil
}

// =========================================================
// Parse / merge chunks
// =========================================================

func parseAndMergeChunks(lines []string) (string, string, string, error) {
	chunks := make([]Chunk, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		chunk, err := parseChunk(line)
		if err != nil {
			return "", "", "", err
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return "", "", "", errors.New("no valid ImgCrypt chunks")
	}

	expectedEncoding := chunks[0].Encoding
	expectedID := chunks[0].MessageID
	expectedTotal := chunks[0].Total

	seen := make(map[int]bool, expectedTotal)
	for _, chunk := range chunks {
		if chunk.Encoding != expectedEncoding {
			return "", "", "", errors.New("mixed encoding types detected")
		}
		if chunk.MessageID != expectedID {
			return "", "", "", errors.New("chunks from different images were mixed")
		}
		if chunk.Total != expectedTotal {
			return "", "", "", errors.New("inconsistent chunk count")
		}
		if seen[chunk.Index] {
			return "", "", "", fmt.Errorf("duplicate chunk: %d", chunk.Index)
		}
		seen[chunk.Index] = true
	}

	if len(chunks) != expectedTotal {
		missing := make([]string, 0)
		for i := 1; i <= expectedTotal; i++ {
			if !seen[i] {
				missing = append(missing, strconv.Itoa(i))
			}
		}
		return "", "", "", fmt.Errorf("missing chunks: expected %d, received %d; missing indexes: %s", expectedTotal, len(chunks), strings.Join(missing, ","))
	}

	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Index < chunks[j].Index
	})

	var builder strings.Builder
	for expectedIndex, chunk := range chunks {
		wanted := expectedIndex + 1
		if chunk.Index != wanted {
			return "", "", "", fmt.Errorf("missing chunk: %d/%d", wanted, expectedTotal)
		}
		builder.WriteString(chunk.Payload)
	}

	return expectedEncoding, expectedID, builder.String(), nil
}

func parseChunk(line string) (Chunk, error) {
	parts := strings.SplitN(line, "|", 5)
	if len(parts) != 5 {
		return Chunk{}, errors.New("invalid ImgCrypt chunk format")
	}
	if parts[0] != "IMGCRYPT" {
		return Chunk{}, errors.New("invalid ImgCrypt prefix")
	}

	encodingType := parts[1]
	if encodingType != "B64" && encodingType != "B85" {
		return Chunk{}, fmt.Errorf("unsupported encoding type: %s", encodingType)
	}

	messageID := parts[2]
	position := strings.Split(parts[3], "/")
	if len(position) != 2 {
		return Chunk{}, errors.New("invalid chunk position")
	}

	index, err := strconv.Atoi(position[0])
	if err != nil {
		return Chunk{}, errors.New("invalid chunk index")
	}

	total, err := strconv.Atoi(position[1])
	if err != nil {
		return Chunk{}, errors.New("invalid total chunk count")
	}

	if index <= 0 || total <= 0 || index > total {
		return Chunk{}, errors.New("invalid chunk index/total")
	}

	payload := parts[4]
	if payload == "" {
		return Chunk{}, errors.New("empty chunk payload")
	}

	return Chunk{
		Encoding:  encodingType,
		MessageID: messageID,
		Index:     index,
		Total:     total,
		Payload:   payload,
	}, nil
}

func generateMessageID(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

// =========================================================
// User input
// =========================================================

func readJPEGQuality() int {
	for {
		input := readLine(fmt.Sprintf("JPEG Quality [default %d]: ", defaultJPEGQuality))
		if input == "" {
			return defaultJPEGQuality
		}

		quality, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("[ERROR] Please enter a number.")
			continue
		}
		if quality < 1 || quality > 100 {
			fmt.Println("[ERROR] Quality must be between 1 and 100.")
			continue
		}
		return quality
	}
}

func readPNGCompressionMode() byte {
	for {
		fmt.Println("PNG Compression:")
		fmt.Println("1. Default")
		fmt.Println("2. Best Speed")
		fmt.Println("3. Best Compression")
		option := readLine("Mode [default 3]: ")

		switch option {
		case "", "3":
			return 3
		case "1":
			return 1
		case "2":
			return 2
		default:
			fmt.Println("[ERROR] Invalid option.")
		}
	}
}

func readEncodingType() string {
	for {
		fmt.Println()
		fmt.Println("Text encoding:")
		fmt.Println("1. Base64 (more compatible)")
		fmt.Println("2. Base85 (shorter)")
		option := readLine("Encoding [default 2]: ")

		switch option {
		case "":
			return "B85"
		case "1":
			return "B64"
		case "2":
			return "B85"
		default:
			fmt.Println("[ERROR] Invalid option.")
		}
	}
}

func readMaxMessageChars() int {
	for {
		input := readLine(fmt.Sprintf("Maximum characters per message [default %d]: ", defaultMaxMessageChars))
		if input == "" {
			return defaultMaxMessageChars
		}

		value, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("[ERROR] Please enter a number.")
			continue
		}
		if value < 80 {
			fmt.Println("[ERROR] Please use at least 80 characters.")
			continue
		}
		return value
	}
}

func readMultipleLines() ([]string, error) {
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		lines = append(lines, line)

		if errors.Is(err, io.EOF) {
			break
		}
	}
	return lines, nil
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return ""
	}
	return strings.TrimSpace(input)
}

// =========================================================
// Paths / helpers
// =========================================================

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"'")
	return path
}

var safeImageExt = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,12}$`)

func validImageName(name string) bool {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\:\x00") {
		return false
	}
	if !safeImageExt.MatchString(filepath.Ext(name)) {
		return false
	}
	return true
}

func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func fileExists(path string) bool { _, err := os.Stat(path); return err == nil }

func writeNewFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

func collectInputs(input, baseDir string, txt bool) ([]string, error) {
	parts := strings.Split(input, ";")
	if input == "" {
		parts = []string{baseDir}
	}
	seen := make(map[string]bool)
	var paths []string
	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	for _, part := range parts {
		part = cleanPath(part)
		if part == "" {
			continue
		}
		if !filepath.IsAbs(part) {
			part = filepath.Join(baseDir, part)
		}
		info, err := os.Stat(part)
		if err != nil {
			return nil, fmt.Errorf("cannot open %s: %w", part, err)
		}
		if info.IsDir() {
			entries, err := os.ReadDir(part)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if txt {
					lower := strings.ToLower(name)
					if strings.HasSuffix(lower, "-imgcrypt.txt") || strings.HasSuffix(lower, ".imgcrypt.txt") {
						add(filepath.Join(part, name))
					}
				} else if isImageExtension(filepath.Ext(name)) {
					add(filepath.Join(part, name))
				}
			}
		} else {
			if txt && !strings.EqualFold(filepath.Ext(part), ".txt") {
				return nil, fmt.Errorf("expected TXT file: %s", part)
			}
			add(part)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func pngSettingName(setting byte) string {
	switch setting {
	case 1:
		return "Default"
	case 2:
		return "Best Speed"
	case 3:
		return "Best Compression"
	default:
		return "Unknown"
	}
}

func formatBytes(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
