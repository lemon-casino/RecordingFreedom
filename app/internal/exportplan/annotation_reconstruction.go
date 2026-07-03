package exportplan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

const (
	annotationElementTimelineReconstructed = "element-events"
	annotationElementTimelinePartial       = "element-events-partial"
	annotationRenderModeElementScenes      = "element-scenes"
	annotationRenderModeElementPNGs        = "element-pngs"
	maxAnnotationElementPreviewFrames      = 64
	maxAnnotationElementSceneAssets        = 2000
)

type annotationElementState struct {
	ID       string
	Type     string
	Version  int64
	OffsetMs int64
	Element  json.RawMessage
}

type annotationElementReconstructor struct {
	active                 map[string]annotationElementState
	deleted                map[string]bool
	previewFrames          []AnnotationElementKeyframePlan
	sceneFrames            []annotationElementSceneFrame
	keyframeCount          int
	missingElementPayloads int
	startOffsetMs          int64
	endOffsetMs            int64
	collectSceneFrames     bool
}

type annotationElementSceneFrame struct {
	Sequence      int
	StartOffsetMs int64
	EventType     string
	ElementID     string
	Elements      []json.RawMessage
}

func newAnnotationElementReconstructor(collectSceneFrames bool) *annotationElementReconstructor {
	return &annotationElementReconstructor{
		active:             map[string]annotationElementState{},
		deleted:            map[string]bool{},
		startOffsetMs:      -1,
		endOffsetMs:        -1,
		collectSceneFrames: collectSceneFrames,
	}
}

func (r *annotationElementReconstructor) Apply(event map[string]any) {
	if r == nil {
		return
	}
	eventType := annotationEventString(event, "type")
	if !strings.HasPrefix(eventType, "element-") {
		return
	}
	elementID := annotationEventString(event, "elementId")
	if elementID == "" {
		r.missingElementPayloads++
		return
	}
	offsetMs, ok := annotationEventOffsetMs(event)
	if !ok {
		offsetMs = 0
	}
	if r.startOffsetMs < 0 {
		r.startOffsetMs = offsetMs
	}
	if offsetMs > r.endOffsetMs {
		r.endOffsetMs = offsetMs
	}
	elementType := annotationEventString(event, "elementType")
	elementVersion, _ := annotationEventInteger(event["elementVersion"])
	sequence, _ := annotationEventInteger(event["sequence"])
	payload, hasPayload := annotationElementPayload(event)
	isDeleted := eventType == "element-deleted" || annotationEventBool(event["isDeleted"])
	if isDeleted {
		delete(r.active, elementID)
		r.deleted[elementID] = true
	} else if hasPayload {
		if elementType == "" {
			elementType = annotationElementPayloadString(payload, "type")
		}
		r.active[elementID] = annotationElementState{
			ID:       elementID,
			Type:     elementType,
			Version:  elementVersion,
			OffsetMs: offsetMs,
			Element:  payload,
		}
		delete(r.deleted, elementID)
	} else {
		r.missingElementPayloads++
	}
	r.keyframeCount++
	if len(r.previewFrames) < maxAnnotationElementPreviewFrames {
		r.previewFrames = append(r.previewFrames, AnnotationElementKeyframePlan{
			Sequence:           int(sequence),
			StartOffsetMs:      offsetMs,
			EventType:          eventType,
			ElementID:          elementID,
			ElementType:        elementType,
			ActiveElementCount: len(r.active),
			HasElementPayload:  hasPayload,
		})
	}
	if r.collectSceneFrames && len(r.sceneFrames) < maxAnnotationElementSceneAssets {
		r.sceneFrames = append(r.sceneFrames, annotationElementSceneFrame{
			Sequence:      int(sequence),
			StartOffsetMs: offsetMs,
			EventType:     eventType,
			ElementID:     elementID,
			Elements:      r.ActiveElements(),
		})
	}
}

func (r *annotationElementReconstructor) ApplyToSummary(summary *AnnotationTimelineSummary) {
	if r == nil || summary == nil || r.keyframeCount == 0 {
		return
	}
	mode := annotationElementTimelineReconstructed
	if r.missingElementPayloads > 0 {
		mode = annotationElementTimelinePartial
	}
	summary.ElementTimelineMode = mode
	summary.ElementKeyframeCount = r.keyframeCount
	summary.FinalElementCount = len(r.active)
	summary.DeletedElementCount = len(r.deleted)
	summary.MissingElementPayloads = r.missingElementPayloads
	summary.ElementTypeCounts = r.elementTypeCounts()
	summary.ElementPreviewFrames = append([]AnnotationElementKeyframePlan(nil), r.previewFrames...)
}

func (r *annotationElementReconstructor) elementTypeCounts() map[string]int {
	if r == nil || len(r.active) == 0 {
		return nil
	}
	counts := map[string]int{}
	for _, state := range r.active {
		elementType := strings.TrimSpace(state.Type)
		if elementType == "" {
			elementType = "unknown"
		}
		counts[elementType]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func (r *annotationElementReconstructor) ActiveElements() []json.RawMessage {
	if r == nil || len(r.active) == 0 {
		return nil
	}
	states := make([]annotationElementState, 0, len(r.active))
	for _, state := range r.active {
		states = append(states, state)
	}
	sort.SliceStable(states, func(i, j int) bool {
		if states[i].OffsetMs == states[j].OffsetMs {
			return states[i].ID < states[j].ID
		}
		return states[i].OffsetMs < states[j].OffsetMs
	})
	elements := make([]json.RawMessage, 0, len(states))
	for _, state := range states {
		if len(state.Element) > 0 {
			elements = append(elements, append(json.RawMessage(nil), state.Element...))
		}
	}
	return elements
}

func (r *annotationElementReconstructor) BuildSceneAssets(packageDir string, canvasWidth int, canvasHeight int) ([]AnnotationElementScenePlan, []string, error) {
	if r == nil || !r.collectSceneFrames || r.keyframeCount == 0 {
		return nil, nil, nil
	}
	if r.missingElementPayloads > 0 {
		return nil, []string{"annotation element scene assets were not generated because element reconstruction is partial"}, nil
	}
	if r.keyframeCount > maxAnnotationElementSceneAssets {
		return nil, []string{fmt.Sprintf("annotation element scene assets were not generated because %d keyframes exceeds the %d scene asset limit; snapshot timeline will be used", r.keyframeCount, maxAnnotationElementSceneAssets)}, nil
	}
	if canvasWidth <= 0 || canvasHeight <= 0 {
		return nil, []string{"annotation element scene assets were not generated because recording canvas size is unknown; snapshot timeline will be used"}, nil
	}
	if len(r.sceneFrames) == 0 {
		return nil, nil, nil
	}
	outputDir := filepath.Join(packageDir, recpackage.AnnotationRenderDir)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, nil, err
	}
	scenes := make([]AnnotationElementScenePlan, 0, len(r.sceneFrames))
	for index, frame := range r.sceneFrames {
		relativePath := filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderDir, fmt.Sprintf("scene-%06d.excalidraw", index+1)))
		outputPath := filepath.Join(packageDir, filepath.FromSlash(relativePath))
		data, err := annotationElementSceneJSON(frame.Elements)
		if err != nil {
			return nil, nil, err
		}
		if err := writeAnnotationElementSceneFile(outputPath, data); err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(outputPath)
		if err != nil {
			return nil, nil, err
		}
		scenes = append(scenes, AnnotationElementScenePlan{
			InputPath:           outputPath,
			RelativePath:        relativePath,
			RenderInputPath:     filepath.Join(packageDir, filepath.FromSlash(annotationElementSceneRenderPath(index+1))),
			RenderRelativePath:  annotationElementSceneRenderPath(index + 1),
			StartOffsetMs:       frame.StartOffsetMs,
			CanvasWidth:         canvasWidth,
			CanvasHeight:        canvasHeight,
			ElementCount:        len(frame.Elements),
			SourceEventSequence: frame.Sequence,
			Bytes:               info.Size(),
		})
	}
	for index := 0; index < len(scenes)-1; index++ {
		nextStart := scenes[index+1].StartOffsetMs
		if nextStart > scenes[index].StartOffsetMs {
			scenes[index].EndOffsetMs = nextStart
			scenes[index].DurationMs = nextStart - scenes[index].StartOffsetMs
		}
	}
	return scenes, nil, nil
}

func annotationElementSceneRenderPath(index int) string {
	return filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, fmt.Sprintf("annotation-%06d.png", index)))
}

func annotationElementSceneJSON(elements []json.RawMessage) ([]byte, error) {
	rawElements := make([]json.RawMessage, 0, len(elements))
	for _, element := range elements {
		if len(element) > 0 {
			rawElements = append(rawElements, element)
		}
	}
	scene := map[string]any{
		"type":     "excalidraw",
		"version":  2,
		"source":   "RecordingFreedom",
		"elements": rawElements,
		"appState": map[string]any{
			"exportBackground":    false,
			"viewBackgroundColor": "transparent",
		},
		"files": map[string]any{},
	}
	return json.MarshalIndent(scene, "", "  ")
}

func writeAnnotationElementSceneFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func annotationElementPayload(event map[string]any) (json.RawMessage, bool) {
	value, ok := event["element"]
	if !ok || value == nil {
		return nil, false
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) == 0 || string(data) == "null" {
		return nil, false
	}
	return data, true
}

func annotationEventString(event map[string]any, key string) string {
	value, ok := event[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func annotationElementPayloadString(payload json.RawMessage, key string) string {
	if len(payload) == 0 {
		return ""
	}
	event := map[string]any{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return ""
	}
	return annotationEventString(event, key)
}

func annotationEventBool(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func annotationEventInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	default:
		return 0, false
	}
}
