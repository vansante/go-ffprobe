package ffprobe

import (
	"encoding/json"
	"errors"
)

var (
	ErrSideDataNotFound       = errors.New("side data not found")
	ErrSideDataUnexpectedType = errors.New("unexpected data type")
)

// All names and structures of side data types got from
// https://github.com/FFmpeg/FFmpeg/blob/4ab1184fae88bd47b9d195ac8224853c6f4e94cf/libavcodec/avpacket.c#L268
// https://github.com/FFmpeg/FFmpeg/blob/master/fftools/ffprobe.c#L2291
const (
	SideDataTypeUnknown                  = "unknown"
	SideDataTypeDisplayMatrix            = "Display Matrix"
	SideDataTypeStereo3D                 = "Stereo 3D"
	SideDataTypeSphericalMapping         = "Spherical Mapping"
	SideDataTypeSkipSamples              = "Skip Samples"
	SideDataTypeMasteringDisplayMetadata = "Mastering display metadata"
	SideDataTypeContentLightLevel        = "Content light level metadata"
)

type SideDataBase struct {
	Type string `json:"side_data_type"`
}

// SideDataDisplayMatrix represents the display matrix side data.
type SideDataDisplayMatrix struct {
	SideDataBase
	Data     string `json:"displaymatrix"`
	Rotation int    `json:"rotation"`
}

// SideDataStereo3D represents the stereo 3D side data.
type SideDataStereo3D struct {
	SideDataBase
	Type     string `json:"type"`
	Inverted bool   `json:"inverted"`
}

// SideDataSphericalMapping represents the spherical mapping side data.
type SideDataSphericalMapping struct {
	SideDataBase
	Projection  string `json:"projection"`
	Padding     int    `json:"padding,omitempty"`
	BoundLeft   int    `json:"bound_left,omitempty"`
	BoundTop    int    `json:"bound_top,omitempty"`
	BoundRight  int    `json:"bound_right,omitempty"`
	BoundBottom int    `json:"bound_bottom,omitempty"`
	Yaw         int    `json:"yaw,omitempty"`
	Pitch       int    `json:"pitch,omitempty"`
	Roll        int    `json:"roll,omitempty"`
}

// SideDataSkipSamples represents the skip samples side data.
type SideDataSkipSamples struct {
	SideDataBase
	SkipSamples    int `json:"skip_samples"`
	DiscardPadding int `json:"discard_padding"`
	SkipReason     int `json:"skip_reason"`
	DiscardReason  int `json:"discard_reason"`
}

// SideDataMasteringDisplayMetadata represents the mastering display metadata side data.
type SideDataMasteringDisplayMetadata struct {
	SideDataBase
	RedX         int `json:"red_x,omitempty"`
	RedY         int `json:"red_y,omitempty"`
	GreenX       int `json:"green_x,omitempty"`
	GreenY       int `json:"green_y,omitempty"`
	BlueX        int `json:"blue_x,omitempty"`
	BlueY        int `json:"blue_y,omitempty"`
	WhitePointX  int `json:"white_point_x,omitempty"`
	WhitePointY  int `json:"white_point_y,omitempty"`
	MinLuminance int `json:"min_luminance,omitempty"`
	MaxLuminance int `json:"max_luminance,omitempty"`
}

// SideDataContentLightLevel represents the content light level side data.
type SideDataContentLightLevel struct {
	SideDataBase
	MaxContent int `json:"max_content,omitempty"`
	MaxAverage int `json:"max_average,omitempty"`
}

// SideDataUnknown represents an unknown side data.
type SideDataUnknown Tags

// SideData represents a side data packet.
type SideData struct {
	SideDataBase
	Data interface{} `json:"-"`
}

func (sd *SideData) UnmarshalJSON(b []byte) error {
	type Alias SideData
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(sd),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	switch sd.Type {
	case SideDataTypeDisplayMatrix:
		sd.Data = &SideDataDisplayMatrix{}
	case SideDataTypeStereo3D:
		sd.Data = new(SideDataStereo3D)
	case SideDataTypeSphericalMapping:
		sd.Data = new(SideDataSphericalMapping)
	case SideDataTypeSkipSamples:
		sd.Data = new(SideDataSkipSamples)
	case SideDataTypeMasteringDisplayMetadata:
		sd.Data = new(SideDataMasteringDisplayMetadata)
	case SideDataTypeContentLightLevel:
		sd.Data = new(SideDataContentLightLevel)
	default:
		sd.Data = new(SideDataUnknown)
	}

	return json.Unmarshal(b, sd.Data)
}

func (sd *SideData) MarshalJSON() ([]byte, error) {
	return json.Marshal(sd.Data)
}

// SideDataList represents a list of side data packets.
type SideDataList []SideData

// UnmarshalJSON for SideDataList
func (s *SideDataList) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	for _, r := range raw {
		var sd SideData
		if err := json.Unmarshal(r, &sd); err != nil {
			return err
		}
		*s = append(*s, sd)
	}

	return nil
}

// FindUnknownSideData searches for SideData of type SideDataUnknown in the SideDataList.
// If such SideData is found, it is returned, otherwise, an error is returned
// indicating that the SideData of type SideDataUnknown was not found or the found
// SideData was of an unexpected type.
func (s SideDataList) FindUnknownSideData(sideDataType string) (*SideDataUnknown, error) {
	data, found := s.findSideDataByName(sideDataType)
	if !found {
		return nil, ErrSideDataNotFound
	}

	unknownSideData, ok := data.(*SideDataUnknown)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}

	return unknownSideData, nil
}

// FindSideData searches for SideData by its type in the SideDataList.
// If SideData of the specified type is found, it is returned, otherwise,
// an error is returned indicating that the SideData of the specified type was not found.
func (s SideDataList) FindSideData(sideDataType string) (interface{}, error) {
	data, found := s.findSideDataByName(sideDataType)
	if !found {
		return nil, ErrSideDataNotFound
	}

	return data, nil
}

// GetDisplayMatrix retrieves the DisplayMatrix from the SideData. If the DisplayMatrix is not found or
// the SideData is of the wrong type, an error is returned.
func (s SideDataList) GetDisplayMatrix() (*SideDataDisplayMatrix, error) {
	data, found := s.findSideDataByName(SideDataTypeDisplayMatrix)
	if !found {
		return nil, ErrSideDataNotFound
	}
	displayMatrix, ok := data.(*SideDataDisplayMatrix)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}
	return displayMatrix, nil
}

// GetStereo3D retrieves the Stereo3D data from the SideData. If the Stereo3D data is not found or
// the SideData is of the wrong type, an error is returned.
func (s SideDataList) GetStereo3D() (*SideDataStereo3D, error) {
	data, found := s.findSideDataByName(SideDataTypeStereo3D)
	if !found {
		return nil, ErrSideDataNotFound
	}
	stereo3D, ok := data.(*SideDataStereo3D)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}
	return stereo3D, nil
}

// GetSphericalMapping retrieves the SphericalMapping data from the SideData. If the SphericalMapping data is not found or
// the SideData is of the wrong type, an error is returned.
func (s SideDataList) GetSphericalMapping() (*SideDataSphericalMapping, error) {
	data, found := s.findSideDataByName(SideDataTypeSphericalMapping)
	if !found {
		return nil, ErrSideDataNotFound
	}
	sphericalMapping, ok := data.(*SideDataSphericalMapping)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}
	return sphericalMapping, nil
}

// GetSkipSamples retrieves the SkipSamples data from the SideData. If the SkipSamples data is not found or
// the SideData is of the wrong type, an error is returned.
func (s SideDataList) GetSkipSamples() (*SideDataSkipSamples, error) {
	data, found := s.findSideDataByName(SideDataTypeSkipSamples)
	if !found {
		return nil, ErrSideDataNotFound
	}
	skipSamples, ok := data.(*SideDataSkipSamples)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}
	return skipSamples, nil
}

// GetMasteringDisplayMetadata retrieves the MasteringDisplayMetadata from the SideData. If the MasteringDisplayMetadata is not found or
// the SideData is of the wrong type, an error is returned.
func (s SideDataList) GetMasteringDisplayMetadata() (*SideDataMasteringDisplayMetadata, error) {
	data, found := s.findSideDataByName(SideDataTypeMasteringDisplayMetadata)
	if !found {
		return nil, ErrSideDataNotFound
	}
	masteringDisplayMetadata, ok := data.(*SideDataMasteringDisplayMetadata)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}
	return masteringDisplayMetadata, nil
}

// GetContentLightLevel retrieves the ContentLightLevel from the SideData. If the ContentLightLevel is not found or
// the SideData is of the wrong type, an error is returned.
func (s SideDataList) GetContentLightLevel() (*SideDataContentLightLevel, error) {
	data, found := s.findSideDataByName(SideDataTypeContentLightLevel)
	if !found {
		return nil, ErrSideDataNotFound
	}
	contentLightLevel, ok := data.(*SideDataContentLightLevel)
	if !ok {
		return nil, ErrSideDataUnexpectedType
	}
	return contentLightLevel, nil
}

func (s SideDataList) findSideDataByName(sideDataType string) (interface{}, bool) {
	for _, sd := range s {
		if sd.Type == sideDataType {
			return sd.Data, true
		}
	}
	return nil, false
}
