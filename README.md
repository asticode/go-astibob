[![GoReportCard](http://goreportcard.com/badge/github.com/asticode/go-astibob)](http://goreportcard.com/report/github.com/asticode/go-astibob)
[![GoDoc](https://godoc.org/github.com/asticode/go-astibob?status.svg)](https://godoc.org/github.com/asticode/go-astibob)

Bob is a distributed AI that can learn to understand your voice, speak back, interact with your computer and anything else you want.

It's strongly recommended to use [DeepSpeech](https://github.com/mozilla/DeepSpeech) and [PortAudio](http://www.portaudio.com) with shipped abilities.

- [I want to learn more about Bob](https://github.com/asticode/go-astibob#i-want-to-learn-more-about-bob)
- [I want to see some code](https://github.com/asticode/go-astibob#i-want-to-see-some-code)
- [I want to run an example](https://github.com/asticode/go-astibob#i-want-to-run-an-example)
- TODO [I want to learn how to add my own ability]()

# I want to learn more about Bob

## Overview

```
                            +----------------+
+-----------+               | Brain #1       |
| Client #1 |-----+   +-----|   - Ability #1 |
+-----------+     |   |     |   - Ability #2 |
                 +-----+    +----------------+
                 | Bob |
                 +-----+    +----------------+
+-----------+     |   |     | Brain #2       |
| Client #2 |-----+   +-----|   - Ability #3 |
+-----------+               |   - Ability #4 |
                            +----------------+
```

## Vocabulary

- an **ability** is a simple task such as audio recording, speech-to-text analysis, speech-synthesis, etc.
- a **brain** has one or more **abilities**
- **Bob** is connected to one or more **brains**
- **clients** connect to **Bob** to interact with **brains** and **abilities** through the **Web UI**

## Shipped abilities

- **hearing**: listen to an audio input and dispatch audio samples (audio recording).
- **speaking**: say words to your audio output (speech synthesis).
- **understanding**: detect spoken words in audio samples and execute a speech-to-text analysis on them (speech-to-text analysis).

But you can [add your own abilities]()!

## FAQ

- Why split abilities in several brains?

    Because abilities may need to run on different machines, for instance if you want to set up the **hearing** ability
    (audio recording) in different rooms of your house.

# I want to see some code

WARNING: the code below doesn't handle errors or configurations for readability purposes, however you SHOULD!

## Bob

```go
// Create Bob
bob, _ := astibob.New(astibob.Configuration{})
defer bob.Close()

// Create interfaces of the following abilities:
// - hearing (audio recording)
// - speaking (speech synthesis)
// - understanding (speech-to-text analysis)
hearing := astihearing.NewInterface(astihearing.InterfaceConfiguration{})
speaking := astispeaking.NewInterface()
understanding, _ := astiunderstanding.NewInterface(astiunderstanding.InterfaceConfiguration{})

// Declare interfaces in Bob
bob.Declare(hearing)
bob.Declare(speaking)
bob.Declare(understanding)

// Make sure the speaking ability says "Hello" whenever it's turned on
bob.On(astibob.EventNameAbilityStarted, func(e astibob.Event) bool {
    if e.Ability != nil && e.Ability.Name == speaking.Name() {
         bob.Exec(speaking.Say("Hello"))
    }
    return false
})

// Make sure Bob sends the audio samples to the understanding ability whenever the hearing ability has recorded some
hearing.OnSamples(func(brainName string, samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error {
    bob.Exec(understanding.Samples(samples, sampleRate, significantBits, silenceMaxAudioLevel))
    return nil
})

// Handle the results of the speech-to-text analysis made by the understanding ability
understanding.OnAnalysis(func(brainName, text string) error {
    astilog.Debugf("main: processing analysis <%s>", text)
    return nil
})

// Run Bob
bob.Run(context.Background())
```

## Brain

```go
// Create portaudio
p, _ := astiportaudio.New()
defer p.Close()

// Create portaudio stream
s, _ := p.NewDefaultStream(make([]int32, 192), astiportaudio.StreamOptions{})
defer s.Close()

// Create silence detector
sd := astiaudio.NewSilenceDetector(astiaudio.SilenceDetectorConfiguration{})

// Create speech to text
stt := astispeechtotext.New(astispeechtotext.Configuration{})

// Create brain
brain := astibrain.New(astibrain.Configuration{})
defer brain.Close()

// Create hearing ability
hearing := astihearing.NewAbility(s, astihearing.AbilityConfiguration{})

// Create speaking ability
speaking := astispeaking.NewAbility(astispeaking.AbilityConfiguration{})

// Create understanding ability
understanding, _ := astiunderstanding.NewAbility(stt, sd, astiunderstanding.AbilityConfiguration)

// Learn abilities
brain.Learn(hearing, astibrain.AbilityConfiguration{})
brain.Learn(speaking, astibrain.AbilityConfiguration{})
brain.Learn(understanding, astibrain.AbilityConfiguration{})

// Run the brain
brain.Run(context.Background())
```

# I want to run an example

## Installation

### Bob

Run the following command:

    $ go get -u github.com/asticode/go-astibob

### Espeak

**Only for Linux users**, visit [the official website](http://espeak.sourceforge.net/).

### DeepSpeech

2 solutions:

- follow [this unofficial guide](https://github.com/asticode/go-astideepspeech#install-deepspeech)
- visit [the official website](https://github.com/mozilla/DeepSpeech)

### PortAudio

Visit [the official website](http://www.portaudio.com).

## Run Bob

Run the following commands:

    $ cd $GOPATH/src/github.com/asticode/go-astibob
    $ go run example/bob/main.go -v

Open your browser and go to `http://127.0.0.1:6969` with username `admin` and password `admin`. You should see something like this:

![Bob is now running!](screenshots/1.png)

**Nice job, Bob is now running and waiting for brains to connect!**

## Run Brain #1

Run the following commands:

    $ cd $GOPATH/src/github.com/asticode/go-astibob
    $ go run example/brains/1/main.go -v

If everything went according to plan, you should now see something like this in your browser:

![Brain #1 is now running!](screenshots/2.png)

**Nice job, Brain #1 is now running and has connected to Bob!**

## Start the speaking ability

The toggle in the menu is red which means that the **speaking** ability is not started. Ability are stopped by default.

If you want one of your ability to start when the brain starts you can use the `AutoStart` attribute of `astibrain.AbilityConfiguration`.

Start the **speaking** ability manually by clicking on the toggle next to its name: it should slide, turn green and you should hear "Hello".

You can turn it off anytime by clicking on the toggle again.

**Nice job, the speaking ability is now started and it can now say words!**

## Run Brain #2

Run the following commands:

    $ cd $GOPATH/src/github.com/asticode/go-astibob
    $ go run example/brains/2/main.go -v

If everything went according to plan, you should now see something like this in your browser:

![Brain #2 is now running!](screenshots/3.png)

**Nice job, Brain #2 is now running and has connected to Bob!**

## Calibrate the hearing ability

In order to detect spoken words, Bob needs to detect silences.

In order to detect silences, Bob needs to know the maximum audio level of a silence which is specific to your audio input.

Fortunately, the **Web UI** provides an easy way to do that.

First off, start the **hearing** ability.

Then in your browser click on `Hearing` in the menu, click on `Calibrate`, say something and wait less than 5 seconds. You should see something like this:

![Calibration information have appeared!](screenshots/4.png)

You can see that in my case the maximum audio level is **189140040** and the suggested silence maximum audio level is **63046680**.

However based on the chart and what I've said, I'd rather set the silence maximum audio level to **35000000**.

Now that you have the correct value you need to update you brain's configuration: set the `SilenceMaxAudioLevel` attribute of `astihearing.AbilityConfiguration` in `example/brains/2/main.go` to the silence maximum audio level you feel is best and restart the brain.

**Nice job, you've calibrated the hearing ability!**

## Build a DeepSpeech model for the understanding ability

In order to understand your voice, the **understanding** ability needs to use a [DeepSpeech](https://github.com/mozilla/DeepSpeech) model trained with samples of your voice.

We'll build one with the help of the **Web UI**.

### Store recorded spoken words

The first step is to tell Bob to store the spoken words it detects.

For that, you need to set the `StoreSamples` attribute of `astiunderstanding.AbilityConfiguration` in `example/brains/2/main.go` to `true` and restart the brain.

It will store the audio samples as wav files in the directory specified by the `SamplesDirectory` attribute (`"example/tmp/understanding"` in our case).

Now that everything is set up, return to your browser, click on `Understanding` in the menu and start the **understanding** ability. Say "Bob", pause 2 seconds and repeat 2 times. Then stop the **understanding** ability.

You should now see something like this:

![Samples ready for validation have appeared!](screenshots/6.png)

### Manually add transcripts for recorded spoken words

Now that spoken words have been recorded, you need to manually add their transcript.

For that, click on the first input box: it should play the spoken words. If it doesn't, make sure the browser you're using can read 32 bits wav file (which Chrome does but Firefox unfortunately doesn't for instance).

Write the exact words you've said (in our case "Bob") and press "ENTER" for each and every recorded audio. If you're not happy with what has been recorded you can press "CTRL+ENTER" and it will remove the audio samples.

You should now see your `wav` files with their transcript in `example/tmp/understanding/validated/<date>`.

### Prepare the data for the DeepSpeech model training

Now that transcripts have been added, you need to prepare the data for the [DeepSpeech](https://github.com/mozilla/DeepSpeech) model training.

For that, run the following command:

    $ go run pkg/speechtotext/cmd/main.go -v -i example/tmp/understanding/validated -o example/tmp/understanding/prepared

You should now see an `example/tmp/understanding/prepared` directory containing the proper training data.

### Train your DeepSpeech model

WARNING: I'm not a DeepSpeech nor a deep learning expert so the command below may not be the best one. Please direct your questions to the [DeepSpeech project](https://github.com/mozilla/DeepSpeech/issues).

Now that the training data is ready, you need to train your DeepSpeech model.

For that, visit [the official repo](https://github.com/mozilla/DeepSpeech#training) and follow the guide to train a model.

Here's a simple command to train a model:

    $ python -u ${DEEPSPEECH_SRC}/DeepSpeech.py \
      	--train_files example/tmp/understanding/prepared/index.csv \
      	--dev_files example/tmp/understanding/prepared/index.csv \
      	--test_files example/tmp/understanding/prepared/index.csv \
      	--train_batch_size 1 \
      	--dev_batch_size 1 \
      	--test_batch_size 1 \
      	--n_hidden 494 \
      	--epoch 50 \
      	--checkpoint_dir example/tmp/understanding/deepspeech/checkpoint \
      	--export_dir example/tmp/understanding/deepspeech/export \
      	--alphabet_config_path example/alphabet.txt \
      	--lm_binary_path ${DEEPSPEECH_SRC}/data/lm/lm.binary \
      	--lm_trie_path ${DEEPSPEECH_SRC}/data/lm/trie

where `${DEEPSPEECH_SRC}` is the path to your local DeepSpeech repository.

You should now have an `example/tmp/deepspeech/export/output_graph.pb` file.

### Test your DeepSpeech model

Disable the `StoreSamples` attribute and restart the brain.

Then turn on the **hearing** and **understanding** abilities and simply say "Bob": yeah you're not mistaken, Bob has responded "Yes"!

**Congratulations, Bob can now understand you and speak back to you!**