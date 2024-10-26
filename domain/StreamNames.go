// package domain defines the core data structures
package domain

// Mapping stream URLs to stream names
var (
	StreamNames = map[string]string{
		"http://streaming.fueralle.org:80/coloradio_160.mp3": "MP3 - 160kBit/s",
		"http://streaming.fueralle.org:80/coloradio_160.ogg": "Ogg - 160kBit/s",
		"http://streaming.fueralle.org:80/coloradio_48.aac":  "AAC - 48kBit/s",
		"http://streaming.fueralle.org:80/coloradio_56.mp3":  "MP3 - 56kBit/s",
		"http://streaming.fueralle.org:80/coloradio_hq":      "Contribution Stream",
	}
)
