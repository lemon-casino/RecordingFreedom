package main

import "image"

func mapRegionSelectionToDisplayCaptureRect(session RegionSelectionSession, relative RegionRect) (image.Rectangle, bool) {
	if len(session.DisplayBounds) == 0 {
		return image.Rectangle{}, false
	}
	absolute := RegionRect{
		X:      session.Bounds.X + relative.X,
		Y:      session.Bounds.Y + relative.Y,
		Width:  relative.Width,
		Height: relative.Height,
	}
	if display, ok := regionDisplayForAbsoluteRect(session.DisplayBounds, absolute); ok {
		return mapDisplayBoundsRectToCapture(display, absolute), true
	}
	origin, originOK := regionAbsolutePointToCapturePoint(session.DisplayBounds, image.Point{X: absolute.X, Y: absolute.Y})
	corner, cornerOK := regionAbsolutePointToCapturePoint(session.DisplayBounds, image.Point{X: absolute.X + absolute.Width, Y: absolute.Y + absolute.Height})
	if !originOK || !cornerOK {
		return image.Rectangle{}, false
	}
	left := minInt(origin.X, corner.X)
	top := minInt(origin.Y, corner.Y)
	right := maxInt(origin.X, corner.X)
	bottom := maxInt(origin.Y, corner.Y)
	if right <= left || bottom <= top {
		return image.Rectangle{}, false
	}
	return image.Rect(left, top, right, bottom), true
}

func mapCaptureRectToDisplaySelection(session RegionSelectionSession, rect image.Rectangle) (RegionRect, bool) {
	if len(session.DisplayBounds) == 0 || rect.Empty() {
		return RegionRect{}, false
	}
	if display, ok := regionDisplayForCaptureRect(session.DisplayBounds, rect); ok {
		absolute := mapDisplayCaptureRectToBounds(display, rect)
		return regionRelativeRectFromAbsolute(session, absolute), true
	}
	origin, originOK := regionCapturePointToAbsolutePoint(session.DisplayBounds, rect.Min)
	corner, cornerOK := regionCapturePointToAbsolutePoint(session.DisplayBounds, rect.Max)
	if !originOK || !cornerOK {
		return RegionRect{}, false
	}
	absolute := RegionRect{
		X:      minInt(origin.X, corner.X),
		Y:      minInt(origin.Y, corner.Y),
		Width:  absInt(corner.X - origin.X),
		Height: absInt(corner.Y - origin.Y),
	}
	if absolute.Width <= 0 || absolute.Height <= 0 {
		return RegionRect{}, false
	}
	return regionRelativeRectFromAbsolute(session, absolute), true
}

func mapRegionPointToDisplayCapturePoint(session RegionSelectionSession, point image.Point) (image.Point, bool) {
	if len(session.DisplayBounds) == 0 {
		return image.Point{}, false
	}
	return regionAbsolutePointToCapturePoint(session.DisplayBounds, image.Point{
		X: session.Bounds.X + point.X,
		Y: session.Bounds.Y + point.Y,
	})
}

func mapRegionPointToAbsolutePoint(session RegionSelectionSession, point image.Point) image.Point {
	return image.Point{
		X: session.Bounds.X + point.X,
		Y: session.Bounds.Y + point.Y,
	}
}

func mapAbsoluteRectToRegionSelection(session RegionSelectionSession, rect image.Rectangle) RegionRect {
	return RegionRect{
		X:      rect.Min.X - session.Bounds.X,
		Y:      rect.Min.Y - session.Bounds.Y,
		Width:  rect.Dx(),
		Height: rect.Dy(),
	}
}

func boundsForRegionPoint(session RegionSelectionSession, point image.Point) image.Rectangle {
	absolute := mapRegionPointToAbsolutePoint(session, point)
	if display, ok := regionDisplayForAbsolutePoint(session.DisplayBounds, absolute); ok {
		return regionRectToImage(display.Bounds)
	}
	return image.Rect(session.Bounds.X, session.Bounds.Y, session.Bounds.X+session.Bounds.Width, session.Bounds.Y+session.Bounds.Height)
}

func captureBoundsForRegionPoint(session RegionSelectionSession, point image.Point) image.Rectangle {
	capturePoint, ok := mapRegionPointToDisplayCapturePoint(session, point)
	if !ok {
		capturePoint = mapRegionPointToCapturePoint(session, point)
	}
	if display, ok := regionDisplayForCapturePoint(session.DisplayBounds, capturePoint); ok {
		return regionRectToImage(display.CaptureBounds)
	}
	return captureBoundsForRegionSession(session)
}

func captureBoundsForRegionSelection(session RegionSelectionSession, selection RegionRect) image.Rectangle {
	captureRect := mapRegionSelectionToCaptureRect(session, selection)
	if display, ok := regionDisplayForCaptureRect(session.DisplayBounds, captureRect); ok {
		return regionRectToImage(display.CaptureBounds)
	}
	return captureBoundsForRegionSession(session)
}

func regionDisplayForAbsoluteRect(displays []RegionDisplayBounds, rect RegionRect) (RegionDisplayBounds, bool) {
	for _, display := range displays {
		if regionRectContainsRect(display.Bounds, rect) {
			return display, true
		}
	}
	return RegionDisplayBounds{}, false
}

func regionDisplayForCaptureRect(displays []RegionDisplayBounds, rect image.Rectangle) (RegionDisplayBounds, bool) {
	if len(displays) == 0 || rect.Empty() {
		return RegionDisplayBounds{}, false
	}
	capture := RegionRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
	for _, display := range displays {
		if regionRectContainsRect(display.CaptureBounds, capture) {
			return display, true
		}
	}
	return RegionDisplayBounds{}, false
}

func regionDisplayForCapturePoint(displays []RegionDisplayBounds, point image.Point) (RegionDisplayBounds, bool) {
	for _, display := range displays {
		if regionRectContainsDisplayPoint(display.CaptureBounds, point) {
			return display, true
		}
	}
	return RegionDisplayBounds{}, false
}

func regionDisplayForAbsolutePoint(displays []RegionDisplayBounds, point image.Point) (RegionDisplayBounds, bool) {
	for _, display := range displays {
		if regionRectContainsDisplayPoint(display.Bounds, point) {
			return display, true
		}
	}
	return RegionDisplayBounds{}, false
}

func regionAbsolutePointToCapturePoint(displays []RegionDisplayBounds, point image.Point) (image.Point, bool) {
	for _, display := range displays {
		if !regionRectContainsDisplayPoint(display.Bounds, point) {
			continue
		}
		return image.Point{
			X: display.CaptureBounds.X + scaleRegionValue(point.X-display.Bounds.X, display.Bounds.Width, display.CaptureBounds.Width),
			Y: display.CaptureBounds.Y + scaleRegionValue(point.Y-display.Bounds.Y, display.Bounds.Height, display.CaptureBounds.Height),
		}, true
	}
	return image.Point{}, false
}

func regionCapturePointToAbsolutePoint(displays []RegionDisplayBounds, point image.Point) (image.Point, bool) {
	for _, display := range displays {
		if !regionRectContainsDisplayPoint(display.CaptureBounds, point) {
			continue
		}
		return image.Point{
			X: display.Bounds.X + scaleRegionValue(point.X-display.CaptureBounds.X, display.CaptureBounds.Width, display.Bounds.Width),
			Y: display.Bounds.Y + scaleRegionValue(point.Y-display.CaptureBounds.Y, display.CaptureBounds.Height, display.Bounds.Height),
		}, true
	}
	return image.Point{}, false
}

func mapDisplayBoundsRectToCapture(display RegionDisplayBounds, rect RegionRect) image.Rectangle {
	x := display.CaptureBounds.X + scaleRegionValue(rect.X-display.Bounds.X, display.Bounds.Width, display.CaptureBounds.Width)
	y := display.CaptureBounds.Y + scaleRegionValue(rect.Y-display.Bounds.Y, display.Bounds.Height, display.CaptureBounds.Height)
	width := scaleRegionValue(rect.Width, display.Bounds.Width, display.CaptureBounds.Width)
	height := scaleRegionValue(rect.Height, display.Bounds.Height, display.CaptureBounds.Height)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return image.Rect(x, y, x+width, y+height)
}

func mapDisplayCaptureRectToBounds(display RegionDisplayBounds, rect image.Rectangle) RegionRect {
	x := display.Bounds.X + scaleRegionValue(rect.Min.X-display.CaptureBounds.X, display.CaptureBounds.Width, display.Bounds.Width)
	y := display.Bounds.Y + scaleRegionValue(rect.Min.Y-display.CaptureBounds.Y, display.CaptureBounds.Height, display.Bounds.Height)
	width := scaleRegionValue(rect.Dx(), display.CaptureBounds.Width, display.Bounds.Width)
	height := scaleRegionValue(rect.Dy(), display.CaptureBounds.Height, display.Bounds.Height)
	return RegionRect{X: x, Y: y, Width: maxInt(1, width), Height: maxInt(1, height)}
}

func regionRelativeRectFromAbsolute(session RegionSelectionSession, absolute RegionRect) RegionRect {
	return RegionRect{
		X:      absolute.X - session.Bounds.X,
		Y:      absolute.Y - session.Bounds.Y,
		Width:  absolute.Width,
		Height: absolute.Height,
	}
}

func regionRectToImage(rect RegionRect) image.Rectangle {
	return image.Rect(rect.X, rect.Y, rect.X+rect.Width, rect.Y+rect.Height)
}

func regionRectContainsRect(outer RegionRect, inner RegionRect) bool {
	if outer.Width <= 0 || outer.Height <= 0 || inner.Width <= 0 || inner.Height <= 0 {
		return false
	}
	return inner.X >= outer.X &&
		inner.Y >= outer.Y &&
		inner.X+inner.Width <= outer.X+outer.Width &&
		inner.Y+inner.Height <= outer.Y+outer.Height
}

func regionRectContainsDisplayPoint(rect RegionRect, point image.Point) bool {
	if rect.Width <= 0 || rect.Height <= 0 {
		return false
	}
	return point.X >= rect.X &&
		point.X < rect.X+rect.Width &&
		point.Y >= rect.Y &&
		point.Y < rect.Y+rect.Height
}
