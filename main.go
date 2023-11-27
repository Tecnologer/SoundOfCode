package main

import (
	"flag"
	"fmt"
	"github.com/ebitengine/oto/v3"
	"io"
	"math"
	"math/rand"
	"runtime"
	"time"
)

var (
	sampleRate   = flag.Int("samplerate", 48000, "sample rate")
	channelCount = flag.Int("channel-count", 2, "number of channel")
	format       = flag.String("format", "s16le", "source format (u8, s16le, or f32le)")
	currentPhase = 0.0
)

const (
	refKey      = 49
	refFreq     = 440.0
	totalKeys   = 99
	maxOvertone = 8
)

type SineWave struct {
	freq         float64
	length       int64
	pos          int64
	channelCount int
	format       oto.Format
	remaining    []byte
	envelope     *Envelope
	phase        float64
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

	// Calculate phase at end of note
	endPhase := currentPhase + 2.0*math.Pi*freq*float64(duration)

	// Use modulo operation to keep phase within range 0 to 2*pi
	endPhase = math.Mod(endPhase, 2.0*math.Pi)

	s := &SineWave{
		freq:         freq,
		length:       l,
		channelCount: channelCount,
		format:       format,
		phase:        currentPhase, // Start note at current phase
		envelope: &Envelope{
			Attack:  0.01,                    // Attack phase lasts 0.1 seconds
			Decay:   0.2,                     // Decay phase lasts 0.2 seconds
			Sustain: 0.7,                     // Sustain level is 70% of the maximum amplitude
			Release: float64(duration) * 0.1, // Release phase lasts 0.5 seconds
		},
	}

	currentPhase = endPhase // Update current phase to end phase of note

	return s
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
			fundamental := float32(math.Sin((freqFundamental*math.Pi*float64(p)/length + s.phase) * amplitudeFundamental))

			// apply envelope
			if float64(p) > length-s.envelope.Release {
				fundamental *= float32((length - float64(p)) / s.envelope.Release)
			} else {
				fundamental *= float32(s.envelope.Amplitude(float64(p)))
			}

			// Generate an overtone at twice the frequency and half the amplitude
			overtone := generateOvertone(p, freqOvertone, amplitudeOvertone, length, s.envelope.Release)
			overtone += float32(math.Sin((freqOvertone*math.Pi*float64(p)/length + s.phase) * amplitudeOvertone))

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

	keys := []int{40, 40, 47, 47, 49, 49, 47, 45, 45, 44, 44, 42, 42, 40}
	durations := []time.Duration{
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		1000 * time.Millisecond,
		750 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		1000 * time.Millisecond,
	}

	//for keyNumber := totalKeys; keyNumber >= 1; keyNumber-- {
	for i, keyNumber := range keys {
		key := pianoKeyFrequency(keyNumber)
		fmt.Printf("Key %d: %.4f Hz\n", keyNumber, key)

		duration := durations[i]
		waitDuration := 500 * time.Millisecond

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
	total := env.Attack + env.Decay + env.Sustain + env.Release
	if t < env.Attack {
		// In the attack phase, the amplitude rises linearly to 1.
		return t / env.Attack
	} else if t < env.Attack+env.Decay {
		// In the decay phase, the amplitude drops linearly to the sustain level.
		return 1 - (t-env.Attack)/env.Decay*(1-env.Sustain)
	} else if t < total-env.Release {
		// In the sustain phase, the amplitude stays at the sustain level.
		return env.Sustain
	} else {
		// In the release phase, use an exponential decay for a smoother decrease to 0.
		releaseTime := t - (total - env.Release)
		return env.Sustain * math.Exp(-(releaseTime / env.Release))
	}
}

func generateOvertone(p int64, seedFreq, seedAmp, length float64, release float64) (overtone float32) {
	var (
		r                 = rand.New(rand.NewSource(time.Now().UnixNano()))
		freqOvertone      float64
		amplitudeOvertone = seedAmp * r.Float64()
	)

	for i := 1; i <= maxOvertone; i++ {
		freqOvertone = seedFreq * float64(i)
		amplitudeOvertone *= 0.5 // decrease the amplitude for each overtone

		phase := rand.Float64() * 2 * math.Pi // random phase for each overtone
		overtone += float32(math.Sin((freqOvertone*math.Pi*float64(p)/length + phase) * amplitudeOvertone))
	}

	// Apply release envelope
	if float64(p) > length-release {
		amplitudeOvertone *= (length - float64(p)) / release
	}

	return
}
