package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

var ErrOpenFileKeywords = errors.New("error opening the file with keywords")
var ErrOpenFileText = errors.New("error opening the file with text")
var ErrInvalidNumOfProcesses = errors.New("error converting number of processes to int ")

type keyWords struct {
	sync.RWMutex
	wordCount  map[string]int
	totalCount int
}

func (kw *keyWords) UpdateValue(currKeywordsCount map[string]int) {
	kw.Lock()
	for word := range currKeywordsCount {
		kw.wordCount[word] += currKeywordsCount[word]
		kw.totalCount += currKeywordsCount[word]
	}
	kw.Unlock()
}

func main() {

	if len(os.Args) != 4 {
		fmt.Println("Необходимо указать 3 аргумента: путь к файлу с текстом, путь к файлу с ключевыми словами, кол-во процессов.")
		return
	}

	filePathInput := os.Args[1]
	filePathKeywords := os.Args[2]
	numberOfProcesses, error := strconv.Atoi(os.Args[3])
	if error != nil {
		fmt.Fprintf(os.Stderr, "Произошла ошибка: %v\n", ErrInvalidNumOfProcesses)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := run(ctx, filePathInput, filePathKeywords, numberOfProcesses); err != nil {
		fmt.Fprintf(os.Stderr, "Произошла ошибка: %v\n", err)
		os.Exit(1)
	}
}

func getKeyWordsFromFile(filePathKeywords string) ([]string, error) {
	keywordsFile, err := os.Open(filePathKeywords)
	if err != nil {
		return []string{}, ErrOpenFileKeywords
	}
	defer keywordsFile.Close()

	keywordsList := make([]string, 0)

	scanner := bufio.NewScanner(keywordsFile)
	for scanner.Scan() {
		word := scanner.Text()
		keywordsList = append(keywordsList, word)
	}
	return keywordsList, nil

}

func run(ctx context.Context, filePathInput, filePathKeywords string, numberOfProcesses int) error {

	keywordsList, err := getKeyWordsFromFile(filePathKeywords)
	if err != nil {
		return err
	}

	input, err := os.Open(filePathInput)
	if err != nil {
		return ErrOpenFileText
	}
	defer input.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	in := make(chan string)
	go func() {
		defer close(in)
		read(ctx, input, in)
	}()

	var wg sync.WaitGroup
	wg.Add(numberOfProcesses)

	keywords := keyWords{wordCount: make(map[string]int, len(keywordsList))}

	for i := 0; i < numberOfProcesses; i++ {
		go process(ctx, in, &keywords, keywordsList, &wg)
	}

	wg.Wait()
	write(keywordsList, &keywords)

	return ctx.Err()
}

func read(ctx context.Context, input io.Reader, out chan<- string) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			out <- scanner.Text()
		}
	}
}

func process(ctx context.Context, in <-chan string, keywords *keyWords, keywordsList []string, wg *sync.WaitGroup) {
	defer wg.Done()

	for readLine := range in {
		currKeywordsCount := make(map[string]int, len(keywordsList))
		for _, word := range keywordsList {
			currKeywordsCount[word] = strings.Count(strings.ToLower(readLine), word)
		}
		keywords.UpdateValue(currKeywordsCount)
	}

}

func write(keywordsList []string, keywords *keyWords) {
	for _, word := range keywordsList {
		fmt.Printf("%s: %d\n", word, keywords.wordCount[word])
	}
	fmt.Printf("%s: %d\n", "всего", keywords.totalCount)
}
