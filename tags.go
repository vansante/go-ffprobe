package ffprobe

import (
	"errors"
	"fmt"
	"strconv"
)

// ErrTagNotFound is a sentinel error used when a queried tag does not exist
var ErrTagNotFound = errors.New("tag not found")

// Tags is the map of tag names to values
type Tags map[string]interface{}

// GetInt returns a tag value as int64 and an error if one occurred.
// ErrTagNotFound will be returned if the key can't be found, ParseError if
// a parsing error occurs.
func (t Tags) GetInt(tag string) (int64, error) {
	v, found := t[tag]
	if !found || v == nil {
		return 0, ErrTagNotFound
	}

	switch v := v.(type) {
	case string:
		return valToInt64(v)
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	}

	str := fmt.Sprintf("%v", v)
	return valToInt64(str)
}

func valToInt64(str string) (int64, error) {
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("int64 parsing error (%v): %w", str, err)
	}
	return val, nil
}

// GetString returns a tag value as string and an error if one occurred.
// ErrTagNotFound will be returned if the key can't be found
func (t Tags) GetString(tag string) (string, error) {
	v, found := t[tag]
	if !found || v == nil {
		return "", ErrTagNotFound
	}
	return valToString(v), nil
}

func valToString(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	}

	return fmt.Sprintf("%v", v)
}

// GetFloat returns a tag value as float64 and an error if one occurred.
// ErrTagNotFound will be returned if the key can't be found.
func (t Tags) GetFloat(tag string) (float64, error) {
	v, found := t[tag]
	if !found || v == nil {
		return 0, ErrTagNotFound
	}

	switch v := v.(type) {
	case string:
		return valToFloat64(v)
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	}

	str := fmt.Sprintf("%v", v)
	return valToFloat64(str)
}

func valToFloat64(str string) (float64, error) {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("float64 parsing error (%v): %w", str, err)
	}
	return val, nil
}

// FormatTags is a json data structure to represent format tags
// Deprecated, use the Tags of TagList instead
type FormatTags struct {
	MajorBrand       string `json:"major_brand"`
	MinorVersion     string `json:"minor_version"`
	CompatibleBrands string `json:"compatible_brands"`
	CreationTime     string `json:"creation_time"`
}

func (f *FormatTags) setFrom(tags Tags) {
	f.MajorBrand, _ = tags.GetString("major_brand")
	f.MinorVersion, _ = tags.GetString("minor_version")
	f.CompatibleBrands, _ = tags.GetString("compatible_brands")
	f.CreationTime, _ = tags.GetString("creation_time")
}

// StreamTags is a json data structure to represent stream tags
// Deprecated, use the Tags of TagList instead
type StreamTags struct {
	Rotate       int    `json:"rotate,string,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	Language     string `json:"language,omitempty"`
	Title        string `json:"title,omitempty"`
	Encoder      string `json:"encoder,omitempty"`
	Location     string `json:"location,omitempty"`
}

func (s *StreamTags) setFrom(tags Tags) {
	rotate, _ := tags.GetInt("rotate")
	s.Rotate = int(rotate)

	s.CreationTime, _ = tags.GetString("creation_time")
	s.Language, _ = tags.GetString("language")
	s.Title, _ = tags.GetString("title")
	s.Encoder, _ = tags.GetString("encoder")
	s.Location, _ = tags.GetString("location")
}
