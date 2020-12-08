package edid

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type Edid14 struct {
	fixedHeader       [8]byte
	ManufacturerID    ManufacturerID //encode decode
	MonitorName       string
	ProductCode       uint16
	SerialNumber      string
	weekOfManufacture byte
	yearOfManufacture byte
	EdidVersion       byte
	EdidRevision      byte
	// BasicDisplayParameters
	BitDepth               BitDepth
	digitalInput           bool
	VideoInterface         VideoInterface
	HorizontalScreenSizeCM byte
	VerticalScreenSizeCM   byte
	DisplayGamma           float32
	DPMS                   DPMS
	DisplayType            DisplayType
	// ChromaticityCoordinates
	// EstablishedTimings
	EstablishedTimings []EstablishedTiming
	//StandardTimings
	StandardTimings           []StandardTiming //encode decode
	DetailedTimingDescriptors []DetailedTimingDescriptor
	NumberOfExtensions        byte
	Checksum                  byte //calculate
	data                      [128]byte
}

func ParseEdid14(edidBytes []byte) (Edid14, error) {
	edid := new(Edid14)
	for i := 0; i < len(edidBytes); i++ {
		edid.data[i] = edidBytes[i]
	}
	eBuffer := bytes.NewBuffer(edidBytes)

	binary.Read(eBuffer, binary.LittleEndian, &edid.fixedHeader)

	var RawManufacturingID [2]byte
	binary.Read(eBuffer, binary.LittleEndian, &RawManufacturingID)
	edid.ManufacturerID = ManufacturerID(DecodeFiveBitASCII(&RawManufacturingID))

	binary.Read(eBuffer, binary.LittleEndian, &edid.ProductCode)
	var serialNumberBytes uint32
	binary.Read(eBuffer, binary.LittleEndian, &serialNumberBytes)
	edid.SerialNumber = fmt.Sprint(serialNumberBytes)
	binary.Read(eBuffer, binary.LittleEndian, &edid.weekOfManufacture)
	binary.Read(eBuffer, binary.LittleEndian, &edid.yearOfManufacture)
	binary.Read(eBuffer, binary.LittleEndian, &edid.EdidVersion)
	binary.Read(eBuffer, binary.LittleEndian, &edid.EdidRevision)

	// Basic display parameters
	var basicDispParams byte
	binary.Read(eBuffer, binary.LittleEndian, &basicDispParams)
	if basicDispParams&0x80 > 0 {
		// digital input
		edid.digitalInput = true
		edid.VideoInterface = VideoInterface(basicDispParams & 0x7)
		edid.BitDepth = BitDepth(basicDispParams & 0x70 >> 4)
	} else {
		edid.digitalInput = false
		// analog input
		// TODO
	}
	binary.Read(eBuffer, binary.LittleEndian, &edid.HorizontalScreenSizeCM)
	binary.Read(eBuffer, binary.LittleEndian, &edid.VerticalScreenSizeCM)

	var displayGamma byte
	binary.Read(eBuffer, binary.LittleEndian, &displayGamma)
	edid.DisplayGamma = (float32(displayGamma) / 100) + 1

	var featureMap byte
	binary.Read(eBuffer, binary.LittleEndian, &featureMap)
	if featureMap&0x80 > 0 {
		edid.DPMS = DPMS_STANDBY_SUPPORTED
	} else if featureMap&0x40 > 0 {
		edid.DPMS = DPMS_SUSPEND_SUPPORTED
	} else if featureMap&0x20 > 0 {
		edid.DPMS = DPMS_ACTIVE_OFF
	} else {
		edid.DPMS = DPMS_NOT_SUPPORTED
	}
	modeBits := featureMap & 0x18 >> 3
	if edid.digitalInput == true {
		edid.DisplayType = DisplayType(modeBits)
	}
	// TODO: analog support
	//TODO: Bit 0-2 feature map

	//Chromaticity coordinates.
	var chromaticityCoordinates [10]byte
	binary.Read(eBuffer, binary.LittleEndian, &chromaticityCoordinates)

	// Established timings
	var establishedTimingsBytes [3]byte
	counter := 7
	binary.Read(eBuffer, binary.LittleEndian, &establishedTimingsBytes)
	for ETByteNumber := 0; ETByteNumber < len(establishedTimingsBytes); ETByteNumber++ {
		for ETBitFlag := byte(0x80); ETBitFlag != 0; ETBitFlag >>= 1 {
			if establishedTimingsBytes[ETByteNumber]&ETBitFlag > 0 {
				var estTime uint32

				estTime = estTime | uint32(ETBitFlag)<<(8*ETByteNumber)
				edid.EstablishedTimings = append(edid.EstablishedTimings, EstablishedTiming(estTime))
				counter--
			}
		}
		counter = 7
	}

	// StandardTimings
	var standardTimingBytes [16]byte
	binary.Read(eBuffer, binary.LittleEndian, &standardTimingBytes)
	for i := 0; i < len(standardTimingBytes); i += 2 {
		if standardTimingBytes[i] == 0 {
			break
		}
		// 0x1 0x1 means the rest is padded
		if standardTimingBytes[i] == 0x1 && standardTimingBytes[i+1] == 0x1 {
			break
		}
		st := new(StandardTiming)
		st.HorizontalActive = (uint(standardTimingBytes[i]) + 31) * 8
		st.AspectRatio = AspectRatio(standardTimingBytes[i+1] >> 6)
		// TODO: Support EDID 1.3 - 1:1 as 0
		st.RefreshRate = standardTimingBytes[i+1]&0x3F + 60
		edid.StandardTimings = append(edid.StandardTimings, *st)
	}

	// Detailed timing descriptors
	// up to 4 x DTDs
	for i := 0; i < 4; i++ {
		var dtdbytes [18]byte
		binary.Read(eBuffer, binary.LittleEndian, &dtdbytes)
		// Identify what kind of descriptor
		// if first 2 bytes / pixel clock is 0 then parse as Display Descriptor
		descriptorHeader := (uint16(dtdbytes[1]) << 8) | uint16(dtdbytes[0])
		if descriptorHeader == 0 {
			switch dtdbytes[3] {
			case 0xFF:
				//Display serial number (ASCII text)
				// This field takes presidence over [12:15]
				edid.SerialNumber = strings.TrimSpace(string(dtdbytes[5:]))
			case 0xFE:
				//Unspecified text (ASCII text)
				fmt.Println("Unspecified text (ASCII text)")
			case 0xFD:
				//Monitor range limits
				fmt.Println("Monitor range limits")
			case 0xFC:
				//Display name (ASCII text)
				edid.MonitorName = strings.TrimSpace(string(dtdbytes[5:]))
			case 0xFB:
				fmt.Println("Additional white point data. 2× 5-byte descriptors, padded with 0A 20 20")
			case 0xFA:
				fmt.Println("Additional standard timing identifiers. 6× 2-byte descriptors, padded with 0A")
			case 0xF9:
				fmt.Println("Display Color Management (DCM)")
			case 0xF8:
				fmt.Println("CVT 3-Byte Timing Codes")
			case 0xF7:
				fmt.Println("Additional standard timing 3")
			case 0x10:
				// Dummy identifier.
			default:
				fmt.Println("Manufacturer reserved descriptors")
			}
		} else {
			dtd := DecodeDTD(&dtdbytes)
			edid.DetailedTimingDescriptors = append(edid.DetailedTimingDescriptors, dtd)
		}
	}
	binary.Read(eBuffer, binary.LittleEndian, &edid.NumberOfExtensions)
	binary.Read(eBuffer, binary.LittleEndian, &edid.Checksum)

	return *edid, nil
}

func (e *Edid14) Encode() [128]byte {
	var returnBytes [128]byte
	// Fixed Header
	returnBytes[0] = 0x00
	returnBytes[1] = 0xFF
	returnBytes[2] = 0xFF
	returnBytes[3] = 0xFF
	returnBytes[4] = 0xFF
	returnBytes[5] = 0xFF
	returnBytes[6] = 0xFF
	returnBytes[7] = 0x00
	manuID := e.ManufacturerID.Encode()
	returnBytes[8] = manuID[0]
	returnBytes[9] = manuID[1]
	returnBytes[10] = byte(e.ProductCode)
	returnBytes[11] = byte(e.ProductCode >> 8)
	// don't use EDID serial number
	// returnBytes[12]
	// returnBytes[13]
	// returnBytes[14]
	// returnBytes[15]
	// returnBytes[16] // week manu
	// returnBytes[17] // year manu
	returnBytes[18] = e.EdidVersion
	returnBytes[19] = e.EdidRevision

	// basic display params
	returnBytes[20] = 0x80 // set digital
	returnBytes[20] = returnBytes[20] | byte(e.VideoInterface)
	returnBytes[20] = returnBytes[20] | byte(e.BitDepth)<<4
	returnBytes[21] = e.HorizontalScreenSizeCM
	returnBytes[22] = e.HorizontalScreenSizeCM
	// TODO: This is not correct
	returnBytes[23] = byte((e.DisplayGamma - 1) * 100)
	returnBytes[24] = byte(e.DPMS) << 5
	returnBytes[24] = returnBytes[24] | byte(e.DisplayType)<<3
	// Chromaticity coordinates.
	// returnBytes[25:34]

	// Established timing
	var combinedEstTimings uint32
	for _, eTiming := range e.EstablishedTimings {
		combinedEstTimings = combinedEstTimings | uint32(eTiming)
	}
	returnBytes[35] = byte(combinedEstTimings)
	returnBytes[36] = byte(combinedEstTimings >> 8)
	returnBytes[37] = byte(combinedEstTimings >> 16)
	// Standard timing information
	// 38–53
	var stBytes []byte
	for _, sTiming := range e.StandardTimings {
		timingBytes := sTiming.Encode()
		for i := 0; i < len(timingBytes); i++ {
			stBytes = append(stBytes, timingBytes[i])
		}
	}
	for i := 0; i < 16; i++ {
		if i < len(stBytes) {
			returnBytes[38+i] = stBytes[i]
		} else {
			// Unused fields are filled with 01 01 hex
			returnBytes[38+i] = 0x1
		}
	}

	// DTD Section
	numDTDs := len(e.DetailedTimingDescriptors)
	for i := 0; i < 4; i++ {
		// ensure we add the timings first, then add other info
		if numDTDs > 0 {
			// get the dtd byte array
			timingBytes := e.DetailedTimingDescriptors[i].Encode()
			// add the bytes to the return bytes
			for dtdBytes := 0; dtdBytes < len(timingBytes); dtdBytes++ {
				// dtd starts in byte 54 + i of 4 descriptors
				returnBytes[dtdBytes+54+(i*18)] = timingBytes[dtdBytes]
			}
			numDTDs--
			continue
		} else {
			if len(e.SerialNumber) > 0 {
				// descriptor header is 0 0
				returnBytes[54+(i*18)+3] = 0xFF
				snBytes := []byte(e.SerialNumber)
				// sn can be max 15 bytes + header
				for snb := 0; snb < 13; snb++ {
					// -1 because strings are null terminated in go
					if snb < len(snBytes) {
						returnBytes[snb+54+(i*18)+5] = snBytes[snb]
						continue
					}
					if snb == len(snBytes) {
						// line feed terminate on +1
						returnBytes[snb+54+(i*18)+5] = 0x0A
						continue
					}
					if snb > len(snBytes) {
						// pad with space
						returnBytes[snb+54+(i*18)+5] = 0x20
					}
				}
				// null serialnumber to not add in next loop
				e.SerialNumber = ""
				continue
			}
			if len(e.MonitorName) > 0 {
				// descriptor header is 0 0
				returnBytes[54+(i*18)+3] = 0xFC
				snBytes := []byte(e.MonitorName)
				// sn can be max 15 bytes + header
				for snb := 0; snb < 13; snb++ {
					if snb < len(snBytes) {
						returnBytes[snb+54+(i*18)+5] = snBytes[snb]
						continue
					}
					if snb == len(snBytes) {
						// line feed terminate on +1
						returnBytes[snb+54+(i*18)+5] = 0x0A
						continue
					}
					if snb > len(snBytes) {
						// pad with space
						returnBytes[snb+54+(i*18)+5] = 0x20
					}
				}
				// for snb := 0; (snb < len(snBytes)) && (snb < 13); snb++ {
				// 	// insert bytes on +1 to accomodate header
				// 	returnBytes[snb+54+(i*18)+5] = snBytes[snb]
				// }

				// null serialnumber to not add in next loop
				e.MonitorName = ""
				continue
			}
			// add dummy descriptors to pad out
			returnBytes[54+(i*18)+3] = 0x10
		}
	}
	var dtdBytes []byte
	for _, dtdTiming := range e.DetailedTimingDescriptors {
		timingBytes := dtdTiming.Encode()
		for i := 0; i < len(timingBytes); i++ {
			dtdBytes = append(dtdBytes, timingBytes[i])
		}
	}
	// for i := 0; i < 4 && i < 127; i += 18 {
	// 	if i < len(stBytes) {
	// 		for j := 0; j < 18; j++ {
	// 			returnBytes[54+j*i] = stBytes[i]
	// 		}
	// 	}
	// }
	returnBytes[126] = e.NumberOfExtensions
	returnBytes[127] = MakeEDIDChecksum(&returnBytes)
	return returnBytes
}

type DisplayType byte

const (
	RGB444              DisplayType = 0
	RGB444_YCRCB444     DisplayType = 1
	RGB444_YCRCB422     DisplayType = 2
	RGB444_YCRCB444_422 DisplayType = 3
)

var dusplayTypeLookup = map[DisplayType]string{
	RGB444:              "RGB 4:4:4",
	RGB444_YCRCB444:     "RGB 4:4:4 + YCrCb 4:4:4",
	RGB444_YCRCB422:     "RGB 4:4:4 + YCrCb 4:2:2",
	RGB444_YCRCB444_422: "RGB 4:4:4 + YCrCb 4:4:4 + YCrCb 4:2:2",
}

func (d DisplayType) String() string {
	return dusplayTypeLookup[d]
}
func (d DisplayType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(d.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type BitDepth byte

const (
	BPP_UNDEFINED BitDepth = 0
	BPP6          BitDepth = 1
	BPP8          BitDepth = 2
	BPP10         BitDepth = 3
	BPP12         BitDepth = 4
	BPP14         BitDepth = 5
	BPP16         BitDepth = 6
)

func (d BitDepth) String() string {
	switch d {
	default:
		return "UNDEFINED"
	case BPP6:
		return "6"
	case BPP8:
		return "8"
	case BPP10:
		return "10"
	case BPP12:
		return "12"
	case BPP14:
		return "14"
	case BPP16:
		return "16"
	}
}
func (d BitDepth) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(d.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type DPMS byte

const (
	DPMS_ACTIVE_OFF        DPMS = 0
	DPMS_SUSPEND_SUPPORTED DPMS = 1
	DPMS_STANDBY_SUPPORTED DPMS = 2
	DPMS_NOT_SUPPORTED     DPMS = 3
)

var dpmsLookup = map[DPMS]string{
	DPMS_ACTIVE_OFF:        "Active OFF",
	DPMS_SUSPEND_SUPPORTED: "Suspend Supported",
	DPMS_STANDBY_SUPPORTED: "Standby Supported",
	DPMS_NOT_SUPPORTED:     "Not Supported",
}

func (d DPMS) String() string {
	return dpmsLookup[d]
}
func (d DPMS) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(d.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type VideoInterface byte

const (
	InterfaceHDMIa       VideoInterface = 2
	InterfaceHDMIb       VideoInterface = 3
	InterfaceMDDI        VideoInterface = 4
	InterfaceDisplayPort VideoInterface = 5
)

var videoInterfaceLookup = map[VideoInterface]string{
	InterfaceHDMIa:       "HDMIa",
	InterfaceHDMIb:       "HDMIb",
	InterfaceMDDI:        "MDDI",
	InterfaceDisplayPort: "DisplayPort",
}

func (v VideoInterface) String() string {
	return videoInterfaceLookup[v]
}
func (v VideoInterface) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(v.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type ManufacturerID string

func (s ManufacturerID) String() string {
	if pnp, ok := pnpLookup[string(s)]; ok {
		return pnp.Company
	} else {
		return string(s)
	}
}
func (s ManufacturerID) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(s.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func (s ManufacturerID) Encode() [2]byte {
	var retBytes [2]byte
	retBytes[0] = (s[0] - 0x40) << 2
	retBytes[0] = retBytes[0] | (s[1]-0x40)>>3
	retBytes[1] = (s[1] - 0x40) << 5
	retBytes[1] = retBytes[1] | (s[2] - 0x40)
	return retBytes
}

type PNPID struct {
	ID      string
	Company string
	Date    string
}

type StandardTiming struct {
	HorizontalActive uint
	VerticalActive   uint
	RefreshRate      byte
	AspectRatio      AspectRatio
}

func (st *StandardTiming) Encode() [2]byte {
	var returnBytes [2]byte
	returnBytes[0] = (byte(st.HorizontalActive / 8)) - 31
	returnBytes[1] = st.RefreshRate - 60
	returnBytes[1] = returnBytes[1] | byte(st.AspectRatio)<<6
	return returnBytes
}

type AspectRatio byte

const (
	AR_16_10 AspectRatio = 0
	AR_4_3   AspectRatio = 1
	AR_5_4   AspectRatio = 2
	AR_16_9  AspectRatio = 3
)

func (ar AspectRatio) String() string {
	switch ar {
	case 0:
		return "16:10"
	case 1:
		return "4:3"
	case 2:
		return "5:4"
	case 3:
		return "16:9"
	default:
		return "NA"
	}
}

func (ar AspectRatio) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(ar.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type DetailedTimingDescriptor struct {
	PixelClockKHz            uint32
	HorizontalActive         uint16 // Zero base in VII
	HorizontalBlanking       uint16 // Zero base in VII
	HorizontalFrontPorch     uint16 // Zero base in VII
	HorizontalSyncPulseWidth uint16 // Zero base in VII
	VerticalActive           uint16 // Zero base in VII
	VerticalBlanking         uint16 // Zero base in VII
	VerticalFrontPorch       uint16 // Zero base in VII
	VerticalSyncPulseWidth   uint16 // Zero base in VII
	HorizontalImageSize      uint16
	VerticalImageSize        uint16
	HorizontalBorder         byte
	VerticalBorder           byte
	Interlaced               bool
	Stereo                   StereoMode
	SyncType                 string
	HorizontalSyncPolarity   SyncPolarity
	VerticalSyncPolarity     SyncPolarity
	data                     [18]byte
	AspectRatio              string // VII Detailed Timing Descriptor
	PreferredTiming          bool   // VII Detailed Timing Descriptor
	// Data from CVT Generator
	VerticalTotal       int
	VerticalBackPorch   int
	HorizontalTotal     int
	HorizontalBackPorch int
	VertRefreshRate     float64
	HorRefreshRate      float64
	RawPixelClock       float64
	ACT_FRAME_RATE      float64
}

func DecodeDTD(edidBytes *[18]byte) DetailedTimingDescriptor {
	d := new(DetailedTimingDescriptor)
	d.PixelClockKHz = (uint32(edidBytes[1]) << 8) | uint32(edidBytes[0])
	d.HorizontalActive = (uint16(edidBytes[4]&0xF0) << 4) | uint16(edidBytes[2])
	d.HorizontalBlanking = (uint16(edidBytes[4]&0xF) << 8) | uint16(edidBytes[3])
	d.VerticalActive = (uint16(edidBytes[7]&0xF0) << 4) | uint16(edidBytes[5])
	d.VerticalBlanking = (uint16(edidBytes[7]&0xF) << 8) | uint16(edidBytes[6])

	d.HorizontalFrontPorch = ((uint16(edidBytes[11]) & 0xC0) << 2) | uint16(edidBytes[8])
	d.HorizontalSyncPulseWidth = ((uint16(edidBytes[11]) & 0x30) << 4) | uint16(edidBytes[9])

	d.VerticalFrontPorch = ((uint16(edidBytes[11]) & 0xC) << 2) | ((uint16(edidBytes[10]) & 0xF0) >> 4)
	d.VerticalSyncPulseWidth = ((uint16(edidBytes[11]) & 0x3) << 4) | (uint16(edidBytes[10]) & 0xF)

	d.HorizontalImageSize = ((uint16(edidBytes[14]) & 0xF0) << 4) | uint16(edidBytes[12])
	d.VerticalImageSize = ((uint16(edidBytes[14]) & 0xF) << 8) | uint16(edidBytes[13])
	d.HorizontalBorder = edidBytes[15]
	d.VerticalBorder = edidBytes[16]
	d.Interlaced = (edidBytes[17] & 0x80) > 0
	syncBits := (edidBytes[17] & 0x18) >> 3
	d.VerticalSyncPolarity = ((edidBytes[17] & 0x4) >> 2) > 0
	d.HorizontalSyncPolarity = ((edidBytes[17] & 0x2) >> 1) > 0

	switch syncBits {
	case 0:
		d.SyncType = "Analog composite"
	case 1:
		d.SyncType = "Bipolar analog composite"
	case 2:
		d.SyncType = "Digital composite (on HSync)"
	case 3:
		d.SyncType = "Digital separate"
	}
	stereoMode := (edidBytes[17] & 0x61)
	d.Stereo = StereoMode(stereoMode)

	return *d
}

func (dtd *DetailedTimingDescriptor) Encode() [18]byte {
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
	return returnBytes
}

func (d *DetailedTimingDescriptor) GetVerticalHz() float64 {
	return float64(d.RawPixelClock / float64(d.VerticalTotal*d.HorizontalTotal))
}
func (d *DetailedTimingDescriptor) GetHorizontalHz() float64 {
	return float64(d.RawPixelClock / float64(d.HorizontalTotal))
}

type SyncPolarity bool

const (
	SYNC_ON_POSITIVE SyncPolarity = true
	SYNC_ON_NEGATIVE SyncPolarity = false
)

var syncPolarityLookup = map[SyncPolarity]string{
	SYNC_ON_POSITIVE: "Positive",
	SYNC_ON_NEGATIVE: "Negative",
}

func (sp SyncPolarity) String() string {
	return syncPolarityLookup[sp]
}

func (sp SyncPolarity) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(sp.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

type StereoMode byte

const (
	Stereo_None                   StereoMode = 0x00
	Stereo_Sequential_Right       StereoMode = 0x20
	Stereo_Sequential_Left        StereoMode = 0x40
	Stereo_2way_Interleaved_Right StereoMode = 0x21
	Stereo_2way_Interleaved_Left  StereoMode = 0x41
	Stereo_4way_Interleaved       StereoMode = 0x60
	Stereo_SideBySide_Interleaved StereoMode = 0x61
)

func (sm StereoMode) String() string {
	switch sm {
	case Stereo_None:
		return "No Stereo"
	case Stereo_Sequential_Right:
		return "field sequential, right during stereo sync"
	case Stereo_Sequential_Left:
		return "field sequential, left during stereo sync"
	case Stereo_2way_Interleaved_Right:
		return "2-way interleaved, right image on even lines"
	case Stereo_2way_Interleaved_Left:
		return "2-way interleaved, left image on even lines"
	case Stereo_4way_Interleaved:
		return "4-way interleaved"
	case Stereo_SideBySide_Interleaved:
		return "side-by-side interleaved"
	default:
		return "RESERVED"
	}
}

func (sm StereoMode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(sm.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func DecodeFiveBitASCII(fivebit *[2]byte) string {
	stringbytes := []byte("   ")
	stringbytes[0] = ((fivebit[0] & 0x7C) >> 2) + 0x40
	stringbytes[1] = ((fivebit[0] & 0x03) << 3) + ((fivebit[1] & 0xE0) >> 5) + 0x40
	stringbytes[2] = (fivebit[1] & 0x1F) + 0x40
	return string(stringbytes)
}

type EstablishedTiming uint32

const (
	ET_800_600_60 EstablishedTiming = 0x1
)

var estTimingLookup = map[uint32]string{
	0x1:     "800×600 @ 60 Hz",
	0x2:     "800×600 @ 56 Hz",
	0x4:     "640×480 @ 75 Hz",
	0x8:     "640×480 @ 72 Hz",
	0x10:    "640×480 @ 67 Hz (Apple Macintosh II)",
	0x20:    "640×480 @ 60 Hz (VGA)",
	0x40:    "720×400 @ 88 Hz (XGA)",
	0x80:    "720×400 @ 70 Hz (VGA)",
	0x100:   "1280×1024 @ 75 Hz",
	0x200:   "1024×768 @ 75 Hz",
	0x400:   "1024×768 @ 72 Hz",
	0x800:   "1024×768 @ 60 Hz",
	0x1000:  "1024×768 @ 87 Hz, interlaced (1024×768i)",
	0x2000:  "832×624 @ 75 Hz (Apple Macintosh II)",
	0x4000:  "800×600 @ 75 Hz",
	0x8000:  "800×600 @ 72 Hz",
	0x10000: "1152x870 @ 75 Hz (Apple Macintosh II)",
}

func (et EstablishedTiming) String() string {
	return estTimingLookup[uint32(et)]
}
func (et EstablishedTiming) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(et.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func MakeEDIDChecksum(checkBytes *[128]byte) byte {
	var checkSum byte
	for i := 0; i < len(checkBytes); i++ {
		checkSum += checkBytes[i]
	}
	return 0xFF - checkSum + 1
}
