package maps

import (
	"flight-tracker-slack/flights"
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"
)

type GlobalPixel struct {
	X, Y float64
}

type TileStore struct {
	cache    map[string]image.Image
	mu       sync.RWMutex
	basePath string
}

var planeColor string = "#99b8cc"
var planeOutlineColor string = "#99b8cc"
var planeIcon image.Image
var planeIcons = make(map[string]image.Image)

func init() {
	// load everything from assets/planes into planeIcons
	var dir, err = os.ReadDir("assets/planes")

	if err != nil {
		panic(err)
	}
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		// check if it's a .svg file
		if entry.Name()[len(entry.Name())-4:] != ".svg" {
			continue
		}
		filePath := "assets/planes/" + entry.Name()
		f, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		// now recolor the svg file's fill and stroke attributes
		svgData, err := os.ReadFile(filePath)
		if err != nil {
			panic(err)
		}
		svgStr := string(svgData)

		svg := strings.ReplaceAll(svgStr, "{{FILL}}", planeColor)
		svg = strings.ReplaceAll(svg, "{{STROKE}}", planeOutlineColor)

		// rasterize the svg to an image.Image
		icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
		if err != nil {
			panic(err)
		}

		size := 256
		icon.SetTarget(0, 0, float64(size), float64(size))
		img := image.NewRGBA(image.Rect(0, 0, size, size))

		// Rasterize
		scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
		raster := rasterx.NewDasher(size, size, scanner)
		icon.Draw(raster, 1.0)

		planeIcons[entry.Name()[:len(entry.Name())-4]] = img
	}
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

func GenerateMapFromFlightDetail(store *TileStore, flightDetails flights.FlightDetail) (*image.RGBA, error) {
	start := time.Now()
	zoom := 6

	var lastTrackPoint flights.TrackPoint
	for _, tp := range flightDetails.Track {
		if tp.Timestamp > lastTrackPoint.Timestamp {
			lastTrackPoint = tp
		}
	}

	rawTopLat := flightDetails.Origin.Coordinates[1]
	rawBottomLat := flightDetails.Origin.Coordinates[1]
	rawLeftLon := flightDetails.Origin.Coordinates[0]
	rawRightLon := flightDetails.Origin.Coordinates[0]

	// expand bounds to include all track points
	for _, tp := range flightDetails.Track {
		lat, lon := tp.Coord[1], tp.Coord[0]
		if lat > rawTopLat {
			rawTopLat = lat
		}
		if lat < rawBottomLat {
			rawBottomLat = lat
		}
		if lon < rawLeftLon {
			rawLeftLon = lon
		}
		if lon > rawRightLon {
			rawRightLon = lon
		}

		if tp.Timestamp > lastTrackPoint.Timestamp {
			lastTrackPoint = tp
		}
	}
	// do the same for waypoints
	for _, wp := range flightDetails.Waypoints {
		lon, lat := wp[0], wp[1]

		if lat > rawTopLat {
			rawTopLat = lat
		}
		if lat < rawBottomLat {
			rawBottomLat = lat
		}
		if lon < rawLeftLon {
			rawLeftLon = lon
		}
		if lon > rawRightLon {
			rawRightLon = lon
		}
	}

	dLat, dLon := flightDetails.Destination.Coordinates[1], flightDetails.Destination.Coordinates[0]
	if dLat > rawTopLat {
		rawTopLat = dLat
	}
	if dLat < rawBottomLat {
		rawBottomLat = dLat
	}
	if dLon < rawLeftLon {
		rawLeftLon = dLon
	}
	if dLon > rawRightLon {
		rawRightLon = dLon
	}

	pMin := LonLatToPixel(rawTopLat, rawLeftLon, zoom)
	pMax := LonLatToPixel(rawBottomLat, rawRightLon, zoom)

	// huge padding so that it looks nice for short flights!
	p1X, p1Y := pMin.X-256, pMin.Y-256
	p2X, p2Y := pMax.X+256, pMax.Y+256

	canvas, err := DrawCroppedMapPixels(store, zoom, p1X, p1Y, p2X, p2Y)
	if err != nil {
		return nil, err
	}

	// scaling thingies

	imgW := float64(canvas.Bounds().Dx())
	imgH := float64(canvas.Bounds().Dy())

	base := math.Sqrt(imgW * imgH)

	dc := gg.NewContextForRGBA(canvas)

	// draw the arrival & departure airports as circles

	airportColor := "#f5bac6"
	originPix := LonLatToPixel(flightDetails.Origin.Coordinates[1], flightDetails.Origin.Coordinates[0], zoom)
	dc.SetHexColor(airportColor)
	dc.DrawCircle(originPix.X-p1X, originPix.Y-p1Y, base*0.01)
	dc.Fill()

	destPix := LonLatToPixel(flightDetails.Destination.Coordinates[1], flightDetails.Destination.Coordinates[0], zoom)
	dc.SetHexColor(airportColor)
	dc.DrawCircle(destPix.X-p1X, destPix.Y-p1Y, base*0.01)
	dc.Fill()

	// draw the waypoints as a dashed line
	if len(flightDetails.Waypoints) >= 2 {
		dc.SetLineWidth(base * 0.003)
		dc.SetDash(base*0.01, base*0.01)
		dc.SetHexColor("#6b95b0")

		// Move to the first point
		first := flightDetails.Waypoints[0]
		firstPix := LonLatToPixel(first[1], first[0], zoom)
		dc.MoveTo(firstPix.X-p1X, firstPix.Y-p1Y)

		// Add lines to the path
		for i := 1; i < len(flightDetails.Waypoints); i++ {
			curr := flightDetails.Waypoints[i]
			currPix := LonLatToPixel(curr[1], curr[0], zoom)
			dc.LineTo(currPix.X-p1X, currPix.Y-p1Y)
		}

		// Stroke ONCE here to apply the dash pattern across the whole path
		dc.Stroke()

		// Reset dash so it doesn't affect future drawing
		dc.SetDash()
	}

	// draw the flight track from each track point
	if len(flightDetails.Track) >= 2 {

		dc.SetLineWidth(base * 0.006)
		dc.SetHexColor("#bbe1fa")
		for i := 1; i < len(flightDetails.Track); i++ {
			prev := flightDetails.Track[i-1]
			curr := flightDetails.Track[i]
			prevPix := LonLatToPixel(prev.Coord[1], prev.Coord[0], zoom)
			currPix := LonLatToPixel(curr.Coord[1], curr.Coord[0], zoom)
			dc.DrawLine(prevPix.X-p1X, prevPix.Y-p1Y, currPix.X-p1X, currPix.Y-p1Y)
			dc.Stroke()
		}
	}

	// draw the plane icon!

	iconInfo, ok := TypeDesignatorIcons[flightDetails.Aircraft.Type]
	if !ok {
		iconInfo = TypeDesignatorIcon{Icon: "unknown", Scale: 1.0}
	}

	planeIcon, ok := planeIcons[iconInfo.Icon]
	if !ok {
		planeIcon = planeIcons["unknown"]
	}

	// size the plane icon relative to the image size

	planeSize := base * 0.06 * iconInfo.Scale
	planePix := LonLatToPixel(lastTrackPoint.Coord[1], lastTrackPoint.Coord[0], zoom)
	planeX := planePix.X - p1X
	planeY := planePix.Y - p1Y
	iconW := float64(planeIcon.Bounds().Dx())
	scale := planeSize / iconW

	dc.Push()
	dc.Translate(planeX, planeY)
	dc.Rotate(gg.Radians(float64(flightDetails.Heading)))
	dc.Scale(scale, scale)
	dc.DrawImageAnchored(planeIcon, 0, 0, 0.5, 0.5)
	dc.Pop()

	duration := time.Since(start)
	fmt.Printf("Map generation took: %s\n", duration)

	return canvas, nil
}
