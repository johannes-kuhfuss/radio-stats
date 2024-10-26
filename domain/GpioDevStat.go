// package domain defines the core data structures
package domain

import "encoding/xml"

// Data structure as received by the GPIO interface
type DevStat struct {
	XMLName xml.Name `xml:"devStat"`
	Text    string   `xml:",chardata"`
	In      []string `xml:"in"`
	Enc     []string `xml:"enc"`
	AnMax   string   `xml:"anMax"`
	AnIn    []string `xml:"anIn"`
	Sensor  []struct {
		Text string `xml:",chardata"`
		ID   string `xml:"ID,attr"`
	} `xml:"sensor"`
	PoILshare []struct {
		Text string `xml:",chardata"`
		ID   string `xml:"ID,attr"`
	} `xml:"PoILshare"`
	Out []string `xml:"out"`
}
