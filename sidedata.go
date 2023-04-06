package ffprobe

import "errors"

const sideDataTypeField = "side_data_type"

// ErrSideDataNotFound is a sentinel error used when a queried side data does not exist
var ErrSideDataNotFound = errors.New("side data not found")

// SideDataList is a json data structure to represent stream side data
type SideDataList []Tags

func (s *SideDataList) GetSideData(sideDataType string) (Tags, error) {
	for _, sideData := range *s {
		dataType, err := sideData.GetString(sideDataTypeField)
		if err != nil {
			continue
		}
		if dataType == sideDataType {
			return sideData, nil
		}
	}
	return Tags{}, ErrSideDataNotFound
}
