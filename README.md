# Abilities
## Audio input

Dependency: 
 - portaudio
 
To know what's on your computer: go run abilities/audio_input/portaudio/cmd/main.go

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