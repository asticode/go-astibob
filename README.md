[![GoReportCard](http://goreportcard.com/badge/github.com/asticode/go-astibob)](http://goreportcard.com/report/github.com/asticode/go-astibob)
[![GoDoc](https://godoc.org/github.com/asticode/go-astibob?status.svg)](https://godoc.org/github.com/asticode/go-astibob)

Golang framework to build a [Jarvis](https://www.youtube.com/watch?v=Wx7RCJvoCMc)-like AI that can learn to understand your voice, speak back and anything else you want.

WARNING: the code below doesn't handle errors or configurations for readability purposes, however you SHOULD!

# How it works

## Overview

```
     THE WORLD      |   MACHINE #1    |        MACHINE #2               |        MACHINE #3
____________________|_________________|_________________________________|___________________________
                    |                 |                                 |
                    |                 |        +---------------------+  |
+-----------+   WS + HTTP           WS + HTTP  |   Worker #1         |      HTTP
|   UI #1   |--------------+     +-------------|     - Ability #1    |---------------+
+-----------+       |      |     |    |        |     - Ability #2    |  |            |
                    |  +-----------+  |        +---------------------+  |            |
                    |  |   Index   |  |                                 |            |
                    |  +-----------+  |                                 |  +---------------------+
+-----------+       |      |     |    |                                 |  |   Worker #2         |
|   UI #2   |--------------+     +-----------------------------------------|     - Ability #3    |
+-----------+   WS + HTTP           WS + HTTP                           |  |     - Ability #4    |
                    |                 |                                 |  +---------------------+
                    |                 |                                 |
```

- humans interact with the AI through the **Web UI**
- the **Web UI** interacts with the AI through the **Index**
- the **Index** keeps an updated list of all **Workers** and forwards **Web UI** messages to **Workers** and vice versa
- **Workers** have one or more **Abilities**
- **Abilities** can run a simple task through the **Runnable** interface (e.g. reading an audio input, executing a speech-to-text analysis, doing speech-synthesis etc.)
- **Abilities** can also implement the **Listenable** interface that allow listening to the remote **Ability** messages or the **Operatable** interface that allow operating it through the **Web UI**
- All communication are done with JSON **Messages** exchanged through HTTP or Websocket

## FAQ

- Why split abilities in several workers?

    Because abilities may need to run on different machines located in different part of the world. The simplest example is wanting to listen to microphones located in several rooms of your house. Each microphone is an ability whereas each room of your house is a worker.

# I want to see some code

## Index

```go
// Create index
i, _ := index.New(index.Options{
    Server: astibob.ServerOptions{
        Addr:     "127.0.0.1:4000",
        Password: "admin",
        Username: "admin",
    },
})

// Make sure to properly close the index
defer i.Close()

// Handle signals
i.HandleSignals()

// Serve
i.Serve()

// Blocking pattern
i.Wait()
```

## Worker

```go
// Create worker
w := worker.New("Worker #1", worker.Options{
    Index: astibob.ServerOptions{
        Addr:     "127.0.0.1:4000",
        Password: "admin",
        Username: "admin",
    },
    Server: astibob.ServerOptions{Addr: "127.0.0.1:4001"},
})

// Make sure to properly close the worker
defer w.Close()

// Create runnables
r1 := pkg1.NewRunnable("Runnable #1")
r2 := pkg2.NewRunnable("Runnable #2")

// Register runnables
w.RegisterRunnables(
	worker.Runnable{
        AutoStart: true,
        Runnable:  r1,
    },
	worker.Runnable{
        Runnable:  r2,
    },
)

// Create listenables
l1 := pkg3.NewListenable(pkg3.ListenableOptions{
	OnEvent: func(arg string) { log.Println(arg) },
})
l2 := pkg4.NewListenable()

// Register listenables
w.RegisterListenables(
	worker.Listenable{
        Listenable: l1,
        Runnable:   "Runnable #1",
        Worker:     "Worker #1",
    },
	worker.Listenable{
        Listenable: l2,
        Runnable:   "Runnable #3",
        Worker:     "Worker #2",
    },
)

// Handle signals
w.HandleSignals()

// Serve
w.Serve()

// Register to index
w.RegisterToIndex()

// Blocking pattern
w.Wait()
```

# Provided abilities

## Audio input

This ability allows you reading from an audio stream e.g. a microphone.

It's strongly recommended to use [PortAudio](http://www.portaudio.com).

To know which devices are available on the machine run:

```
$ go run abilities/audio_input/portaudio/cmd/main.go
```

### Runnable and operatable

```go
// Create portaudio
p := portaudio.New()

// Initialize portaudio
p.Initialize()

// Make sure to close portaudio
defer p.Close()

// Create default stream
s, _ := p.NewDefaultStream(portaudio.StreamOptions{
    BitDepth:             32,
    BufferLength:         5000,
    MaxSilenceAudioLevel: 5 * 1e6,
    NumInputChannels:     2,
    SampleRate:           sampleRate,
})

// Create runnable
r := audio_input.NewRunnable("Audio input", s)

// Register runnables
w.RegisterRunnables(worker.Runnable{
    AutoStart: true,
    Runnable:  r,
})

// Register listenables
// This is mandatory for the Web UI to work properly
w.RegisterListenables(worker.Listenable{
    Listenable: r,
    Runnable:   "Audio input",
    Worker:     "Worker #1",
})
```

### Listenable

```go
// Register listenables
w.RegisterListenables(
    worker.Listenable{
        Listenable: audio_input.NewListenable(audio_input.ListenableOptions{
            OnSamples: func(from astibob.Identifier, samples []int, bitDepth, numChannels, sampleRate int, maxSilenceAudioLevel float64) (err error) {
                // TODO
                return
            },
        }),
        Runnable: "Audio input",
        Worker:   "Worker #1",
    },
)
```

## Text to Speech

This ability allows you running speech synthesis.

If you're using Linux it's strongly recommended to use [ESpeak](http://espeak.sourceforge.net/).

### Runnable

```go
// Create speaker
s := speak.New(speak.Options{})

// Initialize speaker
s.Initialize()

// Make sure to close speaker
defer s.Close()

// Register runnables
w.RegisterRunnables(worker.Runnable{
    AutoStart: true,
    Runnable:  text_to_speech.NewRunnable("Text to Speech", s),
})

// Say something
w.SendMessages("Worker #1", "Text to Speech", text_to_speech.NewSayMessage("Hello world!"))
```

# Create your own ability

Creating your own ability is pretty straight-forward: you need to create an object that implements the **astibob.Runnable** interface. Optionally it can implement the **astibob.Operatable** interface as well.

If you want other abilities to be able to interact with it you'll need to create another object that implements the **astibob.Listenable** interface.

I strongly recommend checking out how provided abilities are built and trying to copy them first.

## Runnable

The quickest way to implement the **astibob.Runnable** interface is to add an embedded **astibob.BaseRunnable** attribute to your object. 

You can then use **astibob.NewBaseRunnable** to initialize it which allows you providing the proper options.

## Operatable

The quickest way to implement the **astibob.Operatable** interface is to add an embedded **astibob.BaseOperatable** attribute to your object.

You can then use the `cmd/operatable` command to generate an `operatable.go` file binding your `resources` folder containing your `static` and `template` files. You can finally add custom routes manually to the **astibob.BaseOperatable** using the **AddRoute** method.

## Listenable

No shortcut here, you need to create an object that implements the **astibob.Listenable** interface yourself.

# Abilities

## Speech to text

Don't forget to allow audio in your browser

>16 bits audio are not supported by Firefox

### Deepspeech

Install only for parsing:
- create working dir
- download a client `native_client.<your system>.tar.xz` matching your system at the bottom of [client](https://github.com/mozilla/DeepSpeech/releases/tag/v0.5.1)
- create lib dir in working dir and extract the client content into it
- create include dir in working dir and download [deepspeech.h](https://github.com/mozilla/DeepSpeech/raw/v0.5.1/native_client/deepspeech.h) into it
- download [model](https://github.com/mozilla/DeepSpeech/releases/download/v0.5.1/deepspeech-0.5.1-models.tar.gz)
- create model dir in working dir and copy the downloaded content into it
- backup the output_graph.pb file since we'll overwrite it
- you should now have "model", "include" and "lib" dirs in your working dir
- whenever you run a worker with deepspeech, make sure to have the following environment variables:

CGO_CXXFLAGS="-I<full path to the include dir you've created>"
CGO_LDFLAGS="-L<full path to the lib dir you've created>"
LIBRARY_PATH=<full path to the lib dir you've created>:$LIBRARY_PATH
LD_LIBRARY_PATH=<full path to the lib dir you've created>:$LD_LIBRARY_PATH

Install for training too:
- clone git clone https://github.com/mozilla/DeepSpeech into working dir
- [install deepspeech](https://github.com/mozilla/DeepSpeech#training-your-own-model)

# Add an ability
 
- use cmd/operatable to generate operatable.go

# Update index

- use cmd/index to generate resources.go