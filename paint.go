package main

import (
	"fmt"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	width        int32 = 1600
	height       int32 = 900
	lastMousePos rl.Vector2
	changed      bool    = false
	pixelSize    float32 = 10.0
	FPS          int32   = 60
	frame        int32   = 0
	frameSpeed   int32   = 30
	wg           sync.WaitGroup
	uiMode       bool = true

	CONFIG_COLOR rl.Color = rl.Magenta
)

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

	rl.DrawCircleLines(rl.GetMouseX(), rl.GetMouseY(), pixelSize, rl.Black)
	rl.DrawFPS(width-200, 20)

	pencilSizeText := fmt.Sprintf("Pencil size: %d", int(pixelSize))
	rl.DrawText(pencilSizeText, 10, 10, 20, CONFIG_COLOR)

	rl.EndDrawing()
}

func DrawIfChanged() {
	if changed {
		for _, player := range players {
			if player.Drawing {
				currentlyDrawingArray := player.Scribbles[len(player.Scribbles)-1]
				cache := GetLastCache(player)
				texture := cache.RenderTexture2D
				rl.BeginTextureMode(*texture)
				rl.ClearBackground(rl.Blank)
				DrawScribble(currentlyDrawingArray)
				rl.EndTextureMode()
			} 
		}
	}
}

func DrawScribble(scribble []*Pixel) {
	var lastPixelLoop *Pixel
	for i, pixel := range scribble {
		rl.DrawCircleV(pixel.Center, pixel.Radius, pixel.Color)
		// Draws a line between the last and newest pixel
		if i > 0 && lastPixelLoop != nil {
			rl.DrawLineEx(pixel.Center, lastPixelLoop.Center, pixel.Radius*2, rl.Black)
		}
		lastPixelLoop = pixel
	}
	lastPixelLoop = &Pixel{}
}

func DrawCache() {
	for _, player := range players {
		for i, cache := range player.CachedScribbles {
			// draw textures that are ready to be drawn or the last texture the player is currently drawing on
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
		newPixel := Pixel{mousePos, pixelSize, rl.Black}

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






// client utils

func GetLastCache(player *Player) *Cache {
	cache := player.CachedScribbles[len(player.CachedScribbles)-1]

	if cache.Empty {
		texture := rl.LoadRenderTexture(width, height)
		cache.RenderTexture2D = &texture
		cache.Empty = false
	}

	return cache
}
