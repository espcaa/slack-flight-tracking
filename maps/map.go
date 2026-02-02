package maps

import (
	"flight-tracker-slack/flights"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sync"

	"github.com/fogleman/gg"
	"github.com/google/uuid"
)

var planeIcon image.Image

type GlobalPixel struct {
	X, Y float64
}

type TileStore struct {
	cache    map[string]image.Image
	mu       sync.RWMutex
	basePath string
}

func init() {
	f, err := os.Open("assets/plane.png")
	if err != nil {
		return
	}
	defer f.Close()
	planeIcon, _, _ = image.Decode(f)
}

func NewTileStore(path string) *TileStore {
	return &TileStore{
		cache:    make(map[string]image.Image),
		basePath: path,
	}
}

func LonLatToPixel(lat, lon float64, zoom int) GlobalPixel {
	size := float64(uint(1<<zoom) * 256)
	x := (lon + 180) / 360 * size
	latRad := lat * math.Pi / 180
	y := (1 - math.Log(math.Tan(latRad)+(1/math.Cos(latRad)))/math.Pi) / 2 * size
	return GlobalPixel{x, y}
}

func (ts *TileStore) GetTile(z, x, y int) (image.Image, error) {
	key := fmt.Sprintf("%d/%d/%d", z, x, y)
	ts.mu.RLock()
	if img, ok := ts.cache[key]; ok {
		ts.mu.RUnlock()
		return img, nil
	}
	ts.mu.RUnlock()

	filePath := fmt.Sprintf("%s/%s.png", ts.basePath, key)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	ts.mu.Lock()
	ts.cache[key] = img
	ts.mu.Unlock()
	return img, nil
}

func DrawCroppedMapPixels(store *TileStore, zoom int, x1, y1, x2, y2 float64) (*image.RGBA, error) {
	imgWidth := int(math.Abs(x2 - x1))
	imgHeight := int(math.Abs(y2 - y1))

	minTileX, maxTileX := int(math.Floor(x1/256)), int(math.Floor(x2/256))
	minTileY, maxTileY := int(math.Floor(y1/256)), int(math.Floor(y2/256))

	canvas := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	for tx := minTileX; tx <= maxTileX; tx++ {
		for ty := minTileY; ty <= maxTileY; ty++ {
			tile, err := store.GetTile(zoom, tx, ty)
			if err != nil {
				continue
			}
			posX := (tx * 256) - int(x1)
			posY := (ty * 256) - int(y1)
			draw.Draw(canvas, image.Rectangle{
				Min: image.Point{posX, posY},
				Max: image.Point{posX + 256, posY + 256},
			}, tile, image.Point{0, 0}, draw.Over)
		}
	}
	return canvas, nil
}

func GenerateMapFromFlightDetail(store *TileStore, flightDetails flights.FlightDetail) (string, error) {
	zoom := 4

	rawTopLat := math.Max(flightDetails.Origin.Coordinates[1], flightDetails.Destination.Coordinates[1])
	rawBottomLat := math.Min(flightDetails.Origin.Coordinates[1], flightDetails.Destination.Coordinates[1])
	rawLeftLon := math.Min(flightDetails.Origin.Coordinates[0], flightDetails.Destination.Coordinates[0])
	rawRightLon := math.Max(flightDetails.Origin.Coordinates[0], flightDetails.Destination.Coordinates[0])

	pMin := LonLatToPixel(rawTopLat, rawLeftLon, zoom)
	pMax := LonLatToPixel(rawBottomLat, rawRightLon, zoom)

	p1X, p1Y := pMin.X-256, pMin.Y-256
	p2X, p2Y := pMax.X+256, pMax.Y+256

	canvas, err := DrawCroppedMapPixels(store, zoom, p1X, p1Y, p2X, p2Y)
	if err != nil {
		return "", err
	}

	var lastTrackPoint flights.TrackPoint
	for _, tp := range flightDetails.Track {
		if tp.Timestamp > lastTrackPoint.Timestamp {
			lastTrackPoint = tp
		}
	}

	planePix := LonLatToPixel(lastTrackPoint.Coord[1], lastTrackPoint.Coord[0], zoom)
	planeX := planePix.X - p1X
	planeY := planePix.Y - p1Y

	if planeIcon != nil {
		offsetX := int(planeX) - planeIcon.Bounds().Dx()/2
		offsetY := int(planeY) - planeIcon.Bounds().Dy()/2
		draw.Draw(canvas, image.Rectangle{
			Min: image.Point{offsetX, offsetY},
			Max: image.Point{offsetX + planeIcon.Bounds().Dx(), offsetY + planeIcon.Bounds().Dy()},
		}, planeIcon, image.Point{0, 0}, draw.Over)
	}

	dc := gg.NewContextForRGBA(canvas)
	if err := dc.LoadFontFace("assets/figtree-heavy.ttf", 24); err != nil {
		return "", err
	}

	dc.SetRGB(1, 1, 1)

	// draw the origin airport's IATA code at its location
	originPix := LonLatToPixel(flightDetails.Origin.Coordinates[1], flightDetails.Origin.Coordinates[0], zoom)
	originX := originPix.X - p1X
	originY := originPix.Y - p1Y
	dc.DrawStringAnchored(flightDetails.Origin.Iata, originX, originY-10, 0.5, 1)

	// draw the destination airport's IATA code at its location
	destPix := LonLatToPixel(flightDetails.Destination.Coordinates[1], flightDetails.Destination.Coordinates[0], zoom)
	destX := destPix.X - p1X
	destY := destPix.Y - p1Y
	dc.DrawStringAnchored(flightDetails.Destination.Iata, destX, destY-10, 0.5, 1)

	outputPath := fmt.Sprintf("flight_map_%s.png", uuid.New().String())
	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if err := png.Encode(outFile, canvas); err != nil {
		return "", err
	}

	return outputPath, nil
}
