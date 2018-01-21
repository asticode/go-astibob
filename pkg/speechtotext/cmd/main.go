package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"io"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/cryptix/wav"
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

	// Create csv dir
	dirPath := filepath.Dir(csvPath)
	if err = os.MkdirAll(dirPath, 0755); err != nil {
		astilog.Fatal(errors.Wrapf(err, "mkdirall %s failed", dirPath))
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

		// ID has already been processed
		wavOutputPath := filepath.Join(outputPath, id+".wav")
		if _, ok := indexedWavFilenames[wavOutputPath]; ok {
			astilog.Debugf("skipping %s", id)
			return nil
		}

		// Retrieve transcript
		var transcript []byte
		txtPath := filepath.Join(inputPath, id+".txt")
		if transcript, err = retrieveTranscript(txtPath); err != nil {
			astilog.Error(errors.Wrapf(err, "retrieving transcript from %s failed", txtPath))
			return nil
		}

		// Convert wav file
		wavInputPath := filepath.Join(inputPath, id+".wav")
		if err = convertWavFile(wavInputPath, wavOutputPath); err != nil {
			astilog.Error(errors.Wrapf(err, "converting wav file from %s to %s failed", wavInputPath, wavOutputPath))
			return nil
		}

		// Append to csv
		if err = appendToCSV(w, wavOutputPath, string(transcript)); err != nil {
			astilog.Error(errors.Wrapf(err, "appending %s with transcript %s to csv failed", wavOutputPath, transcript))
			return nil
		}

		// Log
		astilog.Infof("added %s", id)
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
	// Stat src
	var fi os.FileInfo
	if fi, err = os.Stat(src); err != nil {
		return errors.Wrapf(err, "stating %s failed", src)
	}

	// Open src
	var srcFile *os.File
	if srcFile, err = os.Open(src); err != nil {
		return errors.Wrapf(err, "opening %s failed", src)
	}
	defer srcFile.Close()

	// Create wav reader
	var r *wav.Reader
	if r, err = wav.NewReader(srcFile, fi.Size()); err != nil {
		return errors.Wrap(err, "creating wav reader failed")
	}

	// Get samples
	var samples []int32
	var sample int32
	for {
		// Read sample
		if sample, err = r.ReadSample(); err != nil {
			if err != io.EOF {
				return errors.Wrap(err, "reading wav sample failed")
			}
			break
		}

		// Append sample
		samples = append(samples, sample)
	}

	// Create dst dir
	dstDir := filepath.Dir(dst)
	if err = os.MkdirAll(dstDir, 0755); err != nil {
		return errors.Wrapf(err, "mkdirall %s failed", dstDir)
	}

	// Create dst file
	var dstFile *os.File
	if dstFile, err = os.Create(dst); err != nil {
		return errors.Wrapf(err, "creating %s failed", dst)
	}
	defer dstFile.Close()

	// Create wav file
	wavFile := wav.File{
		Channels:        1,
		SampleRate:      16000,
		SignificantBits: 16,
	}

	// Create wav writer
	var w *wav.Writer
	if w, err = wavFile.NewWriter(dstFile); err != nil {
		return errors.Wrap(err, "creating wav writer failed")
	}
	defer w.Close()

	// Convert sample rate
	if samples, err = astiaudio.ConvertSampleRate(samples, int(r.GetFile().SampleRate), int(wavFile.SampleRate)); err != nil {
		return errors.Wrap(err, "converting sample rate failed")
	}

	// Loop through samples
	for _, sample := range samples {
		// Convert bit depth
		if sample, err = astiaudio.ConvertBitDepth(sample, int(r.GetFile().SignificantBits), int(wavFile.SignificantBits)); err != nil {
			return errors.Wrap(err, "converting bit depth failed")
		}

		// Write
		if err = w.WriteSample([]byte{byte(sample&0xff), byte(sample>>8&0xff)}); err != nil {
			return errors.Wrap(err, "writing wav sample failed")
		}
	}
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
