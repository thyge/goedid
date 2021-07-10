package eedid

import (
	"bytes"
	"fmt"
)

type EEDID struct {
	Type      ExtensionType
	Extension interface{}
}

type ExtensionType byte

const (
	TimingExtension                          ExtensionType = 0x00
	EDIDExtension                            ExtensionType = 0x01
	CEAExtension                             ExtensionType = 0x02
	VideoTimingBlockExtension                ExtensionType = 0x10
	EDID2_0Extension                         ExtensionType = 0x20
	DisplayInformationExtension              ExtensionType = 0x40
	LocalizedStringExtension                 ExtensionType = 0x50
	MicrodisplayInterfaceExtension           ExtensionType = 0x60
	DisplayIDExtension                       ExtensionType = 0x70
	DisplayTransferCharacteristicsDataBlock1 ExtensionType = 0xA7
	DisplayTransferCharacteristicsDataBlock2 ExtensionType = 0xAF
	DisplayTransferCharacteristicsDataBlock3 ExtensionType = 0xBF
	BlockMap                                 ExtensionType = 0xF0
	DisplayDeviceDataBlock                   ExtensionType = 0xFF
)

func (et ExtensionType) String() string {
	return extensionLooup[et]
}

var extensionLooup = map[ExtensionType]string{
	TimingExtension:                          "Timing Extension",
	EDIDExtension:                            "Extended Display Identification Data",
	CEAExtension:                             "Additional Timing Data Block (CEA EDID Timing Extension)",
	VideoTimingBlockExtension:                "Video Timing Block Extension (VTB-EXT)",
	EDID2_0Extension:                         "EDID 2.0 Extension",
	DisplayInformationExtension:              "Display Information Extension (DI-EXT)",
	LocalizedStringExtension:                 "Localized String Extension (LS-EXT)",
	MicrodisplayInterfaceExtension:           "Microdisplay Interface Extension (MI-EXT)",
	DisplayIDExtension:                       "Display ID Extension",
	DisplayTransferCharacteristicsDataBlock1: "Display Transfer Characteristics Data Block (DTCDB)",
	DisplayTransferCharacteristicsDataBlock2: "Display Transfer Characteristics Data Block (DTCDB)",
	DisplayTransferCharacteristicsDataBlock3: "Display Transfer Characteristics Data Block (DTCDB)",
	BlockMap:                                 "Block Map",
	DisplayDeviceDataBlock:                   "Display Device Data Block (DDDB)",
}

func (s ExtensionType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(s.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func DecodeEDID(edidBytes []byte) ([]interface{}, error) {
	var decodedExtensions []interface{}

	etyp := EEDID{Type: ExtensionType(0x01)}
	edi, _ := ParseEdid14(edidBytes[0:127])
	etyp.Extension = edi
	decodedExtensions = append(decodedExtensions, etyp)
	for i := 128; i < len(edidBytes); i += 128 {
		etyp := EEDID{Type: ExtensionType(edidBytes[i])}
		switch etyp.Type {
		case CEAExtension:
			cea, err := DecodeCEA(edidBytes[i : i+128])
			if err != nil {
				return nil, err
			}
			etyp.Extension = cea
			decodedExtensions = append(decodedExtensions, etyp)
		case DisplayIDExtension:
			did := DecodeDisplayID(edidBytes[i : i+128])
			etyp.Extension = did
			decodedExtensions = append(decodedExtensions, etyp)
		default:
			fmt.Println("EDID Decoder not supported for:", etyp)
		}
	}
	return decodedExtensions, nil
}

// func (e *E_EDID) EncodeEDID() [256]byte {
// 	var returnBytes [256]byte
// 	// TODO: logic for setting NumberOfExtensions

// 	// Insert EDID
// 	edidBytes := e.EDID.GetBytes()
// 	for i := 0; i < 128; i++ {
// 		returnBytes[i] = edidBytes[i]
// 	}
// 	// Insert CEA
// 	ceaBytes := e.CEA.GetBytes()
// 	for i := 0; i < 128; i++ {
// 		returnBytes[i+128] = ceaBytes[i]
// 	}
// 	return returnBytes
// }

func PrintEDIDAsHex(edid []byte) {
	printWidth := 15
	counter := 0
	for i := 0; i < len(edid); i++ {
		fmt.Printf("%02X ", edid[i])
		if counter == printWidth {
			fmt.Println()
			counter = 0
		} else {
			counter++
		}
	}
}
