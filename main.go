package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var DefaultRules = map[string]string{
	".jpg":  "Images",
	".jpeg": "Images",
	".png":  "Images",
	".pdf":  "Documents",
	".doc":  "Documents",
	".docx": "Documents",
	".txt":  "Documents",
	".mp3":  "Music",
	".wav":  "Music",
	".mp4":  "Video",
	".avi":  "Video",
	".zip":  "Archives",
	".rar":  "Archives",
}

type FileStats struct {
	Count     int
	TotalSize int64
}

type FileOrganizer struct {
	sourceDir      string
	rulesMap       map[string]string
	processedFiles int
	logFile        *os.File
	statistics     map[string]*FileStats
}

func NewFileOrganizer(sourceDir string, logFile *os.File) *FileOrganizer {

	return &FileOrganizer{
		sourceDir:      sourceDir,
		rulesMap:       DefaultRules,
		processedFiles: 0,
		logFile:        logFile,
		statistics:     make(map[string]*FileStats),
	}
}

func (fo *FileOrganizer) logSuccess(message string) {

	log.SetOutput(fo.logFile)

	log.Printf("[SUCCESS] %s", message)
}

func (fo *FileOrganizer) logError(message string) {

	log.SetOutput(fo.logFile)

	log.Printf("[ERROR] %s", message)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}

func generateNewFileName(path string) string {
	baseName := filepath.Base(path)

	extension := filepath.Ext(baseName)

	nameWithoutExt := strings.TrimSuffix(baseName, extension)

	timestamp := time.Now().Format("20060102_150405") // YYYYMMDD_HHMMSS

	return nameWithoutExt + "_" + timestamp + extension

}

func (fo *FileOrganizer) moveFile(sourcePath, targetDir string) error {

	fmt.Println(targetDir)

	hasTargetDir := dirExists(targetDir)

	if hasTargetDir == false {

		err := os.MkdirAll(targetDir, 0750)

		if err != nil {
			fo.logError(err.Error())
			return err
		}
	}

	newPath := filepath.Join(targetDir, filepath.Base(sourcePath))

	if fileExists(newPath) {

		newPath = filepath.Join(targetDir, generateNewFileName(sourcePath))

	}

	fo.logSuccess("Исходный файл: " + sourcePath)

	fo.logSuccess("Целевая директория: " + targetDir)

	err := os.Rename(sourcePath, newPath)

	if err != nil {
		fo.logError(err.Error())
		return err

	}

	stats, exists := fo.statistics[filepath.Ext(newPath)]

	if !exists {
		stats = &FileStats{}
		fo.statistics[filepath.Ext(newPath)] = stats
	}

	fileInfo, e := os.Stat(newPath)

	if e != nil {
		fo.logError("Ошибка получение информации о файле: " + newPath)
		return e

	}

	stats.Count += 1
	stats.TotalSize += fileInfo.Size()

	fo.logSuccess("Результат: " + newPath)

	return nil
}

func bytesToMegabytes(bytes int64) float64 {
	const bytesPerMB = 1024 * 1024
	return float64(bytes) / bytesPerMB
}

func (fo *FileOrganizer) showStats() {
	totalFiles := 0
	var totalSize int64 = 0
	byExtensionMessages := []string{}

	for ext, stats := range fo.statistics {
		totalFiles += stats.Count
		totalSize += stats.TotalSize

		byExtensionMessages = append(byExtensionMessages, fmt.Sprintf("%s:\n", DefaultRules[ext]))
		byExtensionMessages = append(byExtensionMessages, fmt.Sprintf("	- Количество файлов: %d\n", stats.Count))
		byExtensionMessages = append(byExtensionMessages, fmt.Sprintf("	- Общий размер: %.1f MB\n", bytesToMegabytes(stats.TotalSize)))
	}

	fmt.Println("=== Отчет о перемещении файлов ===")
	fmt.Println()
	fmt.Printf("Всего обработано файлов: %d\n", totalFiles)
	fmt.Printf("Общий размер: %.1f MB\n", bytesToMegabytes(totalSize))
	fmt.Println()
	fmt.Println("Статистика по категориям:")

	for _, message := range byExtensionMessages {
		fmt.Print(message)
	}
}

func (fo *FileOrganizer) Organize() error {
	err := filepath.Walk(fo.sourceDir, func(path string, pathinfo fs.FileInfo, er error) error {
		if er != nil {
			fo.logError(fmt.Sprintf("Ошибка доступа к %s: %v\n", path, er))
			return nil
		}

		dir, exists := DefaultRules[filepath.Ext(path)]

		if exists {
			dir = filepath.Join(fo.sourceDir, dir)

			err := fo.moveFile(path, dir)

			if err != nil {
				fo.logError(err.Error())
				return err
			}

		}

		return nil
	})

	fo.showStats()

	return err
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	// Шаг 1: Спрашиваем путь к директории
	fmt.Println("Добро пожаловать в Органайзер файлов!")
	fmt.Println("==================================")
	fmt.Println("ИНСТРУКЦИИ ПО ИСПОЛЬЗОВАНИЮ:")
	fmt.Println("1. Программа проанализирует все файлы в указанной директории.")
	fmt.Println("2. Файлы будут перемещены в папки согласно их расширениям.")
	fmt.Println("3. Для каждого типа файлов будет показана:")
	fmt.Println("   • Количество файлов")
	fmt.Println("   • Общий размер в мегабайтах")
	fmt.Println("4. Результаты будут выведены в конце работы программы.")
	fmt.Println("==================================")
	fmt.Print("Введите путь к директории для анализа: ")

	if !scanner.Scan() {
		fmt.Println("Ошибка чтения ввода.")
		return
	}

	dirPath := scanner.Text()

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Printf("❌ Ошибка: Директория '%s' не существует.\n", dirPath)
		return
	}

	fmt.Printf("✅ Директория найдена: %s\n\n", dirPath)

	file, er := os.OpenFile("organizer.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)

	if er != nil {
		log.Fatal(er)
	}

	fo := NewFileOrganizer(dirPath, file)

	err := fo.Organize()

	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}

}
