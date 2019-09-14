package speech_to_text

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sort"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiaudio "github.com/asticode/go-astitools/audio"
	astisync "github.com/asticode/go-astitools/sync"
	"github.com/cryptix/wav"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/asticode/go-astitools/limiter"
)

// Message names
const (
	optionsBuildUpdatedMessage = "speech_to_text.options.build.updated"
	progressMessage            = "speech_to_text.progress"
	samplesMessage             = "speech_to_text.samples"
	speechCreatedMessage       = "speech_to_text.speech.created"
	speechDeletedMessage       = "speech_to_text.speech.deleted"
	speechUpdatedMessage       = "speech_to_text.speech.updated"
	textMessage                = "speech_to_text.text"
)

type Parser interface {
	Parse(samples []int32, bitDepth int, sampleRate float64) (string, error)
	Train(speeches []SpeechFile, progressFunc func(Progress))
}

type Progress struct {
	CurrentStep string   `json:"current_step"`
	Error       error    `json:"-"`
	Progress    float64  `json:"progress"`  // In percentage
	Steps       []string `json:"steps"`
}

func (p *Progress) done() bool {
	return p.Error != nil || len(p.Steps) == 0 || (p.CurrentStep == p.Steps[len(p.Steps)-1] && p.Progress == 100)
}

type Speech struct {
	CreatedAt   time.Time `json:"created_at"`
	IsValidated bool      `json:"is_validated"`
	Name        string    `json:"name"`
	Text        string    `json:"text"`
}

type SpeechFile struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type Runnable struct {
	*astibob.BaseOperatable
	*astibob.BaseRunnable
	b *astilimiter.Bucket
	c   *astisync.Chan
	i   *os.File
	mp  *sync.Mutex // Locks pg
	ms  *sync.Mutex // Locks ss
	msd *sync.Mutex // Locks sds
	o   RunnableOptions
	p   Parser
	pg  *Progress
	sds map[string]*astiaudio.SilenceDetector
	ss  map[string]*Speech
}

type RunnableOptions struct {
	SpeechesDirPath  string `toml:"speeches_dir_path"`
	StoreNewSpeeches bool   `toml:"store_new_speeches"`
}

func NewRunnable(name string, p Parser, o RunnableOptions) *Runnable {
	// Create runnable
	r := &Runnable{
		BaseOperatable: newBaseOperatable(),
		c:              astisync.NewChan(astisync.ChanOptions{}),
		mp:             &sync.Mutex{},
		ms:             &sync.Mutex{},
		msd:            &sync.Mutex{},
		o:              o,
		p:              p,
		sds:            make(map[string]*astiaudio.SilenceDetector),
		ss:             make(map[string]*Speech),
	}

	// Add routes
	r.BaseOperatable.AddRoute("/options/build", http.MethodPatch, r.updateBuildOptions)
	r.BaseOperatable.AddRoute("/references/build", http.MethodGet, r.buildReferences)
	r.BaseOperatable.AddRoute("/references/train", http.MethodGet, r.trainReferences)
	r.BaseOperatable.AddRoute("/speeches/*path", http.MethodGet, astibob.DirHandle(r.o.SpeechesDirPath))
	r.BaseOperatable.AddRoute("/speeches/:name", http.MethodDelete, r.deleteSpeech)
	r.BaseOperatable.AddRoute("/speeches/:name", http.MethodPatch, r.updateSpeech)
	r.BaseOperatable.AddRoute("/train", http.MethodGet, r.train)

	// Set base runnable
	r.BaseRunnable = astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
		Metadata: astibob.Metadata{
			Description: "Executes speech to text analysis when detecting silences in audio samples",
			Name:        name,
		},
		OnStart: r.onStart,
	})

	// Create progress limiter
	r.b = astilimiter.New().Add("progress", 20, time.Second)
	return r
}

func (r *Runnable) Init() (err error) {
	// No speeches directory
	if r.o.SpeechesDirPath == "" {
		return
	}

	// Make sure speeches directory exists
	if err = os.MkdirAll(r.o.SpeechesDirPath, 0755); err != nil {
		err = errors.Wrapf(err, "speech_to_text: mkdirall %s failed")
		return
	}

	// Open index
	p := filepath.Join(r.o.SpeechesDirPath, "index.json")
	if r.i, err = os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		err = errors.Wrapf(err, "speech_to_text: opening %s failed", p)
		return
	}

	// Stat index
	var fi os.FileInfo
	if fi, err = r.i.Stat(); err != nil {
		err = errors.Wrapf(err, "speech_to_text: stating %s failed", p)
		return
	}

	// Unmarshal index
	if fi.Size() > 0 {
		r.ms.Lock()
		if err = json.NewDecoder(r.i).Decode(&r.ss); err != nil {
			err = errors.Wrap(err, "speech_to_text: unmarshaling failed")
			return
		}
		r.ms.Unlock()
	}
	return
}

func (r *Runnable) Close() {
	// Close index
	if r.i != nil {
		r.i.Close()
	}

	// Close limiter
	if r.b != nil {
		r.b.Close()
	}
}

func (r *Runnable) onStart(ctx context.Context) (err error) {
	// Reset silence detectors
	r.msd.Lock()
	for _, sd := range r.sds {
		sd.Reset()
	}
	r.msd.Unlock()

	// Start chan
	r.c.Start(ctx)

	// Stop chan
	r.c.Stop()
	return
}

func (r *Runnable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case samplesMessage:
		if err = r.onSamples(m); err != nil {
			err = errors.Wrap(err, "speech_to_text: on samples failed")
			return
		}
	}
	return
}

type Samples struct {
	BitDepth             int                `json:"bit_depth"`
	From                 astibob.Identifier `json:"from"`
	MaxSilenceAudioLevel float64            `json:"max_silence_audio_level"`
	SampleRate           float64            `json:"sample_rate"`
	Samples              []int32            `json:"samples"`
}

func NewSamplesMessage(from astibob.Identifier, samples []int32, bitDepth int, sampleRate, maxSilenceAudioLevel float64) astibob.MessageContent {
	return astibob.MessageContent{
		Name: samplesMessage,
		Payload: Samples{
			BitDepth:             bitDepth,
			From:                 from,
			MaxSilenceAudioLevel: maxSilenceAudioLevel,
			SampleRate:           sampleRate,
			Samples:              samples,
		},
	}
}

func parseSamplesCmdPayload(m *astibob.Message) (s Samples, err error) {
	if err = json.Unmarshal(m.Payload, &s); err != nil {
		err = errors.Wrap(err, "speech_to_text: unmarshaling failed")
		return
	}
	return
}

func (r *Runnable) onSamples(m *astibob.Message) (err error) {
	// Check status
	if r.Status() != astibob.RunningStatus {
		return
	}

	// Parse payload
	var s Samples
	if s, err = parseSamplesCmdPayload(m); err != nil {
		err = errors.Wrap(err, "speech_to_text: parsing payload failed")
		return
	}

	// Invalid from
	if s.From.Type != astibob.RunnableIdentifierType || s.From.Name == nil || s.From.Worker == nil {
		err = errors.New("speech_to_text: invalid from")
		return
	}

	// Make sure samples processing is non blocking but still executed in FIFO order
	r.c.Add(r.samplesFunc(s))
	return
}

func (r *Runnable) samplesFunc(s Samples) func() {
	return func() {
		// Create silence detector key
		k := fmt.Sprintf("worker.%s.runnable.%s", *s.From.Worker, *s.From.Name)

		// Get silence detector
		r.msd.Lock()
		sd, ok := r.sds[k]
		if !ok {
			sd = astiaudio.NewSilenceDetector(astiaudio.SilenceDetectorOptions{
				MaxSilenceAudioLevel: s.MaxSilenceAudioLevel,
				SampleRate:           s.SampleRate,
			})
			r.sds[k] = sd
		}
		r.msd.Unlock()

		// Add samples to silence detector
		vss := sd.Add(s.Samples)

		// No valid samples
		if len(vss) == 0 {
			return
		}

		// Loop through valid samples
		for _, vs := range vss {
			// Parse speech
			text, err := r.parseSpeech(s.From, vs, s.BitDepth, s.SampleRate)
			if err != nil {
				astilog.Error(errors.Wrap(err, "speech_to_text: parsing samples failed"))
			}

			// Store speech
			if r.o.StoreNewSpeeches && r.o.SpeechesDirPath != "" {
				if err := r.storeSpeech(text, vs, s.BitDepth, s.SampleRate); err != nil {
					astilog.Error(errors.Wrap(err, "speech_to_text: storing speech failed"))
				}
			}
		}
	}
}

type Text struct {
	From astibob.Identifier `json:"from"`
	Text string             `json:"text"`
}

func (r *Runnable) newTextMessage(from astibob.Identifier, text string) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = textMessage

	// Marshal
	if m.Payload, err = json.Marshal(Text{
		From: from,
		Text: text,
	}); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling payload failed")
		return
	}
	return
}

func parseTextPayload(m *astibob.Message) (t Text, err error) {
	if err = json.Unmarshal(m.Payload, &t); err != nil {
		err = errors.Wrap(err, "speech_to_text: unmarshaling failed")
		return
	}
	return
}

func (r *Runnable) parseSpeech(from astibob.Identifier, ss []int32, bitDepth int, sampleRate float64) (text string, err error) {
	// No parser
	if r.p == nil {
		return
	}

	// Parse
	astilog.Debugf("speech_to_text: parsing %d samples from runnable %s on worker %s", len(ss), *from.Name, *from.Worker)
	start := time.Now()
	if text, err = r.p.Parse(ss, bitDepth, sampleRate); err != nil {
		err = errors.Wrap(err, "speech_to_text: parsing speech failed")
		return
	}
	astilog.Debugf("speech_to_text: parsed %d samples from runnable %s on worker %s in %s", len(ss), *from.Name, *from.Worker, time.Now().Sub(start))

	// Dispatch text
	if text != "" {
		// Create message
		var m *astibob.Message
		if m, err = r.newTextMessage(from, text); err != nil {
			err = errors.Wrap(err, "speech_to_text: creating text message failed")
			return
		}

		// Dispatch
		r.Dispatch(m)
	}
	return
}

func (r *Runnable) newSpeechCreatedMessage(s Speech) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = speechCreatedMessage

	// Make sure the message is sent to the UI
	m.To = &astibob.Identifier{Type: astibob.UIIdentifierType}

	// Marshal
	if m.Payload, err = json.Marshal(s); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling payload failed")
		return
	}
	return
}

func (r *Runnable) saveIndex() (err error) {
	// Lock
	r.ms.Lock()
	defer r.ms.Unlock()

	// Truncate
	if err = r.i.Truncate(0); err != nil {
		err = errors.Wrap(err, "speech_to_text: truncating index failed")
		return
	}

	// Seek
	if _, err = r.i.Seek(0, 0); err != nil {
		err = errors.Wrap(err, "speech_to_text: seeking in index failed")
		return
	}

	// Marshal
	if err = json.NewEncoder(r.i).Encode(r.ss); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling index failed")
		return
	}
	return
}

func (r *Runnable) storeSpeech(text string, ss []int32, bitDepth int, sampleRate float64) (err error) {
	// Store speech to wav
	var s *Speech
	if s, err = r.storeSpeechToWav(ss, bitDepth, sampleRate); err != nil {
		err = errors.Wrap(err, "speech_to_text: storing speech to wav failed")
		return
	}

	// Update text
	s.Text = text

	// Add speech to index
	r.ms.Lock()
	r.ss[s.Name] = s
	r.ms.Unlock()

	// Save index
	if err = r.saveIndex(); err != nil {
		err = errors.Wrap(err, "speech_to_text: saving index failed")
		return
	}

	// Create message
	var m *astibob.Message
	if m, err = r.newSpeechCreatedMessage(*s); err != nil {
		err = errors.Wrap(err, "speech_to_text: creating speech created message failed")
		return
	}

	// Dispatch
	r.Dispatch(m)
	return
}

func (r *Runnable) storeSpeechToWav(ss []int32, bitDepth int, sampleRate float64) (s *Speech, err error) {
	// Create wav file
	var f *os.File
	if f, err = ioutil.TempFile(r.o.SpeechesDirPath, "*.wav"); err != nil {
		err = errors.Wrap(err, "speech_to_text: creating wav file failed")
		return
	}
	defer f.Close()

	// Create speech
	s = &Speech{
		CreatedAt: time.Now(),
		Name:      strings.TrimSuffix(filepath.Base(f.Name()), ".wav"),
	}

	// Create wav options
	o := wav.File{
		Channels:        1,
		SampleRate:      uint32(sampleRate),
		SignificantBits: uint16(bitDepth),
	}

	// Create wav writer
	var w *wav.Writer
	if w, err = o.NewWriter(f); err != nil {
		err = errors.Wrap(err, "speech_to_text: creating wav writer failed")
		return
	}
	defer w.Close()

	// Loop through samples
	for _, v := range ss {
		// Write sample
		if err = w.WriteInt32(v); err != nil {
			err = errors.Wrap(err, "speech_to_text: writing wav sample failed")
			return
		}
	}
	return
}

type BuildReferences struct {
	Options  BuildOptions `json:"options"`
	Speeches []Speech     `json:"speeches"`
}

type BuildOptions struct {
	StoreNewSpeeches bool `json:"store_new_speeches"`
}

type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (r *Runnable) orderedSpeeches(fn func(s Speech)) {
	// Lock
	r.ms.Lock()
	defer r.ms.Unlock()

	// Index speeches
	var ks int64Slice
	is := make(map[int64][]Speech)
	for _, s := range r.ss {
		if _, ok := is[s.CreatedAt.UnixNano()]; !ok {
			ks = append(ks, s.CreatedAt.UnixNano())
		}
		is[s.CreatedAt.UnixNano()] = append(is[s.CreatedAt.UnixNano()], *s)
	}

	// Sort keys
	sort.Sort(ks)

	// Loop through keys
	for _, k := range ks {
		// Loop through speeches
		for _, s := range is[k] {
			fn(s)
		}
	}
	return
}

func (r *Runnable) buildReferences(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// Create references
	rf := BuildReferences{
		Options: BuildOptions{
			StoreNewSpeeches: r.o.StoreNewSpeeches,
		},
		Speeches: []Speech{},
	}

	// Loop through ordered speeches
	r.orderedSpeeches(func(s Speech) { rf.Speeches = append(rf.Speeches, s) })

	// Write
	astibob.WriteHTTPData(rw, rf)
}

func (r *Runnable) updateBuildOptions(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// Parse body
	var b BuildOptions
	if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
		astibob.WriteHTTPError(rw, http.StatusBadRequest, errors.Wrap(err, "speech_to_text: parsing build options payload failed"))
		return
	}

	// Update options
	r.o.StoreNewSpeeches = b.StoreNewSpeeches

	// Create message
	m, err := r.newBuildOptionsUpdatedMessage()
	if err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "speech_to_text: creating build options updated message failed"))
		return
	}

	// Dispatch
	r.Dispatch(m)
}

func (r *Runnable) newBuildOptionsUpdatedMessage() (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = optionsBuildUpdatedMessage

	// Make sure the message is sent to the UI
	m.To = &astibob.Identifier{Type: astibob.UIIdentifierType}

	// Marshal
	if m.Payload, err = json.Marshal(BuildOptions{StoreNewSpeeches: r.o.StoreNewSpeeches}); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling payload failed")
		return
	}
	return
}

func (r *Runnable) newSpeechDeletedMessage(s Speech) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = speechDeletedMessage

	// Make sure the message is sent to the UI
	m.To = &astibob.Identifier{Type: astibob.UIIdentifierType}

	// Marshal
	if m.Payload, err = json.Marshal(s); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling payload failed")
		return
	}
	return
}

func (r *Runnable) deleteSpeech(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// Get speech
	r.ms.Lock()
	s, ok := r.ss[p.ByName("name")]
	r.ms.Unlock()

	// No speech
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Delete wav
	wp := filepath.Join(r.o.SpeechesDirPath, s.Name+".wav")
	if err := os.Remove(wp); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrapf(err, "speech_to_text: deleting %s failed", wp))
		return
	}

	// Delete from index
	r.ms.Lock()
	delete(r.ss, s.Name)
	r.ms.Unlock()

	// Save index
	if err := r.saveIndex(); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "speech_to_text: saving index failed"))
		return
	}

	// Create message
	m, err := r.newSpeechDeletedMessage(*s)
	if err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "speech_to_text: creating speech deleted message failed"))
		return
	}

	// Dispatch
	r.Dispatch(m)
}

func (r *Runnable) newSpeechUpdatedMessage(s Speech) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = speechUpdatedMessage

	// Make sure the message is sent to the UI
	m.To = &astibob.Identifier{Type: astibob.UIIdentifierType}

	// Marshal
	if m.Payload, err = json.Marshal(s); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling payload failed")
		return
	}
	return
}

func (r *Runnable) updateSpeech(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// Get speech
	r.ms.Lock()
	s, ok := r.ss[p.ByName("name")]
	r.ms.Unlock()

	// No speech
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Unmarshal
	var b Speech
	if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "speech_to_text: unmarshaling failed"))
		return
	}

	// Update speech
	s.IsValidated = b.IsValidated
	s.Text = b.Text

	// Save index
	if err := r.saveIndex(); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "speech_to_text: saving index failed"))
		return
	}

	// Create message
	m, err := r.newSpeechUpdatedMessage(*s)
	if err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "speech_to_text: creating speech updated message failed"))
		return
	}

	// Dispatch
	r.Dispatch(m)
}

type TrainReferences struct {
	Progress *Progress `json:"progress,omitempty"`
}

func (r *Runnable) trainReferences(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// Create references
	r.mp.Lock()
	rf := TrainReferences{
		Progress: r.pg,
	}
	r.mp.Unlock()

	// Write
	astibob.WriteHTTPData(rw, rf)
}

type ProgressMessage struct {
	Progress
	Error string `json:"error"`
}

func (r *Runnable) newProgressMessage(p Progress) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = progressMessage

	// Make sure the message is sent to the UI
	m.To = &astibob.Identifier{Type: astibob.UIIdentifierType}

	// Create payload
	pm := ProgressMessage{Progress: p}
	if p.Error != nil {
		pm.Error = errors.Cause(p.Error).Error()
	}

	// Marshal
	if m.Payload, err = json.Marshal(pm); err != nil {
		err = errors.Wrap(err, "speech_to_text: marshaling payload failed")
		return
	}
	return
}

func (r *Runnable) train(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// No parser
	if r.p == nil {
		return
	}

	// Lock
	r.mp.Lock()
	defer r.mp.Unlock()

	// Training in progress
	if r.pg != nil {
		astibob.WriteHTTPError(rw, http.StatusBadRequest, errors.New("speech_to_text: training in progress"))
		return
	}

	// Create progress
	r.pg = &Progress{}

	// Create speech files
	var fs []SpeechFile
	r.orderedSpeeches(func(s Speech) {
		fs = append(fs, SpeechFile{
			Path: filepath.Join(r.o.SpeechesDirPath, s.Name+".wav"),
			Text: s.Text,
		})
	})

	// Train
	r.p.Train(fs, r.progressFunc)
}

func (r *Runnable) progressFunc(p Progress) {
	// Rate limit here in case Parser is spamming
	if !r.b.Inc() {
		return
	}

	// No progress
	if r.pg == nil {
		return
	}

	// Update progress
	if p.done() {
		r.pg = nil
	} else {
		*r.pg = p
	}

	// Create message
	m, err := r.newProgressMessage(p)
	if err != nil {
		astilog.Error(errors.Wrap(err, "speech_to_text: creating progress message failed"))
		return
	}

	// Dispatch
	r.Dispatch(m)
}
