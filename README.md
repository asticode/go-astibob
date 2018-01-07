[![GoDoc](https://godoc.org/github.com/asticode/go-astibob?status.svg)](https://godoc.org/github.com/asticode/go-astibob)

Bob is a distributed AI written in GO.

# Overview

- an **ability** is a simple task such as audio recording, speech-to-text analysis, speech-synthesis, etc.
- a **brain** has one or more **abilities**
- **Bob** is connected to one or more **brains**
- **clients** connect to **Bob** to interact with **brains** and **abilities**

```
          +-----------+     +-----------+
          | Client #1 |     | Client #2 |
          +-----------+     +-----------+
                     \       /
                      \     /
                      +-----+
                      | Bob |
                      +-----+
                      /     \
                     /       \
     +----------------+     +----------------+
     | Brain #1       |     | Brain #2       |
     |   - Ability #1 |     |   - Ability #3 |
     |   - Ability #2 |     |   - Ability #4 |
     +----------------+     +----------------+
```
              
# Installation

Run the following command:

    $ go get -u github.com/asticode/go-astibob/...
    
# Examples

Every ability is demonstrated in an example located in `examples/<ability name>`.

To try any of the examples:

- make sure you're in `go-astibob` directory:

```$ cd $GOPATH/src/github.com/asticode/go-astibob```

- run the following command to start **Bob**:

```$ go run examples/<ability name>/astibob/main.go -v```

- run the following command to start a **brain**:

```$ go run examples/<ability name>/astibrain/main.go -v```

WARNING: some abilities require specific libs installed on your system, please check the next section to see which ones.

# Abilities

Abilities are simple tasks audio recording, speech-to-text analysis, speech-synthesis, etc.

WARNING: the code below doesn't handle errors for readibility purposes. However you SHOULD!

## Hearing

This ability listens to an audio input and dispatches audio samples.

### Recommendations

It requires the following interface:

```go
type SampleReader interface {
	ReadSample() (int32, error)
}
```

which is best fulfilled with [PortAudio](http://www.portaudio.com/). However you can choose any other solution that fulfill that interface.

The example needs [PortAudio](http://www.portaudio.com/) to be set up on your system and shows you how to use it.

### In your code
#### Brain

```go
// Create sample reader
var r astihearing.SampleReader

// Create ability
hearing := astihearing.NewAbility(r, astihearing.AbilityOptions{})

// Learn ability
brain.Learn(hearing, astibrain.AbilityOptions{})
```

#### Bob

```go
// Create interface
hearing := astihearing.NewInterface()

// Handle samples
hearing.OnSamples(func(samples []int32, sampleRate, significantBits int) error {
    // Do stuff with the samples
    return nil
})

// Declare interface
bob.Declare(hearing)
```

## Speaking

This ability says words to your audio output using speech synthesis.

### Prerequisites
#### Linux

- [espeak](http://espeak.sourceforge.net/)

#### MacOSX

- [say](https://developer.apple.com/legacy/library/documentation/Darwin/Reference/ManPages/man1/say.1.html) which should be there by default

#### Windows

N/A

### In your code
#### Brain

```go
// Create ability
speaking := astispeaking.NewAbility(astispeaking.AbilityOptions{})

// Learn ability
brain.Learn(speaking, astibrain.AbilityOptions{})
```

##### Bob

```go
// Create interface
speaking := astispeaking.NewInterface()
	
// Say something
bob.Exec(speaking.Say("I love you Bob"))
```

## Understanding

This ability executes a speech to text analysis on audio samples.

### Recommendations

It requires the following interface:

```go
type SpeechParser interface {
	SpeechToText(buffer []int32, bufferSize, sampleRate, significantBits int) string
}
```

which is best fulfilled with [DeepSpeech](https://github.com/mozilla/DeepSpeech). However you can choose any other solution that fulfill that interface.

The example requires [DeepSpeech](https://github.com/asticode/go-astideepspeech#install-deepspeech) to be set up on your system and shows you how to use it.

### In your code
#### Brain

```go
// Create speech parser
var p astiunderstanding.SpeechParser

// Create ability
understanding := astiunderstanding.NewAbility(p, , astiunderstanding.AbilityOptions{})

// Learn ability
brain.Learn(understanding, astibrain.AbilityOptions{})
```

#### Bob

```go
// Create interface
understanding := astiunderstanding.NewInterface()

// Handle analysis
understanding.OnAnalysis(func(text string) error {
    // Do stuff with the text
    return nil
})

// Send samples
bob.Exec(understanding.Samples(samples, sampleRate, significantBits))
```

# Events

// TODO

# Roadmap

// TODO