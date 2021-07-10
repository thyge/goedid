package CVTGenerator

import (
	"math"
)

// // CVT - RB
// // Horizontal Active pixels, Total pixels, Sync Pulse, “Front Porch”, “Back Porch” must be divisible by eight
// Horizontal_Blanking = 160         //clock cycles
// Horizontal_SyncPulseDuration = 32 // pix
// // Sync Pulse is located in the center of the Horizontal Blanking period.
// // This implies that the Horizontal Back Porch is fixed to 80 pixel clocks
// Horizontal_BackPorch = 80 //
// // The Vertical Blanking shall be the first multiple of integer Horizontal Lines that exceeds the
// // minimum requirement of 460 microseconds.
// Vertical_Blanking = 460 //us
// // Vertical Front Porch shall in all cases be fixed to three lines
// Vertical_FrontPorch = 3
// // Vertical Sync Width
// Vertical_SyncWidth = 4 // 4:3
// //5 16:9
// //6 16:10
// // 7	5:4 (1280x1024)
// // 7	15:9 (1280x768)
// // If the Vertical Back Porch is less than seven lines, then the Vertical Blanking Time is increased until the Vertical
// // Back Porch equals seven lines. This ensures that the Vertical Back Porch is seven lines or greater.
// CLOCK_STEP_RB := 0.25      //mhz
// Horizontal_Sync = Positive // Sync polarities are used to signal whether the format timing is standard-CRT or Reduced Blanking

type CVTTimingType byte

const (
	CVT        CVTTimingType = 0
	CVT_RB     CVTTimingType = 1
	CVT_RB2    CVTTimingType = 3
	CVT_CUSTOM CVTTimingType = 4
)

type TimingConstaints struct {
	CLOCK_STEP         float64
	MIN_V_BPORCH       float64
	H_BLANK            int
	H_BACK_PORCH       int
	H_SYNC             int
	MIN_V_BLANK        int
	V_FPORCH           int
	NAME               string
	REFRESH_MULTIPLIER int
	VIDEO_OPTIMIZED    float64
}

var timingLookup = map[CVTTimingType]TimingConstaints{
	CVT: {
		CLOCK_STEP:         0.25,
		MIN_V_BPORCH:       6,
		H_BLANK:            160,
		H_SYNC:             32,
		MIN_V_BLANK:        460,
		V_FPORCH:           3,
		REFRESH_MULTIPLIER: 1,
	},
	CVT_RB: {
		CLOCK_STEP:         0.25,
		MIN_V_BPORCH:       6,
		H_BLANK:            160,
		H_BACK_PORCH:       80,
		H_SYNC:             32,
		MIN_V_BLANK:        460,
		V_FPORCH:           3,
		REFRESH_MULTIPLIER: 1,
	},
	CVT_RB2: {
		CLOCK_STEP:         0.001,
		MIN_V_BPORCH:       6,
		H_BLANK:            80,
		H_BACK_PORCH:       40,
		H_SYNC:             32,
		MIN_V_BLANK:        460,
		V_FPORCH:           1,
		REFRESH_MULTIPLIER: 1,
		VIDEO_OPTIMIZED:    1000 / 1001,
	},
}

func GenerateCVT_RB2(hAct int, vAct int, hz float64,
	margins bool, interlaced bool, typ CVTTimingType) edid.DetailedTimingDescriptor {

	H_PIXELS := hAct
	V_LINES := vAct
	IP_FREQ_RQD := hz
	MARGINS_RQD := margins
	INT_RQD := interlaced

	tc := timingLookup[typ]

	var V_FIELD_RATE_RQD float64
	var LEFT_MARGIN float64
	var RIGHT_MARGIN float64
	var V_LINES_RND float64
	var TOP_MARGIN float64
	var BOT_MARGIN float64
	var CELL_GRAN float64 = 8.0
	var MARGIN_PER float64 = 1.8
	var INTERLACE float64
	var ACT_VBI_LINES float64
	var V_SYNC_RND float64 = 8

	var V_BLANK float64
	var V_FRONT_PORCH float64

	var H_FRONT_PORCH int

	var ACT_FRAME_RATE float64
	const (
		CLOCK_STEP          = 0.001
		MIN_V_BPORCH        = 6
		RB_H_BLANK          = 80
		RB_H_SYNC           = 32
		RB_MIN_V_BLANK      = 460
		RB_V_FPORCH         = 1
		REFRESH_MULTIPLIER  = 1
		REFRESH_MULTIPLIER2 = 1000 / 1001
		reduced_blanking    = "cvt_rb2"
	)
	CELL_GRAN_RND := math.Floor(CELL_GRAN)

	// common
	// Find the refresh rate required (Hz):
	if INT_RQD {
		V_FIELD_RATE_RQD = IP_FREQ_RQD * 2
	} else {
		V_FIELD_RATE_RQD = IP_FREQ_RQD
	}
	// round horizontal to nearest divisible by 8
	// not required for rb2
	H_PIXELS_RND := math.Floor(float64(H_PIXELS)/CELL_GRAN_RND) * CELL_GRAN_RND
	// Determine the width of the left and right borders:
	if MARGINS_RQD {
		LEFT_MARGIN = (math.Floor((H_PIXELS_RND*MARGIN_PER/100)/CELL_GRAN_RND) * CELL_GRAN_RND)
		RIGHT_MARGIN = LEFT_MARGIN
	} else {
		LEFT_MARGIN = 0
		RIGHT_MARGIN = 0
	}
	// The total number of active pixels is equal to the rounded horizontal pixels and the margins:
	TOTAL_ACTIVE_PIXELS := H_PIXELS_RND + LEFT_MARGIN + RIGHT_MARGIN
	// If interlace is requested, the number of vertical lines assumed by the calculation must be halved
	if INT_RQD {
		V_LINES_RND = math.Floor(float64(V_LINES) / 2)
	} else {
		V_LINES_RND = float64(V_LINES)
	}
	// Determine the top and bottom margins:
	if MARGINS_RQD {
		TOP_MARGIN = math.Floor(V_LINES_RND * MARGIN_PER / 100)
		BOT_MARGIN = TOP_MARGIN
	} else {
		TOP_MARGIN = 0
		BOT_MARGIN = 0
	}
	// If interlaced is required, then set variable INTERLACE = 0.5:
	if INT_RQD {
		INTERLACE = 0.5
	} else {
		INTERLACE = 0
	}

	// RB Computation
	//Estimate the Horizontal Period (kHz):
	H_PERIOD_EST := ((1000000 / V_FIELD_RATE_RQD) - RB_MIN_V_BLANK) / (V_LINES_RND + TOP_MARGIN + BOT_MARGIN)
	//Determine the number of lines in the vertical blanking interval:
	H_BLANK := RB_H_BLANK
	// Check Vertical Blanking is Sufficient
	VBI_LINES := math.Floor(RB_MIN_V_BLANK/H_PERIOD_EST) + 1
	RB_MIN_VBI := RB_V_FPORCH + V_SYNC_RND + MIN_V_BPORCH
	if VBI_LINES < RB_MIN_VBI {
		ACT_VBI_LINES = RB_MIN_VBI
	} else {
		ACT_VBI_LINES = VBI_LINES
	}
	// Find total number of vertical lines:
	TOTAL_V_LINES := ACT_VBI_LINES + V_LINES_RND + TOP_MARGIN + BOT_MARGIN + INTERLACE
	// Find total number of pixel clocks per line:
	TOTAL_PIXELS := RB_H_BLANK + TOTAL_ACTIVE_PIXELS
	// Calculate Pixel Clock Frequency to nearest CLOCK_STEP MHz:
	ACT_PIXEL_FREQ := tc.CLOCK_STEP * math.Floor((V_FIELD_RATE_RQD*TOTAL_V_LINES*TOTAL_PIXELS/1000000*REFRESH_MULTIPLIER)/tc.CLOCK_STEP)

	// if reduced_blanking == "cvt_rb2" {
	V_BLANK = ACT_VBI_LINES
	V_FRONT_PORCH = ACT_VBI_LINES - V_SYNC_RND - 6
	V_BACK_PORCH := 6

	H_SYNC := 32
	H_BACK_PORCH := 40
	H_FRONT_PORCH = H_BLANK - H_SYNC - H_BACK_PORCH
	// end if
	ACT_H_FREQ := 1000 * float64(ACT_PIXEL_FREQ) / TOTAL_PIXELS
	ACT_FIELD_RATE := 1000 * ACT_H_FREQ / TOTAL_V_LINES
	if INT_RQD {
		ACT_FRAME_RATE = ACT_FIELD_RATE / 2
	} else {
		ACT_FRAME_RATE = ACT_FIELD_RATE
	}

	dtd := new(edid.DetailedTimingDescriptor)
	dtd.SyncType = edid.DigitalSeparate // Does this need to be dynamic?
	dtd.PixelClockKHz = uint32(ACT_PIXEL_FREQ * 100)
	dtd.HorizontalActive = uint16(TOTAL_ACTIVE_PIXELS)
	dtd.HorizontalBlanking = uint16(H_BLANK)
	dtd.HorizontalFrontPorch = uint16(H_FRONT_PORCH)
	dtd.HorizontalSyncPulseWidth = uint16(H_SYNC)
	dtd.HorizontalBackPorch = int(H_BACK_PORCH)
	dtd.VerticalActive = uint16(V_LINES_RND)
	dtd.VerticalBlanking = uint16(V_BLANK)
	dtd.VerticalFrontPorch = uint16(V_FRONT_PORCH)
	dtd.VerticalSyncPulseWidth = uint16(V_SYNC_RND)
	dtd.VerticalBackPorch = int(V_BACK_PORCH)
	dtd.RawPixelClock = ACT_PIXEL_FREQ * 1000000
	dtd.HorizontalTotal = int(TOTAL_PIXELS)
	dtd.VerticalTotal = int(TOTAL_V_LINES)
	dtd.ACT_FRAME_RATE = ACT_FRAME_RATE

	switch typ {
	case CVT:
		dtd.VerticalSyncPolarity = true
		dtd.HorizontalSyncPolarity = false
	case CVT_RB, CVT_RB2:
		dtd.VerticalSyncPolarity = false
		dtd.HorizontalSyncPolarity = true
	default:
		dtd.VerticalSyncPolarity = true
		dtd.HorizontalSyncPolarity = true
	}

	return *dtd
}
