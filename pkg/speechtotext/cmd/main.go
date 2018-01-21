package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"encoding/csv"

	"strconv"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Vars
var (
	input            = flag.String("i", "", "the input path")
	output           = flag.String("o", "", "the output path")
	regexpNonLetters = regexp.MustCompile("[^\\w\\s'-]*")
)

func main() {
	// Init
	flag.Parse()
	astilog.FlagInit()

	// No input path
	if len(*input) == 0 {
		astilog.Fatal("use -i to indicate the input path")
	}

	// Get absolute input path
	inputPath, err := filepath.Abs(*input)
	if err != nil {
		astilog.Fatal(errors.Wrapf(err, "filepath.abs of %s failed", *input))
	}

	// No output path
	if len(*output) == 0 {
		astilog.Fatal("use -o to indicate the output path")
	}

	// Get absolute output path
	outputPath, err := filepath.Abs(*output)
	if err != nil {
		astilog.Fatal(errors.Wrapf(err, "filepath.abs of %s failed", *output))
	}

	// Stat csv
	csvPath := filepath.Join(outputPath, "index.csv")
	_, errStat := os.Stat(csvPath)
	if errStat != nil && !os.IsNotExist(errStat) {
		astilog.Fatal(errors.Wrapf(err, "stating %s failed", csvPath))
	}

	// Open csv
	csvFile, err := os.OpenFile(csvPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		astilog.Fatal(errors.Wrapf(err, "opening %s failed", csvPath))
	}
	defer csvFile.Close()

	// Create csv writer
	w := csv.NewWriter(csvFile)
	defer w.Flush()

	// Check whether csv existed
	indexedWavFilenames := make(map[string]bool)
	if os.IsNotExist(errStat) {
		// Write header
		if err = w.Write([]string{
			"wav_filename",
			"wav_filesize",
			"transcript",
		}); err != nil {
			astilog.Fatal(errors.Wrap(err, "writing csv header failed"))
		}
		w.Flush()
	} else {
		// Create csv reader
		r := csv.NewReader(csvFile)
		r.FieldsPerRecord = 3

		// Read all
		records, err := r.ReadAll()
		if err != nil {
			astilog.Fatal(errors.Wrap(err, "reading all csv failed"))
		}

		// Index wav filenames
		for _, record := range records {
			indexedWavFilenames[record[0]] = true
		}
	}

	// Walk in input path
	if err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		// Process error
		if err != nil {
			return err
		}

		// Only process wav files
		if info.IsDir() || !strings.HasSuffix(path, ".wav") {
			return nil
		}

		// Get id
		id := strings.TrimSuffix(strings.TrimPrefix(path, inputPath), ".wav")
		astilog.Debugf("processing %s", id)

		// ID has already been processed
		wavOutputPath := filepath.Join(outputPath, id+".wav")
		if _, ok := indexedWavFilenames[wavOutputPath]; ok {
			astilog.Debugf("%s found, skipping...", wavOutputPath)
			return nil
		}

		// Retrieve transcript
		var transcript []byte
		txtPath := filepath.Join(inputPath, id+".txt")
		if transcript, err = retrieveTranscript(txtPath); err != nil {
			return errors.Wrapf(err, "retrieving transcript from %s failed", txtPath)
		}
		astilog.Debugf("transcript is %s", transcript)

		// Convert wav file
		wavInputPath := filepath.Join(inputPath, id+".wav")
		if err = convertWavFile(wavInputPath, wavOutputPath); err != nil {
			return errors.Wrapf(err, "converting wav file from %s to %s failed", wavInputPath, wavOutputPath)
		}

		// Append to csv
		if err = appendToCSV(w, wavOutputPath, string(transcript)); err != nil {
			return errors.Wrapf(err, "appending %s with transcript %s to csv failed", wavOutputPath, transcript)
		}
		return nil
	}); err != nil {
		astilog.Fatal(errors.Wrapf(err, "walking through %s failed", inputPath))
	}
}

func retrieveTranscript(txtPath string) (transcript []byte, err error) {
	// Read src
	if transcript, err = ioutil.ReadFile(txtPath); err != nil {
		err = errors.Wrapf(err, "reading %s failed", txtPath)
	}

	// To lower
	transcript = bytes.ToLower(transcript)

	// Remove non letters
	transcript = regexpNonLetters.ReplaceAll(transcript, []byte(""))
	return
}

func convertWavFile(src, dst string) (err error) {
	// TODO convert to 16 bits

	// TODO convert to 16 000 samples
	return
}

func appendToCSV(w *csv.Writer, wavPath, transcript string) (err error) {
	// Stat wav
	var fi os.FileInfo
	if fi, err = os.Stat(wavPath); err != nil {
		return errors.Wrapf(err, "stating %s failed", wavPath)
	}

	// Write
	if err = w.Write([]string{
		wavPath,
		strconv.Itoa(int(fi.Size())),
		transcript,
	}); err != nil {
		return errors.Wrap(err, "writing csv data failed")
	}
	w.Flush()
	return
}
