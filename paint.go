package main

import (
	"fmt"
	"math"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	width         int32 = 1600
	height        int32 = 900
	lastMousePos  rl.Vector2
	changed       bool    = false
	pixelSize     float32 = 10.0
	FPS           int32   = 60
	frame         int32   = 0
	frameSpeed    int32   = 30
	wg            sync.WaitGroup
	uiMode        bool     = true
	selectedColor rl.Color = rl.Black
	CONFIG_COLOR  rl.Color = rl.Magenta
	colorPicker            = ColorPicker{
		Colors: []rl.Color{rl.Black, rl.Blue, rl.Pink, rl.Purple, rl.Yellow, rl.Orange, rl.Red, rl.Green},
		Center: lastMousePos,
		Radius: 120,
	}
	selectedBoundingBox *BoundingBox = nil
)

type BoundingBox struct {
	BoundingBox rl.BoundingBox
	Scribble    []*Pixel
}

func main() {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(width, height, "Paint")

	// might cause trouble
	rl.SetWindowState(rl.FlagWindowAlwaysRun)

	// rl.SetWindowState(rl.FlagVsyncHint)
	rl.SetTargetFPS(FPS)

	defer rl.CloseWindow()

	target := rl.LoadRenderTexture(width, height)
	halfScreenW := rl.GetScreenWidth() / 2
	halfScreenH := rl.GetScreenHeight() / 2
	buttonWidth := 400
	buttonHeight := 100
	serverButton := NewButton(halfScreenW, halfScreenH-(buttonHeight/2)-20, buttonWidth, buttonHeight, rl.Black, "Host", 40)
	clientButton := NewButton(halfScreenW, halfScreenH+(buttonHeight/2)+20, buttonWidth, buttonHeight, rl.Black, "Enter", 40)

	for !rl.WindowShouldClose() {
		frame++
		// ui mode: choose if you're gonna host or enter
		// else: start paint screen
		if uiMode {
			rl.BeginDrawing()
			rl.ClearBackground(rl.White)
			serverButton.Draw()
			serverButton.Click(func() {
				go StartServer()
				go StartClient()
				uiMode = false
			})

			clientButton.Draw()
			clientButton.Click(func() {
				go StartClient()
				uiMode = false
			})
			rl.EndDrawing()
		}

		if !uiMode {
			Input()
			DrawBoard(target)
		}

		if frame == FPS/frameSpeed {
			frame = 0
		}
	}

	// close the application window so they left
	clientEventsToSend <- &Event{
		PlayerId:   me.Id,
		Kind:       "left",
		InnerEvent: LeftEvent{},
	}
	wg.Wait()
}

func Input() {
	HandlePainting()
	HandleColorPicker()

	if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
		go IsMouseClickOnScribble(rl.GetMousePosition())
	}

	if rl.IsKeyDown(rl.KeyEqual) && frame == FPS/frameSpeed {
		pixelSize++
	}

	if rl.IsKeyDown(rl.KeyMinus) && pixelSize > 1 && frame == FPS/frameSpeed {
		pixelSize--
	}

	if rl.IsKeyPressed(rl.KeyU) {
		clientEventsToSend <- &Event{
			PlayerId:   me.Id,
			Kind:       "undo",
			InnerEvent: UndoEvent{},
		}
	}

	if rl.IsKeyPressed(rl.KeyR) {
		clientEventsToSend <- &Event{
			PlayerId:   me.Id,
			Kind:       "redo",
			InnerEvent: RedoEvent{},
		}
	}
}

func DrawBoard(target rl.RenderTexture2D) {
	rl.BeginDrawing()
	rl.ClearBackground(rl.White)

	DrawIfChanged()
	DrawCache()

	if selectedBoundingBox != nil {
		rl.DrawBoundingBox(selectedBoundingBox.BoundingBox, CONFIG_COLOR)
	}

	rl.DrawCircleLines(rl.GetMouseX(), rl.GetMouseY(), pixelSize, rl.Black)
	rl.DrawFPS(width-200, 20)

	mouseXText := fmt.Sprintf("Mouse X: %d", int(rl.GetMousePosition().X))
	mouseYText := fmt.Sprintf("Mouse Y: %d", int(rl.GetMousePosition().Y))
	rl.DrawText(mouseXText, width-200, 40, 20, rl.Black)
	rl.DrawText(mouseYText, width-200, 60, 20, rl.Black)

	pencilSizeText := fmt.Sprintf("Pencil size: %d", int(pixelSize))
	rl.DrawText(pencilSizeText, 10, 10, 20, CONFIG_COLOR)

	rl.DrawText("Selected color: ", 10, 40, 20, CONFIG_COLOR)
	rl.DrawCircle(180, 50, 10, selectedColor)

	rl.EndDrawing()
}

func DrawIfChanged() {
	if changed {
		for _, player := range players {
			if player.Drawing {
				currentlyDrawingArray := player.Scribbles[len(player.Scribbles)-1]
				cache := GetCache(player, len(player.CachedScribbles)-1)
				if cache == nil {
					panic("Cache nil")
				}
				texture := cache.RenderTexture2D
				DrawScribble(currentlyDrawingArray, *texture)
			} else if player.JustJoined {
				for i := range len(player.Scribbles) {
					scribble := player.Scribbles[i]

					// to remember:
					// I initialized cache by iterating over scribbles,
					// so now cache array is the same size as scribbles.
					cache := GetCache(player, i)
					if cache == nil {
						panic("Cache nil")
					}
					texture := cache.RenderTexture2D
					DrawScribble(scribble, *texture)
					cache.Drawing = false
				}
				player.JustJoined = false
			}
		}
	}
}

func DrawScribble(scribble []*Pixel, renderTexture2D rl.RenderTexture2D) {
	rl.BeginTextureMode(renderTexture2D)
	rl.ClearBackground(rl.Blank)
	var lastPixelLoop *Pixel
	for i, pixel := range scribble {
		rl.DrawCircleV(pixel.Center, pixel.Radius, pixel.Color)
		// Draws a line between the last and newest pixel
		if i > 0 && lastPixelLoop != nil {
			rl.DrawLineEx(pixel.Center, lastPixelLoop.Center, pixel.Radius*2, pixel.Color)
		}
		lastPixelLoop = pixel
	}
	lastPixelLoop = &Pixel{}
	rl.EndTextureMode()
}

func DrawCache() {
	for _, player := range players {
		for i, cache := range player.CachedScribbles {
			// Draws textures that are ready to be drawn or the last texture the player is currently drawing on
			shouldDraw := !cache.Drawing || player.Drawing && i == len(player.CachedScribbles)-1
			if shouldDraw {
				rl.DrawTextureRec(
					cache.RenderTexture2D.Texture,
					rl.Rectangle{
						X: 0, Y: 0,
						Width:  float32(cache.RenderTexture2D.Texture.Width),
						Height: -float32(cache.RenderTexture2D.Texture.Height),
					},
					rl.Vector2{X: 0, Y: 0},
					rl.White,
				)
			}
		}
	}
}

func HandlePainting() {
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		mousePos := rl.GetMousePosition()
		newPixel := Pixel{mousePos, pixelSize, selectedColor}

		// avoid send redundant events, otherwise, drawing will be true
		// as long as the player hold the mouse button
		// so it would send these events again
		if !me.Drawing {
			clientEventsToSend <- &Event{
				PlayerId:   me.Id,
				Kind:       "started",
				InnerEvent: StartedEvent{},
			}
		} else if newPixel.Center != lastMousePos {
			clientEventsToSend <- &Event{
				PlayerId: me.Id,
				Kind:     "drawing",
				InnerEvent: DrawingEvent{
					Pixel: &newPixel,
				},
			}
			lastMousePos = mousePos
		}
	} else {
		if me.Drawing {
			clientEventsToSend <- &Event{
				PlayerId:   me.Id,
				Kind:       "done",
				InnerEvent: DoneEvent{},
			}
		}
	}
}

func HandleColorPicker() {
	if rl.IsKeyPressed(rl.KeyC) {
		lastMousePos = rl.GetMousePosition()
	}

	if rl.IsKeyDown(rl.KeyC) {
		colorPicker.Center = lastMousePos
		colorPicker.Draw()
	}

	if rl.IsKeyReleased(rl.KeyC) {
		if colorPicker.IsHovering() {
			currentMousePosition := rl.GetMousePosition()
			dx := float64(currentMousePosition.X - colorPicker.Center.X)
			dy := float64(currentMousePosition.Y - colorPicker.Center.Y)
			angle := math.Atan2(dy, dx) * (180 / math.Pi)
			if angle < 0 {
				// 0, 360
				angle += 360
			}
			sectorSize := 360 / len(colorPicker.Colors)
			index := int(angle) / sectorSize
			selectedColor = colorPicker.Colors[index]
		}
	}
}

// client utils

func GetCache(player *Player, index int) *Cache {
	if len(player.CachedScribbles) == 0 {
		return nil
	}

	cache := player.CachedScribbles[index]

	if cache.Empty {
		texture := rl.LoadRenderTexture(width, height)
		cache.RenderTexture2D = &texture
		cache.Empty = false
	}

	return cache
}

func Interpolate(from float32, to float32, percent float32) float32 {
	difference := to - from
	return from + (difference * percent)
}

func IsMouseClickOnScribble(clickPositon rl.Vector2) {
	// TO DO:
	// implement spatial hashing to increase perfomance
	
	// change Scribble to have it's own type and store min/max info 
	// when receiving the pixels
	
	for _, player := range players {
		for _, pixelArray := range player.Scribbles {
			for i := range len(pixelArray) - 1 {
				x1 := pixelArray[i].Center.X
				x2 := pixelArray[i+1].Center.X
				y1 := pixelArray[i].Center.Y
				y2 := pixelArray[i+1].Center.Y

				for j := range 100 {
					k := float32(j) / 100.0
					xa := Interpolate(x1, x2, k)
					ya := Interpolate(y1, y2, k)

					radius := pixelArray[0].Radius
					xHoveringLine := clickPositon.X > xa-radius && clickPositon.X < xa+radius
					yHoveringLine := clickPositon.Y > ya-radius && clickPositon.Y < ya+radius
					hoveringLine := xHoveringLine && yHoveringLine
					if hoveringLine {
						go FindBoundingBox(pixelArray)
						return
					}

					xInsideBoundingBox := selectedBoundingBox != nil && clickPositon.X > selectedBoundingBox.BoundingBox.Min.X && clickPositon.X < selectedBoundingBox.BoundingBox.Max.X
					yInsideBoundingBox := selectedBoundingBox != nil && clickPositon.Y > selectedBoundingBox.BoundingBox.Min.Y && clickPositon.Y < selectedBoundingBox.BoundingBox.Max.Y
					insideBoundingBox := xInsideBoundingBox && yInsideBoundingBox
					if insideBoundingBox {
						fmt.Println("Inside")
						return
					}

					if !insideBoundingBox {
						selectedBoundingBox = nil
					}
				}
			}
		}
	}
}

func FindBoundingBox(scribble []*Pixel) {
	if len(scribble) < 2 {
		return
	}

	var min = rl.NewVector3(float32(width), float32(height), 0)
	var max = rl.NewVector3(-1, -1, -1)

	for _, pixel := range scribble {
		if min.X > pixel.Center.X {
			min.X = pixel.Center.X
		}

		if min.Y > pixel.Center.Y {
			min.Y = pixel.Center.Y
		}

		if max.X < pixel.Center.X {
			max.X = pixel.Center.X
		}

		if max.Y < pixel.Center.Y {
			max.Y = pixel.Center.Y
		}
	}

	// adjusting padding
	min.X -= 10
	max.X += 10
	min.Y -= 10
	max.Y += 10

	selectedBoundingBox = &BoundingBox{Scribble: scribble, BoundingBox: rl.BoundingBox{Min: min, Max: max}}
}
