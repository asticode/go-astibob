[![GoDoc](https://godoc.org/github.com/asticode/go-astibob?status.svg)](https://godoc.org/github.com/asticode/go-astibob)

Bob is a distributed AI written in GO.

# Overview

- an **ability** is a simple task such as speech-synthesis, recording audio, etc.
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
     +----------+     +----------+
     | Brain #1 |     | Brain #2 |       
     +----------+     +----------+
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

Abilities are simple tasks such as speech-synthesis, recording audio, etc.

WARNING: the code below doesn't handle errors for readibility purposes. However you SHOULD!

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
// Init speaking
speaking := astispeaking.NewAbility(astispeaking.AbilityOptions{})

// Learn speaking
brain.Learn(speaking, astibrain.AbilityOptions{})
```

##### Bob

```go
// Say something
bob.Exec(astispeaking.CmdSay("I love you Bob"))
```

# Events

// TODO

# Roadmap

// TODO