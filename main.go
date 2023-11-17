// Copyright 2019 The Oto Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"github.com/ebitengine/oto/v3"
	"io"
	"math"
	"runtime"
	"time"
)

var (
	sampleRate   = flag.Int("samplerate", 48000, "sample rate")
	channelCount = flag.Int("channel-count", 2, "number of channel")
	format       = flag.String("format", "s16le", "source format (u8, s16le, or f32le)")
)

const (
	refKey      = 49
	refFreq     = 440.0
	totalKeys   = 99
	maxOvertone = 8
)

type SineWave struct {
	freq   float64
	length int64
	pos    int64

	channelCount int
	format       oto.Format

	remaining []byte
	envelope  *Envelope
}

func formatByteLength(format oto.Format) int {
	switch format {
	case oto.FormatFloat32LE:
		return 4
	case oto.FormatUnsignedInt8:
		return 1
	case oto.FormatSignedInt16LE:
		return 2
	default:
		panic(fmt.Sprintf("unexpected format: %d", format))
	}
}

func NewSineWave(freq float64, duration time.Duration, channelCount int, format oto.Format) *SineWave {
	l := int64(channelCount) * int64(formatByteLength(format)) * int64(*sampleRate) * int64(duration) / int64(time.Second)
	l = l / 4 * 4
	return &SineWave{
		freq:         freq,
		length:       l,
		channelCount: channelCount,
		format:       format,
		envelope: &Envelope{
			Attack:  0.01, // Attack phase lasts 0.1 seconds
			Decay:   0.2,  // Decay phase lasts 0.2 seconds
			Sustain: 0.7,  // Sustain level is 70% of the maximum amplitude
			Release: 0.5,  // Release phase lasts 0.5 seconds
		},
	}
}

func (s *SineWave) Read(buf []byte) (int, error) {
	if len(s.remaining) > 0 {
		n := copy(buf, s.remaining)
		copy(s.remaining, s.remaining[n:])
		s.remaining = s.remaining[:len(s.remaining)-n]
		return n, nil
	}

	if s.pos == s.length {
		return 0, io.EOF
	}

	eof := false
	if s.pos+int64(len(buf)) > s.length {
		buf = buf[:s.length-s.pos]
		eof = true
	}

	var origBuf []byte
	if len(buf)%4 > 0 {
		origBuf = buf
		buf = make([]byte, len(origBuf)+4-len(origBuf)%4)
	}

	var (
		length               = float64(*sampleRate) / s.freq
		freqFundamental      = float64(2)
		amplitudeFundamental = 0.6
		amplitudeOvertone    = amplitudeFundamental * 0.33
		freqOvertone         = freqFundamental * 2
	)

	num := formatByteLength(s.format) * s.channelCount
	p := s.pos / int64(num)
	switch s.format {
	case oto.FormatFloat32LE:
		for i := 0; i < len(buf)/num; i++ {
			// Generate the fundamental sine wave
			fundamental := float32(math.Sin(freqFundamental*math.Pi*float64(p)/length) * amplitudeFundamental)

			// add envelope
			fundamental *= float32(s.envelope.Amplitude(float64(p)))

			// Generate an overtone at twice the frequency and half the amplitude
			overtone := generateOvertone(p, freqOvertone, amplitudeOvertone, length)

			// Add the fundamental and overtone together
			sample := fundamental + overtone

			// Convert the sample to bytes and store it in the buffer
			bs := math.Float32bits(sample)
			for ch := 0; ch < *channelCount; ch++ {
				buf[num*i+4*ch] = byte(bs)
				buf[num*i+1+4*ch] = byte(bs >> 8)
				buf[num*i+2+4*ch] = byte(bs >> 16)
				buf[num*i+3+4*ch] = byte(bs >> 24)
			}
			p++
		}
	case oto.FormatUnsignedInt8:
		for i := 0; i < len(buf)/num; i++ {
			const max = 127
			b := int(math.Sin(2*math.Pi*float64(p)/length) * 0.3 * max)
			for ch := 0; ch < *channelCount; ch++ {
				buf[num*i+ch] = byte(b + 128)
			}
			p++
		}
	case oto.FormatSignedInt16LE:
		for i := 0; i < len(buf)/num; i++ {
			const max = 32767
			b := int16(math.Sin(2*math.Pi*float64(p)/length) * 0.3 * max)
			for ch := 0; ch < *channelCount; ch++ {
				buf[num*i+2*ch] = byte(b)
				buf[num*i+1+2*ch] = byte(b >> 8)
			}
			p++
		}
	}

	s.pos += int64(len(buf))

	n := len(buf)
	if origBuf != nil {
		n = copy(origBuf, buf)
		s.remaining = buf[n:]
	}

	if eof {
		return n, io.EOF
	}
	return n, nil
}

func play(context *oto.Context, freq float64, duration time.Duration, channelCount int, format oto.Format) *oto.Player {
	p := context.NewPlayer(NewSineWave(freq, duration, channelCount, format))
	p.Play()
	return p
}

func run() error {
	const (
		freqC = 261.63
		freqE = 329.63
		freqG = 392.00
	)

	op := &oto.NewContextOptions{}
	op.SampleRate = *sampleRate
	op.ChannelCount = *channelCount

	switch *format {
	case "f32le":
		op.Format = oto.FormatFloat32LE
	case "u8":
		op.Format = oto.FormatUnsignedInt8
	case "s16le":
		op.Format = oto.FormatSignedInt16LE
	default:
		return fmt.Errorf("format must be u8, s16le, or f32le but: %s", *format)
	}
	c, ready, err := oto.NewContext(op)
	if err != nil {
		return err
	}
	<-ready

	//var wg sync.WaitGroup
	var players []*oto.Player
	//var m sync.Mutex

	//wg.Add(1)
	//keyChannel := make(chan float64)

	//go func() {
	//	for key := range keyChannel {
	//		p := play(c, key, 22*time.Millisecond, op.ChannelCount, op.Format)
	//		m.Lock()
	//		players = append(players, p)
	//		m.Unlock()
	//		//time.Sleep(3 * time.Second)
	//	}

	//defer wg.Done()
	//}()

	duration := 18 * time.Millisecond
	waitDuration := getWaitDuration(duration)
	fmt.Printf("Duration: %s, Wait duration: %s\n", duration, waitDuration)

	for keyNumber := totalKeys; keyNumber >= 1; keyNumber-- {
		key := pianoKeyFrequency(keyNumber)
		fmt.Printf("Key %d: %.4f Hz\n", keyNumber, key)

		p := play(c, key, duration, op.ChannelCount, op.Format)
		players = append(players, p)
		time.Sleep(waitDuration)
	}

	time.Sleep(200 * time.Millisecond)

	// Pin the players not to GC the players.
	runtime.KeepAlive(players)

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		panic(err)
	}
}

func pianoKeyFrequency(key int) float64 {
	return refFreq * math.Pow(2.0, float64(key-refKey)/12.0)
}

func getWaitDuration(d time.Duration) time.Duration {
	return time.Duration(float64(d) * 0.60)
}

type Envelope struct {
	Attack  float64
	Decay   float64
	Sustain float64
	Release float64
}

func (env *Envelope) Amplitude(t float64) float64 {
	if t < env.Attack {
		// In the attack phase, the amplitude rises linearly to 1.
		return t / env.Attack
	} else if t < env.Attack+env.Decay {
		// In the decay phase, the amplitude drops linearly to the sustain level.
		return 1 - (t-env.Attack)/env.Decay*(1-env.Sustain)
	}

	// In the sustain phase, the amplitude stays at the sustain level.
	return env.Sustain

	// The release phase is not handled in this example, but you could add it if you need it.
}

func generateOvertone(p int64, seedFreq, seedAmp, length float64) (overtone float32) {
	var (
		freqOvertone      float64
		amplitudeOvertone = seedAmp + 0.08
		//r                 = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	for i := 0; i < maxOvertone; i++ {
		freqOvertone = seedFreq * float64(i+2)
		amplitudeOvertone += 0.08

		overtone += float32(math.Sin(freqOvertone*math.Pi*float64(p)/length) * amplitudeOvertone)
	}

	return
}
