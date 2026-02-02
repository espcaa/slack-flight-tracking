package maps

import (
	"flight-tracker-slack/flights"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
	"github.com/google/uuid"
)

var planeIcon image.Image

func init() {
	f, err := os.Open("assets/plane.png")
	if err != nil {
		return
	}
	defer f.Close()
	planeIcon, _, _ = image.Decode(f)
}

func GenerateMap(flight flights.FlightDetail) (string, error) {
	if len(flight.Track) == 0 {
		return "", fmt.Errorf("no tracking data")
	}

	ctx := sm.NewContext()
	ctx.SetSize(1200, 900)
	ctx.SetZoom(4)
	ctx.OverrideAttribution("")

	lastPoint := flight.Track[len(flight.Track)-1]
	aircraftPos := s2.LatLngFromDegrees(lastPoint.Coord[1], lastPoint.Coord[0])

	var rotatedIcon image.Image
	if planeIcon != nil {
		w, h := planeIcon.Bounds().Dx(), planeIcon.Bounds().Dy()
		maxDim := float64(w)
		if h > w {
			maxDim = float64(h)
		}

		dc := gg.NewContext(int(maxDim), int(maxDim))
		fmt.Printf("Heading: %d\n", flight.Heading)
		dc.RotateAbout(gg.Radians(float64(flight.Heading)+90), maxDim/2, maxDim/2)
		dc.DrawImageAnchored(planeIcon, int(maxDim/2), int(maxDim/2), 0.5, 0.5)
		rotatedIcon = dc.Image()
	}

	pathPositions := make([]s2.LatLng, len(flight.Track))
	for i, t := range flight.Track {
		pathPositions[i] = s2.LatLngFromDegrees(t.Coord[1], t.Coord[0])
	}
	ctx.AddObject(sm.NewPath(pathPositions, color.RGBA{235, 64, 52, 255}, 3.0))

	if rotatedIcon != nil {
		ctx.AddObject(sm.NewImageMarker(aircraftPos, rotatedIcon, 25, 25))
	}

	img, err := ctx.Render()
	if err != nil {
		return "", fmt.Errorf("failed to render: %w", err)
	}

	if err := os.MkdirAll("tmp", 0755); err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("tmp/%s.png", uuid.New().String())
	return fileName, gg.SavePNG(fileName, img)
}
