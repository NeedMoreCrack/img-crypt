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
	"encoding/hex"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	magicHeader = "IMGCRYPT"
	versionV1   = byte(1)
	versionV2   = byte(2)

	defaultJPEGQuality     = 50
	defaultMaxMessageChars = 500

	saltSize = 16
	keySize  = 32

	scryptN = 32768
	scryptR = 8
	scryptP = 1

	formatJPEG = byte(1)
	formatPNG  = byte(2)
	formatWebP = byte(3)
)

var reader = bufio.NewReader(os.Stdin)

type ImageInfo struct {
	FormatCode byte
	FormatName string
	Extension  string
	Setting    byte
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
	fmt.Println("1. Encrypt image -> Base64/Base85 text chunks")
	fmt.Println("2. Decrypt pasted text chunks -> Image")
	fmt.Println("3. Decrypt TXT file -> Image")
	fmt.Println("0. Exit")
	fmt.Println()
}

// =========================================================
// Encrypt flow
// =========================================================

func encryptImageToChunks() error {
	fmt.Println("=== Encrypt Image To Text Chunks ===")
	fmt.Println()

	inputPath := cleanPath(readLine("Image path: "))
	if inputPath == "" {
		return errors.New("image path cannot be empty")
	}

	imageInfo, err := detectImageFormat(inputPath)
	if err != nil {
		return err
	}

	password := readLine("Password: ")
	if password == "" {
		return errors.New("password cannot be empty")
	}

	encodingType := readEncodingType()
	maxMessageChars := readMaxMessageChars()

	fmt.Println()
	fmt.Printf("[INFO] Input format: %s\n", imageInfo.FormatName)

	processedData, originalSize, processedInfo, err := processImageForEncryption(inputPath, imageInfo)
	if err != nil {
		return err
	}

	fmt.Printf("[OK] Image processed: %s -> %s\n", formatBytes(originalSize), formatBytes(int64(len(processedData))))
	if originalSize > 0 {
		ratio := 100 - (float64(len(processedData))/float64(originalSize))*100
		fmt.Printf("[INFO] Size changed: %.2f%%\n", ratio)
	}

	fmt.Println("[INFO] Encrypting with AES-256-GCM...")
	encryptedData, err := encryptDataV2(processedData, password, processedInfo)
	if err != nil {
		return err
	}
	fmt.Printf("[OK] Encrypted size: %s\n", formatBytes(int64(len(encryptedData))))

	encodedData, err := encodeText(encryptedData, encodingType)
	if err != nil {
		return err
	}
	fmt.Printf("[OK] Encoded length: %d characters\n", len(encodedData))

	messageID := generateMessageID(encryptedData)
	chunks, err := createChunksWithMaxLength(encodedData, encodingType, messageID, maxMessageChars)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("==============================================")
	fmt.Printf("Format              : %s\n", processedInfo.FormatName)
	fmt.Printf("Encoding            : %s\n", encodingType)
	fmt.Printf("Message ID          : %s\n", messageID)
	fmt.Printf("Total chunks        : %d\n", len(chunks))
	fmt.Printf("Max chars / message : %d\n", maxMessageChars)
	fmt.Println("==============================================")
	fmt.Println()

	for _, chunk := range chunks {
		fmt.Println(chunk)
	}

	fmt.Println()
	save := strings.ToLower(readLine("Save all chunks to TXT file? (Y/n): "))
	if save == "" || save == "y" || save == "yes" {
		defaultOutput := inputPath + ".imgcrypt.txt"
		customOutput := cleanPath(readLine(fmt.Sprintf("Output path [%s]: ", defaultOutput)))
		outputPath := resolveTextOutputPath(inputPath, customOutput)

		content := strings.Join(chunks, "\n")
		if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save chunks: %w", err)
		}

		fmt.Println("[SUCCESS] Chunks saved:", outputPath)
	}

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

	return decryptChunkLines(lines)
}

// =========================================================
// Decrypt TXT file
// =========================================================

func decryptChunksFromTXT() error {
	fmt.Println("=== Decrypt TXT File ===")
	fmt.Println()

	inputPath := cleanPath(readLine("TXT file path: "))
	if inputPath == "" {
		return errors.New("TXT path cannot be empty")
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read TXT file: %w", err)
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	rawLines := strings.Split(content, "\n")
	lines := make([]string, 0, len(rawLines))

	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return errors.New("TXT file contains no ImgCrypt chunks")
	}

	fmt.Printf("[INFO] Loaded %d chunk lines from TXT.\n", len(lines))
	return decryptChunkLines(lines)
}

func decryptChunkLines(lines []string) error {
	encodingType, messageID, encodedData, err := parseAndMergeChunks(lines)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("[INFO] Encoding:", encodingType)
	fmt.Println("[INFO] Message ID:", messageID)
	fmt.Printf("[INFO] Combined encoded length: %d characters\n", len(encodedData))

	encryptedData, err := decodeText(encodedData, encodingType)
	if err != nil {
		return err
	}

	actualID := generateMessageID(encryptedData)
	if actualID != messageID {
		return fmt.Errorf("message ID verification failed: expected %s, got %s", messageID, actualID)
	}

	password := readLine("Password: ")
	if password == "" {
		return errors.New("password cannot be empty")
	}

	fmt.Println("[INFO] Decrypting AES-256-GCM...")
	decryptedData, info, err := decryptData(encryptedData, password)
	if err != nil {
		return err
	}

	defaultOutput := "decrypted" + info.Extension
	customOutput := cleanPath(readLine(fmt.Sprintf("Output path [%s]: ", defaultOutput)))
	outputPath := resolveDecryptOutputPath(defaultOutput, customOutput)

	if err := os.WriteFile(outputPath, decryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	fmt.Println()
	fmt.Println("[SUCCESS] Image restored successfully.")
	fmt.Println("Output:", outputPath)
	fmt.Println("Format:", info.FormatName)
	if info.FormatCode == formatJPEG {
		fmt.Println("JPEG Quality:", info.Setting)
	} else if info.FormatCode == formatPNG {
		fmt.Println("PNG Compression Mode:", pngSettingName(info.Setting))
	} else if info.FormatCode == formatWebP {
		fmt.Println("WebP Mode: passthrough (original WebP bytes preserved)")
	}
	fmt.Println("Image size:", formatBytes(int64(len(decryptedData))))

	return nil
}

// =========================================================
// Image format / compression
// =========================================================

func detectImageFormat(path string) (ImageInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("failed to read image: %w", err)
	}

	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ImageInfo{FormatCode: formatJPEG, FormatName: "JPEG", Extension: ".jpg"}, nil
	}

	pngMagic := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	if len(data) >= len(pngMagic) && bytes.Equal(data[:len(pngMagic)], pngMagic) {
		return ImageInfo{FormatCode: formatPNG, FormatName: "PNG", Extension: ".png"}, nil
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return ImageInfo{FormatCode: formatWebP, FormatName: "WebP", Extension: ".webp"}, nil
	}

	return ImageInfo{}, errors.New("unsupported image format: only JPG/JPEG, PNG, and WebP are supported")
}

func processImageForEncryption(inputPath string, info ImageInfo) ([]byte, int64, ImageInfo, error) {
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, 0, ImageInfo{}, fmt.Errorf("failed to read image info: %w", err)
	}

	switch info.FormatCode {
	case formatJPEG:
		quality := readJPEGQuality()
		data, err := compressJPEG(inputPath, quality)
		if err != nil {
			return nil, 0, ImageInfo{}, err
		}
		info.Setting = byte(quality)
		return data, fileInfo.Size(), info, nil

	case formatPNG:
		mode := readPNGCompressionMode()
		data, err := compressPNG(inputPath, mode)
		if err != nil {
			return nil, 0, ImageInfo{}, err
		}
		info.Setting = mode
		return data, fileInfo.Size(), info, nil

	case formatWebP:
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, 0, ImageInfo{}, fmt.Errorf("failed to read WebP: %w", err)
		}
		info.Setting = 0
		fmt.Println("[INFO] WebP is already compressed; pure-Go mode keeps the original WebP bytes unchanged.")
		return data, fileInfo.Size(), info, nil

	default:
		return nil, 0, ImageInfo{}, errors.New("unsupported image format")
	}
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

func encryptDataV2(data []byte, password string, info ImageInfo) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	result := make([]byte, 0, len(magicHeader)+1+1+1+saltSize+1+len(nonce)+len(ciphertext))
	result = append(result, []byte(magicHeader)...)
	result = append(result, versionV2)
	result = append(result, info.FormatCode)
	result = append(result, info.Setting)
	result = append(result, salt...)
	result = append(result, byte(len(nonce)))
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func decryptData(data []byte, password string) ([]byte, ImageInfo, error) {
	minimumV1 := len(magicHeader) + 1 + 1 + saltSize + 1
	if len(data) < minimumV1 {
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

	var info ImageInfo

	switch fileVersion {
	case versionV1:
		// Backward compatibility with the earlier JPEG-only format.
		if len(data) < offset+1+saltSize+1 {
			return nil, ImageInfo{}, errors.New("invalid ImgCrypt v1 data")
		}
		info = ImageInfo{FormatCode: formatJPEG, FormatName: "JPEG", Extension: ".jpg", Setting: data[offset]}
		offset++

	case versionV2:
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

	return plaintext, info, nil
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

func resolveTextOutputPath(inputPath string, userOutput string) string {
	defaultOutput := inputPath + ".imgcrypt.txt"
	if userOutput == "" {
		return defaultOutput
	}

	info, err := os.Stat(userOutput)
	if err == nil && info.IsDir() {
		return filepath.Join(userOutput, filepath.Base(inputPath)+".imgcrypt.txt")
	}
	if strings.HasSuffix(userOutput, "\\") || strings.HasSuffix(userOutput, "/") {
		return filepath.Join(userOutput, filepath.Base(inputPath)+".imgcrypt.txt")
	}
	return userOutput
}

func resolveDecryptOutputPath(defaultOutput string, userOutput string) string {
	if userOutput == "" {
		return defaultOutput
	}

	info, err := os.Stat(userOutput)
	if err == nil && info.IsDir() {
		return filepath.Join(userOutput, filepath.Base(defaultOutput))
	}
	if strings.HasSuffix(userOutput, "\\") || strings.HasSuffix(userOutput, "/") {
		return filepath.Join(userOutput, filepath.Base(defaultOutput))
	}
	return userOutput
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
