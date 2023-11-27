# Sound Generation in Go

This document outlines the key principles of sound generation, including relevant formulas and considerations for synthesizing realistic sounds.

## Table of Contents

1. [Introduction](#introduction)
2. [Waveforms](#waveforms)
3. [Overtones and Harmonics](#overtones-and-harmonics)
4. [ADSR Envelope](#adsr-envelope)
5. [Sound Synthesis in Go](#sound-synthesis-in-go)
6. [Run](#run)
7. [Resources](#resources)
    1. [Books](#books)
    2. [Online Resources](#online-resources)
    3. [Online Courses](#online-courses)

## Introduction

Sound synthesis is the process of generating sound artificially, often with the goal of mimicking real-world instruments. This involves understanding the physics of sound, including waveforms, frequencies, and harmonics.

## Waveforms

Sound is a type of wave. The most basic waveform is the sine wave, represented by the formula:

```text
y(t) = A * sin(2 * π * f * t + φ)
```

Where:

- `A` is the amplitude, which determines the maximum height of the wave, or the 'loudness' of the sound.
- `f` is the frequency, representing the number of cycles the wave completes per second. It's responsible for the 'pitch' of the sound - higher frequencies result in higher pitched sounds.
- `t` is the time, which is a variable representing the specific point in time for which we are calculating the wave's position.
- `φ` is the phase, which determines the initial position or 'starting point' of the wave in its cycle at t=0. It shifts the wave along the time axis, affecting the wave's alignment with the origin and other waves when multiple waves are combined.

# Sound Generation in Go

## Overtones and Harmonics

Real-world sounds usually consist of a fundamental frequency plus multiple overtones or harmonics. Each harmonic is a multiple of the fundamental frequency, and contributes to the overall timbre of the sound.

In sound synthesis, it's common to include the first 8-10 overtones. This strikes a balance between realism and computational efficiency. The lower overtones (those closest to the fundamental frequency) have the biggest impact on the timbre of the sound, so they should be prioritized.

However, the exact number of overtones to include will depend on the specific sound you're trying to synthesize. Experiment with different numbers of overtones to find the sound you want.

## Frequencies of Piano Keys

The frequencies of the keys on a piano are determined by the formula:
```text
f = refFreq * 2^((n-refKey)/12)
```


Where:
- `f` is the frequency of the note.
- `n` is the number of the key, with 1 being the lowest key and 88 being the highest.
- `refFreq` is the frequency of the reference key, usually A4, which is typically set to 440 Hz.
- `refKey` is the number of the reference key, usually the 49th key.

This formula is based on the equal-tempered scale, where each note is a fixed ratio (the 12th root of 2) higher in frequency than the previous one.

Here's a function in Go that calculates the frequency of a piano key:

```go
const (
    refKey  = 49
    refFreq = 440.0
)

func pianoKeyFrequency(key int) float64 {
    return refFreq * math.Pow(2.0, float64(key-refKey)/12.0)
}
```

## ADSR Envelope

The ADSR envelope is used to shape the amplitude of a sound over time. It consists of four phases: Attack, Decay, Sustain, and Release.

- **Attack**: The time it takes for the sound to reach its maximum level.
- **Decay**: The time it takes for the sound to reduce from the maximum level to the sustain level.
- **Sustain**: The level at which the sound is held as long as the note is held.
- **Release**: The time it takes for the sound to reduce from the sustain level to zero after the note is released.

>  During the Attack stage, the amplitude increases from 0 to its maximum level. 
>  During the Decay stage, the amplitude decreases from its maximum level to the sustain level. 
>  During the Sustain stage, the amplitude stays at the sustain level. During the Release stage, the amplitude decreases from the sustain level back to 0.


## Sound Synthesis in Go

In Go, we can use the `math` package to generate waveforms and envelopes. Here's a basic example of a sine wave generator:

```golang
func sineWave(freq float64, sampleRate float64, t float64) float64 {
    return math.Sin(2 * math.Pi * freq * t / sampleRate)
}
```

And here's an example of an ADSR envelope:

```golang
type Envelope struct {
    Attack  float64
    Decay   float64
    Sustain float64
    Release float64
}

func (env *Envelope) Amplitude(t float64) float64 {
    // ...implementation here...
}
```

## Run
    
```shell
    go run main.go -format f32le -samplerate 44100 -channel-count 2
```

## Resources

### Books

1. ["The Computer Music Tutorial"][1] by Curtis Roads: This is a comprehensive resource that covers a wide range of topics in computer music, including sound synthesis.
2. ["Designing Sound"][2] by Andy Farnell: This book provides a practical guide to creating sound effects using pure data, and it covers many principles of sound synthesis.
3. ["Musimathics: The Mathematical Foundations of Music, Volume 1"][3] by Gareth Loy: This book provides a deep dive into the mathematics behind music and sound, including the physics of sound waves and the principles of harmony and scales.

### Online Resources

1. The Synthesis of Sound by Computer: This is a classic paper by Max Mathews, one of the pioneers of computer music.
2. Introduction to Sound Synthesis: This is a series of articles from Sound on Sound magazine that covers many aspects of sound synthesis in detail.
3. Musicdsp.org: This is a collection of algorithms, techniques, and source code for synthesizing and processing music and sound.

### Online Courses

1. Introduction to Digital Sound Design: This online course from Kadenze covers the basics of sound design, including sound synthesis.
2. Audio Signal Processing for Music Applications: This course on Coursera provides a good introduction to the signal processing techniques used in music production, including sound synthesis.
3. Remember, the best way to learn is to experiment and listen to the results. Don't be afraid to try different things and see what sounds you can create.


[1]: https://www.amazon.com.mx/Computer-Music-Tutorial-second/dp/0262044919/
[2]: https://www.amazon.com/-/es/Andy-Farnell/dp/0262014416/
[3]: https://www.amazon.com/-/es/Gareth-Loy/dp/0262516551/