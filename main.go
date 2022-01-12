package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/speaker"
	"github.com/go-co-op/gocron"
)

type Queue struct {
	current beep.StreamSeekCloser
	next    beep.StreamSeekCloser
}

func (q *Queue) SetNext(s beep.StreamSeekCloser) {
	if q.current == nil {
		q.current = s
	} else {
		q.next = s
	}
}

func (q *Queue) Stream(samples [][2]float64) (n int, ok bool) {
	for len(samples) > 0 {
		// We stream from the current streamer
		sn, ok := q.current.Stream(samples)

		// Loop current stream if next stream isn't ready
		// otherwise switch to next stream
		if !ok {
			if q.next == nil {
				err := q.current.Seek(0)
				if err != nil {
					return n, true
				}
			} else {
				q.current.Close()
				q.current = q.next
				q.next = nil

				log.Println("Now playing hour " + fmt.Sprintf("%d", getHour()))
			}

			continue
		}

		// We update the number of filled samples.
		samples = samples[sn:]
		n += sn
	}
	return n, true
}

func (q *Queue) Err() error {
	return nil
}

func getHour() int {
	hours, _, _ := time.Now().Clock()
	return hours
}

func getCurrentTrack() string {
	hour := getHour()
	hourString := fmt.Sprintf("%02d", hour)
	return "music/" + hourString + ".flac"
}

func loadTrack(path string) (beep.StreamSeekCloser, beep.Format) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := flac.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	return streamer, format
}

var lastTrack string

func main() {
	s := gocron.NewScheduler(time.UTC)
	lastTrack = getCurrentTrack()
	streamer, format := loadTrack(lastTrack)

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	var queue Queue
	queue.SetNext(streamer)

	done := make(chan bool)
	speaker.Play(beep.Seq(&queue, beep.Callback(func() {
		done <- true
	})))

	s.Every(2).Minutes().Do(func() {
		newTrack := getCurrentTrack()
		if newTrack != lastTrack {
			streamer, _ = loadTrack(getCurrentTrack())
			queue.SetNext(streamer)
			lastTrack = newTrack
		}
	})

	s.StartAsync()

	<-done
}
