# SOON\_ FM 2.0 Player

The SOON\_ FM 2.0 Player supports streaming music from the following serices:

* `Google Music`
* `SoundCloud`

# Building

To build the `go` binary the following dependency libraries must be installed.

## Darwin (macOS)

Ensure the following dependencies are installed:

* Pulse Audio: `brew install pulseaudio`

Once instaalled you can setup your workspace.

1. Make your `GOPATH`: `mkdir -p ~/player/src/player`
2. Set your `GOPATH`: export `GOPATH=~/lightswatmsvc`
3. Set your `PATH`: export `PATH=$PATH:$GOPATH/bin`
4. `cd` into your `src` directory: `cd $GOPATH/src/player`
5. Clone the source code: `git clone git@github.com:soon-fm/player.git .`
6. Build: `make build`

A `sfmplayer.darwin-x86_64` binary will be generated in your current woking directory.

## Raspberry Pi ARM7

A Raspbian ARM7 compatible binary can be built via docker:

```
docker run --rm -v `pwd`:/go/src/player registry.soon.build/sfm/player:rpxc
```

This will generate a `sfmplayer.linux-arm7` binary in the current working directory.

## Events

The player responds to and emits certain events.

## Consumed Events

The `sfmplayer` will connect to a remote web socket service and will subscribe
to the following event topics:

* `player:play`: Fired when a track should start playing.
* `player:pause`: Fired to pause the player.
* `player:resume`: Fired to resume player playback.
* `player:stop`: Fired to stop the current track.

## Emitted Events

The player will emit the following events:

* `player:playing`: Fired when the player starts playing a track.
* `player:paused`: Fired when the player has paued playing a track
* `player:resumed`: Fired when the player has resumed playing.
* `player:stopped`: Fired when the player has finished playing a track.
