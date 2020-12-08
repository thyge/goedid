package edid

import (
	"bytes"
	"fmt"
)

// Short Video Reference 1 refers to the most-preferred Video Format,
// while higher numbered SVRs (2, 3, through N) refer to Video Formatsin order of decreasing preference.
// All Video Formats referred to in the VFPDB are preferred over
// Video Formats for which preference expressed elsewhere in the EDID.
// However, for Video Formats not referred to in the VFPDB, preferences expressed elsewhere shall be used.

// Short Video References (SVRs) refer to Video Formats via VICs and/or DTD indices.
// The Source shall interpret SVR codes according to the following pseudo code:

// If SVR = 0 then
// 	Reserved
// Elseif SVR >=1 and SVR <=127 then
// 	Interpret as a VIC
// Elseif SVR =128 then
// 	Reserved
// Elseif SVR >=129 and SVR <=144 then
// 	Interpret as the Kth DTD in the EDID, where K = SVR – 128 (for K=1 to 16)
// Elseif SVR >=145 and SVR <=192 then
// 	Reserved
// Elseif SVR >=193 and SVR <=253 then
// 	Interpret as a VIC
// Elseif SVR >=254 and SVR <=255 then
// 	Reserved
// End if

type CEA struct {
	ExtensionTag              byte
	Revision                  byte
	DTDStart                  byte
	Underscan                 bool
	BasicAudio                bool
	YCbCr444                  bool
	YCbCr422                  bool
	NumberOfNativeDTD         byte
	DataBlocks                []CEADataBlock
	DetailedTimingDescriptors []DetailedTimingDescriptor
	// 18 byte descriptors onwards with 00 padding
}

func (cea *CEA) DecodeDTDInfo(dtdinfo byte) {
	if dtdinfo&0x80 > 0 {
		cea.Underscan = true
	}
	if dtdinfo&0x40 > 0 {
		cea.BasicAudio = true
	}
	if dtdinfo&0x20 > 0 {
		cea.YCbCr444 = true
	}
	if dtdinfo&0x10 > 0 {
		cea.YCbCr422 = true
	}
	cea.NumberOfNativeDTD = dtdinfo & 0xF
}
func (cea *CEA) EncodeDTDInfo() byte {
	var dtdByte byte
	// TODO: How to find out if DTD is native?
	// newNumberNativeDTD := len(cea.DetailedTimingDescriptors)
	dtdByte += byte(cea.NumberOfNativeDTD)

	if cea.Underscan {
		dtdByte = dtdByte | 0x80
	}
	if cea.BasicAudio {
		dtdByte = dtdByte | 0x40
	}
	if cea.YCbCr444 {
		dtdByte = dtdByte | 0x20
	}
	if cea.YCbCr422 {
		dtdByte = dtdByte | 0x10
	}

	return dtdByte
}
func (cea *CEA) GetBytes() [128]byte {
	var ceaArray [128]byte
	ceaArray[0] = cea.ExtensionTag
	ceaArray[1] = cea.Revision
	// Export DTD Info Byte
	ceaArray[3] = cea.EncodeDTDInfo()
	// Export Data Blocks
	ArrayCounter := byte(4)
	for _, dataBlock := range cea.DataBlocks {
		dataBlockBytes := dataBlock.GetBytes()
		for i := 0; i < len(dataBlockBytes); i++ {
			ceaArray[ArrayCounter] = dataBlockBytes[i]
			ArrayCounter++
		}
	}
	ceaArray[2] = ArrayCounter // ArrayCounter = DTD Start byte here
	for _, dtd := range cea.DetailedTimingDescriptors {
		dataBlockBytes := dtd.Encode()
		for i := 0; i < len(dataBlockBytes); i++ {
			ceaArray[ArrayCounter] = dataBlockBytes[i]
			ArrayCounter++
		}
	}
	// pad the rest of the structure
	for i := ArrayCounter; i < 126; i++ {
		ceaArray[i] = 0
	}

	// checksum
	ceaArray[127] = MakeEDIDChecksum(&ceaArray)
	return ceaArray
}
func (cea *CEA) MakeChecksum(ceaBytes *[128]byte) byte {
	var checkSum byte
	for i := 0; i < len(ceaBytes); i++ {
		checkSum += ceaBytes[i]
	}
	return 0xFF - checkSum + 1
}

type CEADataBlock struct {
	Type          string
	numberOfBytes int
	data          []byte
	Block         []interface{}
}

func (db *CEADataBlock) GetBytes() []byte {
	return db.data
}

// this is used with 4:2:0 capability map
var ceaResoutionsGlobal []CEAResulution

func DecodeDataBlocks(datablockbytes []byte) []CEADataBlock {
	var returnblocks []CEADataBlock
	for i := 0; i < len(datablockbytes); {
		// header data block length
		thisBlock := new(CEADataBlock)
		thisBlock.numberOfBytes = int(datablockbytes[i] & 0x1F)
		if thisBlock.numberOfBytes == 0 {
			break
		}
		// header data block type
		// ofset 1 to skip header, numberOfBytes is bytes after header so +1
		blocktype := datablockbytes[i] >> 5
		if blocktype == 1 {
			thisBlock.Type = "audio"
			audiBlock := DecodeAudio(datablockbytes[i+1 : i+thisBlock.numberOfBytes+1])
			thisBlock.Block = append(thisBlock.Block, audiBlock)
		} else if blocktype == 2 {
			thisBlock.Type = "video"
			cearesoutions := DecodeCEAResulutions(datablockbytes[i+1 : i+thisBlock.numberOfBytes+1])
			ceaResoutionsGlobal = cearesoutions
			thisBlock.Block = append(thisBlock.Block, cearesoutions)
		} else if blocktype == 3 {
			thisBlock.Type = "vendor specific"
			vsdbBlock := DecodeVSDB(datablockbytes[i+1 : i+thisBlock.numberOfBytes+1])
			thisBlock.Block = append(thisBlock.Block, vsdbBlock)
		} else if blocktype == 4 {
			thisBlock.Type = "speaker allocation"
			speakerBlock := DecodeSpeaker(datablockbytes[i+1 : i+thisBlock.numberOfBytes+1])
			thisBlock.Block = append(thisBlock.Block, speakerBlock)
		} else if blocktype == 5 {
			thisBlock.Type = "VESA Display Transfer Characteristic"
		} else if blocktype == 7 {
			thisBlock.Type = "Extended Type"
			extendecBlock := DecodeExtendedType(datablockbytes[i+1 : i+thisBlock.numberOfBytes+1])
			thisBlock.Block = append(thisBlock.Block, extendecBlock)
		} else {
			thisBlock.Type = "reserved"
		}

		// Total number of bytes including header
		for j := i; j < i+thisBlock.numberOfBytes+1; j++ {
			thisBlock.data = append(thisBlock.data, datablockbytes[j])
		}
		returnblocks = append(returnblocks, *thisBlock)
		i = i + thisBlock.numberOfBytes + 1
	}
	return returnblocks
}

type ExtendedType byte

func (et ExtendedType) String() string {
	return extendedTypeLookup[et]
}
func (et ExtendedType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(et.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

const (
	VideoCapabilityDB             ExtendedType = 0
	VendorSpecificVideoDB         ExtendedType = 1
	VESADisplayDeviceDB           ExtendedType = 2
	VESAVideoTimingBlockExtension ExtendedType = 3
	HDMIVideoDB                   ExtendedType = 4
	ColorimetryDB                 ExtendedType = 5
	HDRStaticMetadataDB           ExtendedType = 6
	// reserved video related blocks 8-12
	VideoFormatPreferenceDB ExtendedType = 13
	YCBCR420VideoDB         ExtendedType = 14
	YCBCR420CapabilityMap   ExtendedType = 15
	CTAMiscellaneousAudioDB ExtendedType = 16
	VendorSpecificAudioDB   ExtendedType = 17
	HDMIAudioDB             ExtendedType = 18
	RoomConfigurationDB     ExtendedType = 19
	SpeakerLocationDB       ExtendedType = 20
	// reserved audio related blocks 21-31
	InfoFrameDB ExtendedType = 32
)

var extendedTypeLookup = map[ExtendedType]string{
	VideoCapabilityDB:             "Video Capability Data Block",
	VendorSpecificVideoDB:         "Vendor-Specific Video Data Block",
	VESADisplayDeviceDB:           "VESA Display Device Data Block [100]",
	VESAVideoTimingBlockExtension: "VESA Video Timing Block Extension",
	HDMIVideoDB:                   "Reserved for HDMI Video Data Block",
	ColorimetryDB:                 "Colorimetry Data Block",
	HDRStaticMetadataDB:           "HDR Static Metadata Data Block",
	VideoFormatPreferenceDB:       "Video Format Preference Data Block",
	YCBCR420VideoDB:               "YCBCR 4:2:0 Video Data Block",
	YCBCR420CapabilityMap:         "YCBCR 4:2:0 Capability Map Data Block",
	CTAMiscellaneousAudioDB:       "Reserved for CTA Miscellaneous Audio Fields",
	VendorSpecificAudioDB:         "Vendor-Specific Audio Data Block",
	HDMIAudioDB:                   "Reserved for HDMI Audio Data Block",
	RoomConfigurationDB:           "Room Configuration Data Block",
	SpeakerLocationDB:             "Speaker Location Data Block",
	InfoFrameDB:                   "InfoFrame Data Block (includes one or more Short InfoFrame Descriptors)",
}

type ExtendedTypeBlock struct {
	Type  ExtendedType
	Block interface{}
}

func DecodeExtendedType(eBytes []byte) ExtendedTypeBlock {
	et := new(ExtendedTypeBlock)
	et.Type = ExtendedType(eBytes[0])
	switch et.Type {
	case VideoCapabilityDB:
		bl := DecodeVideoCapability(eBytes[1:])
		et.Block = bl
	case VendorSpecificVideoDB:
	case VESADisplayDeviceDB:
	case VESAVideoTimingBlockExtension:
	case HDMIVideoDB:
	case ColorimetryDB:
		bl := DecodeColorimetry(eBytes[1:])
		et.Block = bl
	case HDRStaticMetadataDB:
		bl := DecodeHDRStatic(eBytes[1:])
		et.Block = bl
	case VideoFormatPreferenceDB:
	case YCBCR420VideoDB:
	case YCBCR420CapabilityMap:
		bl := DecodeYCBCR420Capability(eBytes[1:])
		et.Block = bl
	}
	return *et
}

type YCBCR420CapabilityBlock struct {
	// indicates exactly which SVDs also support YCBCR 4:2:0 sampling
	Resolutions []CEAResulution
}

func DecodeYCBCR420Capability(cmdb []byte) YCBCR420CapabilityBlock {
	hdr := new(YCBCR420CapabilityBlock)
	svdCount := 0
	for i := 0; i < len(cmdb); i++ {
		for bit := byte(0x80); bit != 0; bit >>= 1 {
			if cmdb[i]&bit > 1 {
				if svdCount < len(ceaResoutionsGlobal) {
					hdr.Resolutions = append(hdr.Resolutions, ceaResoutionsGlobal[svdCount])
				} else {
					fmt.Println("420Capability set to a higher number than availible SVR")
				}

			}
			svdCount++
		}
	}
	return *hdr
}

type HDRStaticMetdataBlock struct {
	HybridLogGamma             bool
	SMPTE2084                  bool
	TraditionalGammaHDR        bool
	TraditionalGammaSDR        bool
	StaticMetadataType1        bool
	ContentMaxLuminance        byte
	ContentMaxAverageLuminance byte
	ContentMinLuminance        byte
}

func DecodeHDRStatic(c []byte) HDRStaticMetdataBlock {
	hdr := new(HDRStaticMetdataBlock)
	if c[0]&0x08 > 1 {
		hdr.HybridLogGamma = true
	}
	if c[0]&0x04 > 1 {
		hdr.SMPTE2084 = true
	}
	if c[0]&0x02 > 1 {
		hdr.TraditionalGammaHDR = true
	}
	if c[0]&0x01 > 1 {
		hdr.TraditionalGammaSDR = true
	}
	if len(c) < 2 {
		return *hdr
	}
	if c[1]&0x01 > 1 {
		hdr.StaticMetadataType1 = true
	}
	if len(c) < 3 {
		return *hdr
	}
	hdr.ContentMaxLuminance = c[2]
	if len(c) < 4 {
		return *hdr
	}
	hdr.ContentMaxAverageLuminance = c[3]
	if len(c) < 5 {
		return *hdr
	}
	hdr.ContentMinLuminance = c[4]
	if len(c) < 6 {
		return *hdr
	}
	return *hdr
}

type VideoCapabilityBlock struct {
	QY    bool
	QS    bool
	S_PT1 bool
	S_PT0 bool
	S_IT1 bool
	S_IT0 bool
	S_CE1 bool
	S_CE0 bool
}

func DecodeVideoCapability(c []byte) VideoCapabilityBlock {
	vc := new(VideoCapabilityBlock)
	if c[0]&0x80 > 1 {
		vc.QY = true
	}
	if c[0]&0x40 > 1 {
		vc.QS = true
	}
	if c[0]&0x20 > 1 {
		vc.S_PT1 = true
	}
	if c[0]&0x10 > 1 {
		vc.S_PT0 = true
	}
	if c[0]&0x08 > 1 {
		vc.S_IT1 = true
	}
	if c[0]&0x04 > 1 {
		vc.S_IT0 = true
	}
	if c[0]&0x02 > 1 {
		vc.S_CE1 = true
	}
	if c[0]&0x01 > 1 {
		vc.S_CE0 = true
	}
	return *vc
}

type ColorimetryBlock struct {
	BT2020_RGB   bool
	BT2020_YCC   bool
	BT2020_cYCC  bool
	OpRGB        bool
	OpYCC_601601 bool
	SYCC_601     bool
	XvYCC_709    bool
	XvYCC_601    bool
	DCI_P3       bool
	F46          bool
	F45          bool
	F44          bool
	MD3          bool
	MD2          bool
	MD1          bool
	MD0          bool
}

func DecodeColorimetry(c []byte) ColorimetryBlock {
	cb := new(ColorimetryBlock)
	if c[0]&0x80 > 1 {
		cb.BT2020_RGB = true
	}
	if c[0]&0x40 > 1 {
		cb.BT2020_YCC = true
	}
	if c[0]&0x20 > 1 {
		cb.BT2020_cYCC = true
	}
	if c[0]&0x10 > 1 {
		cb.OpRGB = true
	}
	if c[0]&0x08 > 1 {
		cb.OpYCC_601601 = true
	}
	if c[0]&0x04 > 1 {
		cb.SYCC_601 = true
	}
	if c[0]&0x02 > 1 {
		cb.XvYCC_709 = true
	}
	if c[0]&0x01 > 1 {
		cb.XvYCC_601 = true
	}
	// byte 2
	if c[1]&0x80 > 1 {
		cb.DCI_P3 = true
	}
	if c[1]&0x40 > 1 {
		cb.F46 = true
	}
	if c[1]&0x20 > 1 {
		cb.F45 = true
	}
	if c[1]&0x10 > 1 {
		cb.F44 = true
	}
	if c[1]&0x08 > 1 {
		cb.MD3 = true
	}
	if c[1]&0x04 > 1 {
		cb.MD2 = true
	}
	if c[1]&0x02 > 1 {
		cb.MD1 = true
	}
	if c[1]&0x01 > 1 {
		cb.MD0 = true
	}
	return *cb
}

type ShortAudioDescriptor struct {
	AudioFormat      string
	NumberOfChannels byte
	SamplingKHz      []float32
	BitDepth         []int
}

func DecodeAudio(audioBytes []byte) ShortAudioDescriptor {
	audioBlock := new(ShortAudioDescriptor)
	audioBlock.NumberOfChannels = audioBytes[0] & 0x7

	formatByte := audioBytes[0] & 0x78
	formatByte = formatByte >> 3
	switch formatByte {
	case 1:
		audioBlock.AudioFormat = "LPCM"
	case 2:
		audioBlock.AudioFormat = "AC-3"
	case 3:
		audioBlock.AudioFormat = "MPEG-1"
	case 4:
		audioBlock.AudioFormat = "MP3"
	case 5:
		audioBlock.AudioFormat = "MPEG-2"
	case 6:
		audioBlock.AudioFormat = "AAC"
	case 7:
		audioBlock.AudioFormat = "DTS"
	case 8:
		audioBlock.AudioFormat = "ATRAC"
	case 9:
		audioBlock.AudioFormat = "1-bit audio"
	case 10:
		audioBlock.AudioFormat = "DD+"
	case 11:
		audioBlock.AudioFormat = "DTS-HD"
	case 12:
		audioBlock.AudioFormat = "MLP/Dolby TrueHD"
	case 13:
		audioBlock.AudioFormat = "DST Audio"
	case 14:
		audioBlock.AudioFormat = "Microsoft WMA Pro"
	default:
		audioBlock.AudioFormat = "reserved"
	}
	samplingByte := audioBytes[1]
	if samplingByte&0x1 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 32)
	}
	if samplingByte&0x2 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 44.1)
	}
	if samplingByte&0x4 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 48)
	}
	if samplingByte&0x8 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 88)
	}
	if samplingByte&0x10 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 96)
	}
	if samplingByte&0x20 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 176)
	}
	if samplingByte&0x40 > 0 {
		audioBlock.SamplingKHz = append(audioBlock.SamplingKHz, 192)
	}
	bitrateByte := audioBytes[2] & 0x7
	if audioBlock.AudioFormat == "LPCM" {
		if bitrateByte&0x1 > 0 {
			audioBlock.BitDepth = append(audioBlock.BitDepth, 16)
		}
		if bitrateByte&0x2 > 0 {
			audioBlock.BitDepth = append(audioBlock.BitDepth, 20)
		}
		if bitrateByte&0x4 > 0 {
			audioBlock.BitDepth = append(audioBlock.BitDepth, 24)
		}
	} else {
		audioBlock.BitDepth = append(audioBlock.BitDepth, int(bitrateByte))
	}
	return *audioBlock
}

type SpeakerAllocation struct {
	Location string
}

func DecodeSpeaker(speakerBytes []byte) SpeakerAllocation {
	// TODO flesh out speaker allocation for all permutations
	speakerByte := speakerBytes[0]
	speakerBlock := new(SpeakerAllocation)
	if speakerByte&0x1 > 0 {
		speakerBlock.Location = "Front left and right"
	}
	if speakerByte&0x2 > 0 {
		speakerBlock.Location = "Low-frequency effects (LFE)"
	}
	if speakerByte&0x4 > 0 {
		speakerBlock.Location = "Front center"
	}
	if speakerByte&0x8 > 0 {
		speakerBlock.Location = "Rear left and right"
	}
	if speakerByte&0x10 > 0 {
		speakerBlock.Location = "Rear center"
	}
	if speakerByte&0x20 > 0 {
		speakerBlock.Location = "Front left and right center"
	}
	if speakerByte&0x40 > 0 {
		speakerBlock.Location = "Rear left and right center"
	}
	return *speakerBlock
}

type VSDBType byte

func (v VSDBType) String() string {
	return vsdbLookup[v]
}
func (v VSDBType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(v.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

const (
	TYPE_HDMI1_4            VSDBType = iota
	TYPE_HDMI2_0            VSDBType = iota
	TYPE_DolbyVision        VSDBType = iota
	TYPE_HDR10              VSDBType = iota
	TYPE_SpecializedMonitor VSDBType = iota
)

var vsdbLookup = map[VSDBType]string{
	TYPE_HDMI1_4:            "HDMI 1.4",
	TYPE_HDMI2_0:            "HDMI 2.0",
	TYPE_DolbyVision:        "Dolby Vision",
	TYPE_HDR10:              "HDR10",
	TYPE_SpecializedMonitor: "Specialized Monitor",
}

type VendorSpecificDataBlock struct {
	Type  VSDBType
	Block interface{}
	data  []byte
}

func DecodeVSDB(videobytes []byte) VendorSpecificDataBlock {
	var HDMI1_4 = [3]byte{0x00, 0x0C, 0x03}
	var HDMI2_0 = [3]byte{0xC4, 0x5D, 0xD8}
	var HDMIDolbyVision = [3]byte{0x00, 0xD0, 0x46}
	var HDMIHDR10 = [3]byte{0x00, 0x0C, 0x03}
	var SpecializedMonitor = [3]byte{0x5C, 0x12, 0xCA}

	vsdb := new(VendorSpecificDataBlock)
	var IEEEIdentifyer [3]byte
	IEEEIdentifyer[0] = videobytes[2]
	IEEEIdentifyer[1] = videobytes[1]
	IEEEIdentifyer[2] = videobytes[0]

	switch IEEEIdentifyer {
	case HDMI1_4:
		vsdb.Type = TYPE_HDMI1_4
		vsdb.Block = DecodeVSDB_HDMI1_4(videobytes)
	case HDMI2_0:
		vsdb.Type = TYPE_HDMI2_0
		vsdb.Block = DecodeVSDB_HDMI2_0(videobytes)
	case HDMIDolbyVision:
		vsdb.Type = TYPE_DolbyVision
	case HDMIHDR10:
		vsdb.Type = TYPE_HDR10
	case SpecializedMonitor:
		vsdb.Type = TYPE_SpecializedMonitor
		vsdb.Block = DecodeVSDB_SpecializedMonitor(videobytes)
	}

	for i := 0; i < len(videobytes); i++ {
		vsdb.data = append(vsdb.data, videobytes[i])
	}

	return *vsdb
}

type HDMI_Address struct {
	A byte
	B byte
	C byte
	D byte
}

type VSDB_HDMI1_4 struct {
	IEEE                 [3]byte
	Address              HDMI_Address
	DVIDualLinkOperation bool
	BitDepth16           bool
	BitDepth12           bool
	BitDepth10           bool
	DeepColour444        bool
	MaxTMDS              byte
	// Latency_Fields_Present   bool
	// I_Latency_Fields_Present bool
	// HDMI_Video_present       bool
	// CNC0                     bool
	// CNC1                     bool
	// CNC2                     bool
	// CNC3                     bool
	// VideoLatency             byte
	// AudioLatency             byte
	// InterlacedVideoLatency   byte
	// InterlacedAudioLatency   byte
	//     Extended HDMI video details:
	// VICs

}

func DecodeVSDB_HDMI1_4(dataBytes []byte) VSDB_HDMI1_4 {
	db := new(VSDB_HDMI1_4)
	db.Address.A = dataBytes[3] >> 4
	db.Address.B = dataBytes[3] & 0xF
	db.Address.C = dataBytes[4] >> 4
	db.Address.D = dataBytes[4] & 0xF
	if len(dataBytes) < 6 {
		return *db
	}
	if dataBytes[5]&0x40 > 1 {
		db.BitDepth16 = true
	}
	if dataBytes[5]&0x20 > 1 {
		db.BitDepth12 = true
	}
	if dataBytes[5]&0x10 > 1 {
		db.BitDepth10 = true
	}
	if dataBytes[5]&0x08 > 1 {
		db.DeepColour444 = true
	}
	if dataBytes[5]&0x01 > 1 {
		db.DVIDualLinkOperation = true
	}
	return *db
}

type MaxFixedRateLink byte

const (
	FRL_NotSupported MaxFixedRateLink = 0
	FRL_3G_3Lanes    MaxFixedRateLink = 1
	FRL_6G_3Lanes    MaxFixedRateLink = 2
	FRL_6G_4Lanes    MaxFixedRateLink = 3
	FRL_8G_4Lanes    MaxFixedRateLink = 4
	FRL_10G_4Lanes   MaxFixedRateLink = 5
	FRL_12G_4Lanes   MaxFixedRateLink = 6
)

type VSDB_HDMI2_0 struct {
	// Table 10-6:
	IEEE                 [3]byte
	Version              byte
	MaxTMDS              byte
	OSD_3D_Disparity     bool
	Dual_View            bool
	Independent_View     bool
	LTE_340Mcsc_scramble bool
	CCBPCI               bool
	RR_Capable           bool
	SCDC_Present         bool
	DC_30bit_420         bool
	DC_36bit_420         bool
	DC_48bit_420         bool
	MaxFixedRateLink     MaxFixedRateLink
	FAPA_start_location  bool
	ALLM                 bool
	FVA                  bool
	CNMVRR               bool
	CinemaVRR            bool
	MDelta               bool
	VRRMIN               byte
	VRRMAX               uint16
	DSC_10bpc            bool
	DSC_12bpc            bool
	DSC_16bpc            bool
	DSC_All_bpp          bool
	DSC_Native_420       bool
	DSC_1p2              bool
	DSC_MaxSlices        byte
	DSC_Max_FRL_Rate     byte
	DSC_TotalChunkKBytes byte
}

func DecodeVSDB_HDMI2_0(dataBytes []byte) VSDB_HDMI2_0 {
	db := new(VSDB_HDMI2_0)
	db.Version = dataBytes[3]
	db.MaxTMDS = dataBytes[4]
	if len(dataBytes) < 6 {
		return *db
	}
	if dataBytes[5]&0x80 > 1 {
		db.SCDC_Present = true
	}
	if dataBytes[5]&0x40 > 1 {
		db.RR_Capable = true
	}
	if dataBytes[5]&0x10 > 1 {
		db.CCBPCI = true
	}
	if dataBytes[5]&0x08 > 1 {
		db.LTE_340Mcsc_scramble = true
	}
	if dataBytes[5]&0x04 > 1 {
		db.Independent_View = true
	}
	if dataBytes[5]&0x02 > 1 {
		db.Dual_View = true
	}
	if dataBytes[5]&0x01 > 1 {
		db.OSD_3D_Disparity = true
	}
	if len(dataBytes) < 7 {
		return *db
	}
	db.MaxFixedRateLink = MaxFixedRateLink(dataBytes[6] >> 4)
	if dataBytes[6]&0x04 > 1 {
		db.DC_48bit_420 = true
	}
	if dataBytes[6]&0x02 > 1 {
		db.DC_36bit_420 = true
	}
	if dataBytes[6]&0x01 > 1 {
		db.DC_30bit_420 = true
	}
	return *db
}

type VSDB_DolbyVision struct {
}
type VSDB_HDMIHDR10 struct {
}

func (vsdb *VendorSpecificDataBlock) EncodeVSDB() []byte {
	var returnBytes []byte
	// add header
	dbType := byte(3 << 5)
	blockLength := byte(len(vsdb.data))
	returnBytes = append(returnBytes, 0x0)
	returnBytes[0] = blockLength | dbType
	// add bytes
	for i := 0; i < len(vsdb.data); i++ {
		returnBytes = append(returnBytes, vsdb.data[i])
	}
	return returnBytes
}

type VSDB_SpecializedMonitor struct {
	Name                         string
	Version                      byte
	VersionDescription           string
	WindowsMRExperience          bool
	ThirdPartyMRExperience       bool
	SpecializedDisplay           bool
	DesktopDisplay               bool
	DesktopThirdParty            bool
	PrimaryUseCase               string
	TestEquipment                bool
	GenericDisplay               bool
	TelevisionDisplay            bool
	DesktopProductivityDisplay   bool
	DesktopGamingDisplay         bool
	PresentationDisplay          bool
	VirtualRealityHeadsets       bool
	AugmentedReality             bool
	VideoWallDisplay             bool
	MedicalImagingDisplay        bool
	DedicatedGamingDisplay       bool
	DedicatedVideoMonitorDisplay bool
	AccessoryDisplay             bool
	ContainerID                  [16]byte
}

func DecodeVSDB_SpecializedMonitor(datablock []byte) VSDB_SpecializedMonitor {

	vsdbSM := new(VSDB_SpecializedMonitor)
	vsdbSM.Name = "Specialized Monitor DataBlock"

	vsdbSM.Version = datablock[3]
	// give verbose description
	switch vsdbSM.Version {
	case 1:
		vsdbSM.VersionDescription = "HMD (VR/AR) display devices that will be used by the Windows Mixed Reality experience"
		vsdbSM.WindowsMRExperience = true
	case 2:
		vsdbSM.VersionDescription = "HMD (VR/AR) display devices that will be used by third-party compositors (other than the Windows Mixed Reality experience)"
		vsdbSM.ThirdPartyMRExperience = true
	case 3:
		vsdbSM.VersionDescription = "Specialized display devices that are not HMDs"
		vsdbSM.SpecializedDisplay = true
	default:
		vsdbSM.VersionDescription = "ERROR"
	}

	if datablock[4]&0x40 > 0 {
		vsdbSM.DesktopDisplay = true
	} else {
		vsdbSM.DesktopDisplay = false
	}
	if datablock[4]&0x20 > 0 {
		vsdbSM.DesktopThirdParty = true
	} else {
		vsdbSM.DesktopThirdParty = false
	}
	switch datablock[4] & 0xF {
	case 1:
		vsdbSM.TestEquipment = true
		vsdbSM.PrimaryUseCase = "Test equipment"
	case 2:
		vsdbSM.GenericDisplay = true
		vsdbSM.PrimaryUseCase = "Generic display"
	case 3:
		vsdbSM.TelevisionDisplay = true
		vsdbSM.PrimaryUseCase = "Television display"
	case 4:
		vsdbSM.DesktopProductivityDisplay = true
		vsdbSM.PrimaryUseCase = "Desktop productivity display"
	case 5:
		vsdbSM.DesktopGamingDisplay = true
		vsdbSM.PrimaryUseCase = "Desktop gaming display"
	case 6:
		vsdbSM.PresentationDisplay = true
		vsdbSM.PrimaryUseCase = "Presentation display"
	case 7:
		vsdbSM.VirtualRealityHeadsets = true
		vsdbSM.PrimaryUseCase = "Virtual reality headsets"
	case 8:
		vsdbSM.AugmentedReality = true
		vsdbSM.PrimaryUseCase = "Augmented reality"
	case 10:
		vsdbSM.VideoWallDisplay = true
		vsdbSM.PrimaryUseCase = "Video wall display"
	case 11:
		vsdbSM.MedicalImagingDisplay = true
		vsdbSM.PrimaryUseCase = "Medical imaging display"
	case 12:
		vsdbSM.DedicatedGamingDisplay = true
		vsdbSM.PrimaryUseCase = "Dedicated gaming display"
	case 13:
		vsdbSM.DedicatedVideoMonitorDisplay = true
		vsdbSM.PrimaryUseCase = "Dedicated video monitor display"
	case 14:
		vsdbSM.AccessoryDisplay = true
		vsdbSM.PrimaryUseCase = "Accessory display"
	default:
		vsdbSM.PrimaryUseCase = "NA"
	}
	// TODO: look for overflow
	for i := 5; i < len(datablock); i++ {
		vsdbSM.ContainerID[i] = datablock[i]
	}
	return *vsdbSM
}

func (vsdbSM *VSDB_SpecializedMonitor) EncodeVSDB_SpecializedMonitor() []byte {
	var returnByte []byte
	returnByte = append(returnByte, 0xCA)
	returnByte = append(returnByte, 0x12)
	returnByte = append(returnByte, 0x5C)

	if vsdbSM.WindowsMRExperience {
		returnByte = append(returnByte, 0x1)
	} else if vsdbSM.ThirdPartyMRExperience {
		returnByte = append(returnByte, 0x2)
	} else if vsdbSM.SpecializedDisplay {
		returnByte = append(returnByte, 0x3)
	}

	returnByte = append(returnByte, 0x00)
	if vsdbSM.DesktopDisplay {
		returnByte[4] = returnByte[4] | 0x40
	}
	if vsdbSM.DesktopThirdParty {
		returnByte[4] = returnByte[4] | 0x20
	}

	if vsdbSM.TestEquipment {
		returnByte[4] = returnByte[4] | 0x1
	}
	if vsdbSM.GenericDisplay {
		returnByte[4] = returnByte[4] | 0x2
	}
	if vsdbSM.TelevisionDisplay {
		returnByte[4] = returnByte[4] | 0x3
	}
	if vsdbSM.DesktopProductivityDisplay {
		returnByte[4] = returnByte[4] | 0x4
	}
	if vsdbSM.DesktopGamingDisplay {
		returnByte[4] = returnByte[4] | 0x5
	}
	if vsdbSM.PresentationDisplay {
		returnByte[4] = returnByte[4] | 0x6
	}
	if vsdbSM.VirtualRealityHeadsets {
		returnByte[4] = returnByte[4] | 0x7
	}
	if vsdbSM.AugmentedReality {
		returnByte[4] = returnByte[4] | 0x8
	}
	if vsdbSM.VideoWallDisplay {
		returnByte[4] = returnByte[4] | 0x10
	}
	if vsdbSM.MedicalImagingDisplay {
		returnByte[4] = returnByte[4] | 0x11
	}
	if vsdbSM.DedicatedGamingDisplay {
		returnByte[4] = returnByte[4] | 0x12
	}
	if vsdbSM.DedicatedVideoMonitorDisplay {
		returnByte[4] = returnByte[4] | 0x13
	}
	if vsdbSM.AccessoryDisplay {
		returnByte[4] = returnByte[4] | 0x14
	}
	return returnByte
}

type CEAResulution struct {
	VIC              byte
	Name             string
	Description      string
	PixelMHz         float32
	vHz              float32
	hHz              float32
	HorizontalActive int
	VerticalActive   int
	hTotal           int
	vTotal           float32
	Native           string
}

func DecodeCEAResulutions(videobytes []byte) []CEAResulution {
	var returnBlocks []CEAResulution
	for i := 0; i < len(videobytes); i++ {
		tempVIC := videobytes[i]
		ceaResBlock := vicLooup[tempVIC-1]
		returnBlocks = append(returnBlocks, ceaResBlock)
	}
	return returnBlocks
}

func DecodeCEA(ceaByteArray []byte) (CEA, error) {
	cea := new(CEA)
	cea.ExtensionTag = ceaByteArray[0]
	cea.Revision = ceaByteArray[1]
	cea.DTDStart = ceaByteArray[2]
	cea.DecodeDTDInfo(ceaByteArray[3])

	// CEA DataBlocks
	dataBlocks := DecodeDataBlocks(ceaByteArray[4:cea.DTDStart])
	for _, db := range dataBlocks {
		cea.DataBlocks = append(cea.DataBlocks, db)
	}
	// DTDs
	for i := cea.DTDStart; i < 126; i += 18 {
		var temp [18]byte
		// check for null in pixel clock == end of DTDs
		if ceaByteArray[i] == 0 {
			break
		}

		for e := 0; e < 18; e++ {
			temp[e] = ceaByteArray[e+int(i)]
		}
		thisDTD := DecodeDTD(&temp)

		for e := 0; e < 18; e++ {
			thisDTD.data[e] = ceaByteArray[e+int(i)]
		}
		cea.DetailedTimingDescriptors = append(cea.DetailedTimingDescriptors, thisDTD)
	}
	return *cea, nil
}
