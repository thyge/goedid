package eedid

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type DisplayID struct {
	Version                 byte
	VariableDataBlockLength byte
	DisplayPrimaryUsecase   DisplayPrimaryUsecase
	NumberOfExtensions      byte
	DataBlocks              []byte
	DetailedTimingDataBlock DetailedTimingDataBlock
}

type DisplayPrimaryUsecase byte

const (
	ExtensionSection DisplayPrimaryUsecase = 0x00
	TestStructure    DisplayPrimaryUsecase = 0x01
	Generic          DisplayPrimaryUsecase = 0x02
	Television       DisplayPrimaryUsecase = 0x03
	Productivity     DisplayPrimaryUsecase = 0x04
	Gaming           DisplayPrimaryUsecase = 0x05
	Presentation     DisplayPrimaryUsecase = 0x06
	VirtualReality   DisplayPrimaryUsecase = 0x07
	AugmentedReality DisplayPrimaryUsecase = 0x08
)

func (dpu DisplayPrimaryUsecase) String() string {
	if dpu > 8 {
		return "Unknown"
	}
	displayPrimaryUsecaseStrings := [...]string{
		"Extension section",
		"Test structure",
		"Generic",
		"Television",
		"Productivity",
		"Gaming",
		"Presentation",
		"Virtual reality",
		"Augmented reality",
	}
	return displayPrimaryUsecaseStrings[dpu]
}

func (s DisplayPrimaryUsecase) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(s.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type DataBlockType byte

const (
	ProductIdentification         DataBlockType = 0x00
	DisplayParameters             DataBlockType = 0x01
	ColorCharacteristics          DataBlockType = 0x02
	TypeITiming                   DataBlockType = 0x03
	TypeIITiming                  DataBlockType = 0x04
	TypeIIITiming                 DataBlockType = 0x05
	TypeIVTiming                  DataBlockType = 0x06
	VESATimingStandard            DataBlockType = 0x07
	CEATimingStandard             DataBlockType = 0x08
	VideoTimingRange              DataBlockType = 0x09
	ProductSerialNumber           DataBlockType = 0x0A
	GeneralPurposeASCIIString     DataBlockType = 0x0B
	DisplayDeviceData             DataBlockType = 0x0C
	InterfacePowerSequencing      DataBlockType = 0x0D
	TransferCharacteristics       DataBlockType = 0x0E
	DisplayInterfaceData          DataBlockType = 0x0F
	StereoDisplayInterface        DataBlockType = 0x10
	TypeVTiming                   DataBlockType = 0x11
	TypeVITiming                  DataBlockType = 0x13
	ProductIdentification2        DataBlockType = 0x20
	DisplayParameters2            DataBlockType = 0x21
	TypeVIIDetailedTiming         DataBlockType = 0x22
	TypeVIIIEnumeratedTiming      DataBlockType = 0x23
	TypeIXFormulaTiming           DataBlockType = 0x24
	DynamicVideoTimingRangeLimits DataBlockType = 0x25
	DisplayInterfaceFeatures      DataBlockType = 0x26
	StereoDisplayInterface2       DataBlockType = 0x27
	TiledDisplayTopology          DataBlockType = 0x28
	ContainerID                   DataBlockType = 0x29
)

func (dbt DataBlockType) String() string {
	dataBlockTypeStrings := [...]string{
		"Product Identification",             // 0x00
		"Display Parameters",                 // 0x01
		"Color Characteristics",              // 0x02
		"Type I Timing - Detailed",           // 0x03
		"Type II Timing - Detailed",          // 0x04
		"Type III Timing - Short",            // 0x05
		"Type IV Timing - DMT ID Code",       // 0x06
		"VESA Timing Standard",               // 0x07
		"CEA Timing Standard",                // 0x08
		"Video Timing Range",                 // 0x09
		"Product Serial Number",              // 0x0A
		"General Purpose ASCII String",       // 0x0B
		"Display Device Data",                // 0x0C
		"Interface Power Sequencing",         // 0x0D
		"Transfer Characteristics",           // 0x0E
		"Display Interface Data",             // 0x0F
		"Stereo Display Interface",           // 0x10
		"Type V Timing - Short",              // 0x11
		"UNKNOWN",                            // 0x12
		"Type VI Timing - Detailed",          // 0x13
		"UNKNOWN",                            // 0x14
		"UNKNOWN",                            // 0x15
		"UNKNOWN",                            // 0x16
		"UNKNOWN",                            // 0x17
		"UNKNOWN",                            // 0x18
		"UNKNOWN",                            // 0x19
		"UNKNOWN",                            // 0x1A
		"UNKNOWN",                            // 0x1B
		"UNKNOWN",                            // 0x1C
		"UNKNOWN",                            // 0x1D
		"UNKNOWN",                            // 0x1E
		"UNKNOWN",                            // 0x1F
		"Product Identification",             // 0x20
		"Display Parameters",                 // 0x21
		"Type VII - Detailed Timing",         // 0x22
		"Type VIII - Enumerated Timing Code", // 0x23
		"Type IX - Formula-based Timing",     // 0x24
		"Dynamic Video Timing Range Limits",  // 0x25
		"Display Interface Features",         // 0x26
		"Stereo Display Interface",           // 0x27
		"Tiled Display Topology",             // 0x28
		"ContainerID",                        // 0x29
	}
	switch dbt {
	case 0xD0:
		return "Interface Power Sequencing"
	case 0x7F:
		return "Vendor specific"
	case 0x81:
		return "CTA DisplayID"
	default:
		if dbt > 0x29 {
			return "UNKNOWN"
		} else {
			return dataBlockTypeStrings[dbt]
		}
	}
}

func (s DataBlockType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(s.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type DetailedTimingDataBlock struct {
	DataBlockType   DataBlockType `json:",test"`
	BlockRevision   byte
	NumberOfBytes   byte
	DetailedTimings []DetailedTimingDescriptor
}

//https://github.com/pkorobeinikov/golang-example/blob/master/math/gcd.go
// GCDEuclidean calculates GCD by Euclidian algorithm.
func GCDEuclidean(a, b int) int {
	for a != b {
		if a > b {
			a -= b
		} else {
			b -= a
		}
	}
	return a
}

func DecodeDTDVII(dtdBytes []byte) DetailedTimingDescriptor {
	dtd := new(DetailedTimingDescriptor)
	dtd.PixelClockKHz = (uint32(dtdBytes[2]) << 16) | (uint32(dtdBytes[1]) << 8) | uint32(dtdBytes[0])

	dtd.HorizontalActive = binary.LittleEndian.Uint16(dtdBytes[4:])
	dtd.HorizontalBlanking = binary.LittleEndian.Uint16(dtdBytes[6:])
	dtd.HorizontalFrontPorch = binary.LittleEndian.Uint16(dtdBytes[8:]) & 0x7FFF
	dtd.HorizontalSyncPolarity = binary.LittleEndian.Uint16(dtdBytes[8:])&0x8000 > 1
	dtd.HorizontalSyncPulseWidth = binary.LittleEndian.Uint16(dtdBytes[10:])

	dtd.VerticalActive = binary.LittleEndian.Uint16(dtdBytes[12:])
	dtd.VerticalBlanking = binary.LittleEndian.Uint16(dtdBytes[14:])
	dtd.VerticalFrontPorch = binary.LittleEndian.Uint16(dtdBytes[16:]) & 0x7FFF
	dtd.VerticalSyncPolarity = binary.LittleEndian.Uint16(dtdBytes[16:])&0x8000 > 1
	dtd.VerticalSyncPulseWidth = binary.LittleEndian.Uint16(dtdBytes[18:])

	// Do this last because we may need hAct and vAct for calculation
	switch dtdBytes[3] & 0x7 {
	case 0:
		dtd.AspectRatio = "1:1"
	case 1:
		dtd.AspectRatio = "5:4"
	case 2:
		dtd.AspectRatio = "4:3"
	case 3:
		dtd.AspectRatio = "15:9"
	case 4:
		dtd.AspectRatio = "16:9"
	case 5:
		dtd.AspectRatio = "16:10"
	case 6:
		dtd.AspectRatio = "64:27"
	case 7:
		dtd.AspectRatio = "256:135"
	case 8:
		gcd := GCDEuclidean(int(dtd.HorizontalActive+1), int(dtd.VerticalActive+1))
		hor := int(dtd.HorizontalActive+1) / gcd
		vert := int(dtd.VerticalActive+1) / gcd
		dtd.AspectRatio = string(hor) + ":" + string(vert)
	default:
		dtd.AspectRatio = "RESERVED"
	}
	// StereoMode is wrong for displayID
	// TODO: fix this
	dtd.Stereo = StereoMode(dtdBytes[3] & 0x61)
	dtd.Interlaced = (dtdBytes[3] & 0x10) > 0

	return *dtd
}

func (dtd *DetailedTimingDescriptor) EncodeDTDVII() [18]byte {
	// TODO: Finish this encode
	var returnBytes [18]byte
	returnBytes[0] = byte(dtd.PixelClockKHz & 0xFF)
	returnBytes[1] = byte(dtd.PixelClockKHz >> 8)

	returnBytes[2] = byte(dtd.HorizontalActive & 0xFF)
	returnBytes[3] = byte(dtd.HorizontalBlanking & 0xFF)
	returnBytes[5] = byte(dtd.VerticalActive & 0xFF)
	returnBytes[6] = byte(dtd.VerticalBlanking & 0xFF)

	returnBytes[4] = byte(dtd.HorizontalBlanking&0xF000>>12 | dtd.HorizontalActive&0x0F00>>8)
	returnBytes[7] = byte(dtd.VerticalBlanking&0xF000>>12 | dtd.VerticalActive&0x0F00>>8)

	returnBytes[8] = byte(dtd.HorizontalFrontPorch)
	returnBytes[9] = byte(dtd.HorizontalSyncPulseWidth)

	returnBytes[10] = byte(dtd.VerticalSyncPulseWidth&0xF) | byte((dtd.VerticalFrontPorch&0xF)<<4)

	returnBytes[11] = byte(dtd.VerticalSyncPulseWidth & 0x30 >> 4)
	returnBytes[11] = returnBytes[11] | byte(dtd.VerticalFrontPorch&0x30)>>2
	returnBytes[11] = returnBytes[11] | byte(dtd.HorizontalSyncPulseWidth&0x30)
	returnBytes[11] = returnBytes[11] | byte(dtd.HorizontalFrontPorch&0x30)<<2

	returnBytes[12] = byte(dtd.HorizontalImageSize)
	returnBytes[13] = byte(dtd.VerticalImageSize)
	returnBytes[14] = byte(dtd.HorizontalImageSize>>4) | byte(dtd.VerticalImageSize>>8)

	returnBytes[15] = dtd.HorizontalBorder
	returnBytes[16] = dtd.VerticalBorder

	if dtd.Interlaced {
		returnBytes[17] = returnBytes[17] | 0x80
	}
	returnBytes[17] |= byte(dtd.Stereo)
	// force Digital sync., separate
	if dtd.VerticalSyncPolarity == SYNC_ON_POSITIVE {
		returnBytes[17] |= 0x4
	}
	if dtd.HorizontalSyncPolarity == SYNC_ON_POSITIVE {
		returnBytes[17] |= 0x2
	}
	return dtd.data
}

func GetDetailedTimingDataBlock(dtdblockBytes []byte) DetailedTimingDataBlock {
	dttdb := new(DetailedTimingDataBlock)
	// Header
	dttdb.DataBlockType = DataBlockType(dtdblockBytes[0])
	dttdb.BlockRevision = dtdblockBytes[1] & 0x7
	dttdb.NumberOfBytes = dtdblockBytes[2]
	// timing blocks
	for i := byte(3); i < dttdb.NumberOfBytes; i += 20 {
		// each block is 20
		dtd := DecodeDTDVII(dtdblockBytes[i : i+20])
		dttdb.DetailedTimings = append(dttdb.DetailedTimings, dtd)
	}
	return *dttdb
}

func DecodeDisplayID(didBytes []byte) DisplayID {
	did := new(DisplayID)
	did.Version = didBytes[1]
	did.VariableDataBlockLength = didBytes[2]
	did.DisplayPrimaryUsecase = DisplayPrimaryUsecase(didBytes[3])
	did.NumberOfExtensions = didBytes[4]

	for i := byte(5); i < did.VariableDataBlockLength; i++ {
		blockType := DataBlockType(didBytes[i])

		// Catch data blocks without number of bytes set
		numBytes := didBytes[i+2]
		if numBytes < 1 {
			fmt.Println("Block does not have byte count", blockType)
		} else {
			fmt.Println(blockType)
		}
		switch blockType {
		case ProductIdentification:
			// fmt.Println(blockType)
		case DisplayInterfaceFeatures:
			iffBlock := GetDisplayInterfaceFeatures(didBytes[i : i+3+numBytes])
			fmt.Println(iffBlock)
		case TypeITiming:
			dtviitb := GetDetailedTimingDataBlock(didBytes[i : i+3+numBytes])

			did.DetailedTimingDataBlock = dtviitb
			i += numBytes + 2
		}
	}

	return *did
}

type DisplayInterfaceFeaturesBlock struct {
	DataBlockType      DataBlockType
	BlockRevision      byte
	NumberOfBytes      byte
	RGBBitDepth        byte
	YCbCr444BitDepth   byte
	YCbCr422BitDepth   byte
	YCbCr420BitDepth   byte
	YCbCr420MinPixRate byte
	AudioCapability    byte
	ColourSpace        byte
	EOTFBytes          byte
}

func GetDisplayInterfaceFeatures(iffBytes []byte) DisplayInterfaceFeaturesBlock {
	iffdb := new(DisplayInterfaceFeaturesBlock)
	// Header
	iffdb.DataBlockType = DataBlockType(iffBytes[0])
	iffdb.BlockRevision = iffBytes[1] & 0x7
	iffdb.NumberOfBytes = iffBytes[2]
	// 9 bytes
	iffdb.RGBBitDepth = iffBytes[3]
	iffdb.YCbCr444BitDepth = iffBytes[4]
	iffdb.YCbCr422BitDepth = iffBytes[5]
	iffdb.YCbCr420BitDepth = iffBytes[6]
	iffdb.YCbCr420MinPixRate = iffBytes[7]
	iffdb.AudioCapability = iffBytes[8]
	iffdb.ColourSpace = iffBytes[9]
	iffdb.EOTFBytes = iffBytes[11]
	if int(iffdb.EOTFBytes) > len(iffBytes)+3 {
		// Parse EOTF here
	}
	return *iffdb
}
