package export

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/graph"
)

// ExcalidrawBinding represents the connection of an arrow to an element.
type ExcalidrawBinding struct {
	ElementID string  `json:"elementId"`
	Focus     float64 `json:"focus"`
	Gap       float64 `json:"gap"`
}

// ExcalidrawElement represents a single element in the Excalidraw scene.
type ExcalidrawElement struct {
	Type            string             `json:"type"`
	Version         int                `json:"version"`
	VersionNonce    int                `json:"versionNonce"`
	IsDeleted       bool               `json:"isDeleted"`
	ID              string             `json:"id"`
	FillStyle       string             `json:"fillStyle"`
	StrokeWidth     int                `json:"strokeWidth"`
	StrokeStyle     string             `json:"strokeStyle"`
	Roughness       int                `json:"roughness"`
	Opacity         int                `json:"opacity"`
	Angle           int                `json:"angle"`
	X               float64            `json:"x"`
	Y               float64            `json:"y"`
	StrokeColor     string             `json:"strokeColor"`
	BackgroundColor string             `json:"backgroundColor"`
	Width           float64            `json:"width"`
	Height          float64            `json:"height"`
	Seed            int                `json:"seed"`
	GroupIds        []string           `json:"groupIds"`
	Roundness       any                `json:"roundness"`
	BoundElements   []any              `json:"boundElements"`
	Updated         int64              `json:"updated"`
	Link            any                `json:"link"`
	Locked          bool               `json:"locked"`
	Text            string             `json:"text,omitempty"`
	FontSize        int                `json:"fontSize,omitempty"`
	FontFamily      int                `json:"fontFamily,omitempty"`
	TextAlign       string             `json:"textAlign,omitempty"`
	VerticalAlign   string             `json:"verticalAlign,omitempty"`
	StartBinding    *ExcalidrawBinding `json:"startBinding,omitempty"`
	EndBinding      *ExcalidrawBinding `json:"endBinding,omitempty"`
	Points          [][]float64        `json:"points,omitempty"`
	StartArrowhead  string             `json:"startArrowhead,omitempty"`
	EndArrowhead    string             `json:"endArrowhead,omitempty"`
}

// ExcalidrawScene represents the full file format.
type ExcalidrawScene struct {
	Type     string              `json:"type"`
	Version  int                 `json:"version"`
	Source   string              `json:"source"`
	Elements []ExcalidrawElement `json:"elements"`
	AppState map[string]any      `json:"appState"`
	Files    map[string]any      `json:"files"`
}

// ExportExcalidraw generates an Excalidraw JSON file from the graph.
func ExportExcalidraw(g *graph.Graph, outputPath string) error {
	nodes := g.GetAllNodes()
	elements := []ExcalidrawElement{}

	// Layout constants
	const (
		nodeWidth  = 200.0
		nodeHeight = 100.0
		paddingX   = 50.0
		paddingY   = 50.0
		layerGap   = 300.0
	)

	// Group nodes by layer
	layers := map[string][]*domain.Node{
		"domain":         {},
		"application":    {},
		"infrastructure": {},
		"interface":      {},
		"other":          {},
	}

	nodeMap := make(map[string]ExcalidrawElement) // ID -> Element

	for _, n := range nodes {
		layer := "other"
		if l, ok := n.Metadata["layer"].(string); ok {
			layer = l
		}
		if _, ok := layers[layer]; !ok {
			layer = "other"
		}
		layers[layer] = append(layers[layer], n)
	}

	// Sort layers for deterministic output
	layerOrder := []string{"domain", "application", "interface", "infrastructure", "other"}

	currentY := 0.0

	for _, layerName := range layerOrder {
		layerNodes := layers[layerName]
		if len(layerNodes) == 0 {
			continue
		}

		// Sort nodes by ID
		sort.Slice(layerNodes, func(i, j int) bool {
			return layerNodes[i].ID < layerNodes[j].ID
		})

		// Color mapping
		bgColor := "#ffffff"
		strokeColor := "#000000"
		switch layerName {
		case "domain":
			bgColor = "#e6f7ff" // Light Blue
			strokeColor = "#1890ff"
		case "application":
			bgColor = "#f6ffed" // Light Green
			strokeColor = "#52c41a"
		case "infrastructure":
			bgColor = "#fff7e6" // Light Orange
			strokeColor = "#fa8c16"
		case "interface":
			bgColor = "#fff0f6" // Light Pink
			strokeColor = "#eb2f96"
		}

		currentX := 0.0
		for _, n := range layerNodes {
			// Create Rectangle
			rect := ExcalidrawElement{
				Type:            "rectangle",
				Version:         1,
				VersionNonce:    0,
				IsDeleted:       false,
				ID:              n.ID,
				FillStyle:       "solid",
				StrokeWidth:     1,
				StrokeStyle:     "solid",
				Roughness:       1,
				Opacity:         100,
				Angle:           0,
				X:               currentX,
				Y:               currentY,
				StrokeColor:     strokeColor,
				BackgroundColor: bgColor,
				Width:           nodeWidth,
				Height:          nodeHeight,
				Seed:            1,
				GroupIds:        []string{},
				Roundness:       map[string]int{"type": 3},
			}
			elements = append(elements, rect)
			nodeMap[n.ID] = rect

			// Create Text Label
			text := ExcalidrawElement{
				Type:            "text",
				Version:         1,
				VersionNonce:    0,
				IsDeleted:       false,
				ID:              n.ID + "-text",
				FillStyle:       "solid",
				StrokeWidth:     1,
				StrokeStyle:     "solid",
				Roughness:       1,
				Opacity:         100,
				Angle:           0,
				X:               currentX + 10,
				Y:               currentY + 10,
				StrokeColor:     "#000000",
				BackgroundColor: "transparent",
				Width:           nodeWidth - 20,
				Height:          nodeHeight - 20,
				Seed:            1,
				GroupIds:        []string{},
				Text:            fmt.Sprintf("%s\n(%s)", n.ID, n.Kind),
				FontSize:        16,
				FontFamily:      1,
				TextAlign:       "left",
				VerticalAlign:   "top",
			}
			elements = append(elements, text)

			currentX += nodeWidth + paddingX
		}
		currentY += nodeHeight + layerGap
	}

	// Create Edges (Arrows)
	for _, n := range nodes {
		edges := g.GetEdgesFrom(n.ID)
		sourceRect, ok1 := nodeMap[n.ID]
		if !ok1 {
			continue
		}

		for _, e := range edges {
			targetRect, ok2 := nodeMap[e.TargetID]
			if !ok2 {
				continue
			}

			// Calculate start and end points (center to center roughly)
			startX := sourceRect.X + nodeWidth/2
			startY := sourceRect.Y + nodeHeight
			endX := targetRect.X + nodeWidth/2
			endY := targetRect.Y

			arrow := ExcalidrawElement{
				Type:            "arrow",
				Version:         1,
				VersionNonce:    0,
				IsDeleted:       false,
				ID:              fmt.Sprintf("%s-%s", n.ID, e.TargetID),
				FillStyle:       "solid",
				StrokeWidth:     1,
				StrokeStyle:     "solid",
				Roughness:       1,
				Opacity:         100,
				Angle:           0,
				X:               startX,
				Y:               startY,
				StrokeColor:     "#000000",
				BackgroundColor: "transparent",
				Width:           endX - startX,
				Height:          endY - startY,
				Seed:            1,
				GroupIds:        []string{},
				Points:          [][]float64{{0, 0}, {endX - startX, endY - startY}},
				StartBinding: &ExcalidrawBinding{
					ElementID: sourceRect.ID,
					Focus:     0.1,
					Gap:       1,
				},
				EndBinding: &ExcalidrawBinding{
					ElementID: targetRect.ID,
					Focus:     0.1,
					Gap:       1,
				},
				EndArrowhead: "arrow",
			}
			elements = append(elements, arrow)
		}
	}

	scene := ExcalidrawScene{
		Type:     "excalidraw",
		Version:  2,
		Source:   "hexanorm",
		Elements: elements,
		AppState: map[string]any{"viewBackgroundColor": "#ffffff"},
		Files:    map[string]any{},
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(scene)
}
