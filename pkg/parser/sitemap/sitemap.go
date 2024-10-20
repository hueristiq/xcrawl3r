package sitemap

import (
	"encoding/xml"
	"errors"
	"io"
)

type entry struct {
	Type            EntryType
	Location        string               `xml:"loc"`
	LastModified    string               `xml:"lastmod,omitempy"`
	ChangeFrequency EntryChangeFrequency `xml:"changefreq,omitempty"`
	Priority        float32              `xml:"priority,omitempty"`
}

func (e *entry) GetType() EntryType {
	return e.Type
}

func (e *entry) GetLocation() string {
	return e.Location
}

func (e *entry) GetChangeFrequency() EntryChangeFrequency {
	return e.ChangeFrequency
}

func (e *entry) GetPriority() float32 {
	return e.Priority
}

type EntryType string

func (t EntryType) String() (entryType string) {
	entryType = string(t)

	return
}

type EntryChangeFrequency string

func (f EntryChangeFrequency) String() (entryChangeFrequency string) {
	entryChangeFrequency = string(f)

	return
}

type Consumer func(entry Entry) (err error)

type elementParser func(*xml.Decoder, *xml.StartElement) error

type Entry interface {
	GetType() EntryType
	GetLocation() string
	GetChangeFrequency() EntryChangeFrequency
	GetPriority() float32
}

const (
	EntryTypeSitemap EntryType = "sitemap"
	EntryTypeURL     EntryType = "url"

	EntryChangeFrequencyAlways  EntryChangeFrequency = "always"
	EntryChangeFrequencyHourly  EntryChangeFrequency = "hourly"
	EntryChangeFrequencyDaily   EntryChangeFrequency = "daily"
	EntryChangeFrequencyWeekly  EntryChangeFrequency = "weekly"
	EntryChangeFrequencyMonthly EntryChangeFrequency = "monthly"
	EntryChangeFrequencyYearly  EntryChangeFrequency = "yearly"
	EntryChangeFrequencyNever   EntryChangeFrequency = "never"
)

func Parse(reader io.Reader, consumer Consumer) (err error) {
	return parseLoop(reader, func(d *xml.Decoder, se *xml.StartElement) (err error) {
		return entryParser(d, se, consumer)
	})
}

func entryParser(decoder *xml.Decoder, se *xml.StartElement, consume Consumer) (err error) {
	if se.Name.Local == "url" {
		entry := newURLEntry()

		if err = decoder.DecodeElement(entry, se); err != nil {
			return
		}

		if err = consume(entry); err != nil {
			return
		}
	}

	if se.Name.Local == "sitemap" {
		entry := newSitemapEntry()

		if err = decoder.DecodeElement(entry, se); err != nil {
			return
		}

		if err = consume(entry); err != nil {
			return
		}
	}

	return
}

func newURLEntry() (instance *entry) {
	instance = &entry{
		Type:            EntryTypeURL,
		ChangeFrequency: EntryChangeFrequencyAlways,
		Priority:        0.5,
	}

	return
}

func newSitemapEntry() (instance *entry) {
	instance = &entry{
		Type: EntryTypeSitemap,
	}

	return
}

func parseLoop(reader io.Reader, parser elementParser) (err error) {
	decoder := xml.NewDecoder(reader)

	for {
		var token xml.Token

		token, err = decoder.Token()

		if errors.Is(err, io.EOF) {
			err = nil

			break
		}

		if err != nil {
			return
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if err = parser(decoder, &se); err != nil {
			return
		}
	}

	return
}
