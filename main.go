package main

/* program to create a pitch perfect (440Hz) sound */

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

const (
	// Duration   = 1
	SampleRate = 44100
	// Frequency  = 4186
	// nsamps = 44100 // samples to generate
)

var (
	//π is Pi
	π = math.Pi
	//τ is tau (from Greek alphabet) constant for π*2
	τ = π * 2
)

func main() {
	fmt.Fprintf(os.Stderr, "generating sine wave..\n")
	file := "out.bin"
	f, _ := os.Create(file)
	// sound := make([]byte, 0)
	sound := generate(float32(0.3), float32(440))
	// for f := float32(27.5); f < 4186; f += 5 {
	// 	sound = append(sound, generate(0.01, f)...)
	// }

	_, _ = f.Write(sound)
	// fmt.Printf("\rWrote: %v bytes to %s\n", bw, file)
	// fmt.Fprintf(os.Stderr, "done")
}

func generate(duration, frequency float32) (sound []byte) {
	// var (
	// 	start float64 = 1.0
	// 	end   float64 = 1.0e-4
	// )
	sound = make([]byte, 0)
	nsamps := duration * SampleRate
	var angle float64 = τ / float64(nsamps)

	// decayfac := math.Pow(end/start, 1.0/float64(nsamps))
	for i := float32(0); i < nsamps; i++ {
		sample := math.Sin(angle * float64(frequency) * float64(i))
		// sample *= start
		// start *= decayfac
		fmt.Printf("%.8f\n", sample)
		var buf [8]byte
		binary.LittleEndian.PutUint32(buf[:],
			math.Float32bits(float32(sample)))

		sound = append(sound, buf[:]...)
	}
	return
}
