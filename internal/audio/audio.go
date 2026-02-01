package audio

import (
	"math"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

const (
	sampleRate = beep.SampleRate(44100)
)

var (
	initialized bool
)

// Init initializes the audio system
func Init() error {
	if initialized {
		return nil
	}

	err := speaker.Init(sampleRate, sampleRate.N(time.Second/30))
	if err != nil {
		return err
	}

	initialized = true
	return nil
}

// Close shuts down the audio system
func Close() {
	if initialized {
		speaker.Close()
		initialized = false
	}
}

// tone generates a sine wave tone at the given frequency for the given duration
func tone(freq float64, duration time.Duration) beep.Streamer {
	numSamples := sampleRate.N(duration)
	phase := 0.0
	phaseStep := 2 * math.Pi * freq / float64(sampleRate)

	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		for i := range samples {
			if numSamples <= 0 {
				return i, false
			}
			val := math.Sin(phase) * 0.3 // 0.3 volume
			samples[i][0] = val
			samples[i][1] = val
			phase += phaseStep
			numSamples--
		}
		return len(samples), true
	})
}

// squareWave generates a square wave tone (more retro/8-bit feel)
func squareWave(freq float64, duration time.Duration) beep.Streamer {
	numSamples := sampleRate.N(duration)
	phase := 0.0
	phaseStep := freq / float64(sampleRate)

	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		for i := range samples {
			if numSamples <= 0 {
				return i, false
			}
			// Square wave: positive or negative based on phase
			val := 0.2 // volume
			if math.Mod(phase, 1.0) > 0.5 {
				val = -val
			}
			samples[i][0] = val
			samples[i][1] = val
			phase += phaseStep
			numSamples--
		}
		return len(samples), true
	})
}

// PlayPaddleHit plays the sound for ball hitting a paddle
func PlayPaddleHit() {
	if !initialized {
		return
	}
	// High-pitched short beep
	speaker.Play(squareWave(880, 50*time.Millisecond))
}

// PlayWallBounce plays the sound for ball hitting top/bottom wall
func PlayWallBounce() {
	if !initialized {
		return
	}
	// Medium-pitched short beep
	speaker.Play(squareWave(440, 30*time.Millisecond))
}

// PlayScore plays the sound when a team scores
func PlayScore() {
	if !initialized {
		return
	}
	// Descending tone for score
	go func() {
		speaker.Play(squareWave(660, 100*time.Millisecond))
		time.Sleep(100 * time.Millisecond)
		speaker.Play(squareWave(440, 100*time.Millisecond))
		time.Sleep(100 * time.Millisecond)
		speaker.Play(squareWave(330, 150*time.Millisecond))
	}()
}
