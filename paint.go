package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	board := NewBoard()

	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(board.Width, board.Height, "Paint")

	// might cause trouble
	rl.SetWindowState(rl.FlagWindowAlwaysRun)

	// rl.SetWindowState(rl.FlagVsyncHint)
	rl.SetTargetFPS(board.FPS)

	defer rl.CloseWindow()

	target := rl.LoadRenderTexture(board.FPS, board.Height)
	halfScreenW := rl.GetScreenWidth() / 2
	halfScreenH := rl.GetScreenHeight() / 2
	buttonWidth := 400
	buttonHeight := 100
	serverButton := NewButton(halfScreenW, halfScreenH-(buttonHeight/2)-20, buttonWidth, buttonHeight, rl.Black, "Host", 40)
	clientButton := NewButton(halfScreenW, halfScreenH+(buttonHeight/2)+20, buttonWidth, buttonHeight, rl.Black, "Enter", 40)

	for !rl.WindowShouldClose() {
		board.FrameCount++
		// ui mode: choose if you're gonna host or enter
		// else: start paint screen
		rl.BeginDrawing()
		if board.UiMode {
			board.DrawUIMode(serverButton, clientButton)
		}

		if !board.UiMode {
			board.Input()
			board.Draw(target)
		}

		if board.FrameCount == board.FPS/board.FrameSpeed {
			board.FrameCount = 0
		}
		rl.EndDrawing()
	}

	// close the application window so they left
	board.Client.EnqueueEvent(board.Me.Id, "left", LeftEvent{})
	board.Wg.Wait()
}

func (b *Board) Input() {
	b.HandlePainting()
	b.HandleColorPicker()

	if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
		go b.IsMouseClickOnScribble(rl.GetMousePosition())
	}

	if rl.IsKeyDown(rl.KeyEqual) && b.FrameCount == b.FPS/b.FrameSpeed {
		b.PixelSize++
	}

	if rl.IsKeyDown(rl.KeyMinus) && b.PixelSize > 1 && b.FrameCount == b.FPS/b.FrameSpeed {
		b.PixelSize--
	}

	if rl.IsKeyPressed(rl.KeyU) {
		b.Client.EnqueueEvent(b.Me.Id, "undo", UndoEvent{})
	}

	if rl.IsKeyPressed(rl.KeyR) {
		b.Client.EnqueueEvent(b.Me.Id, "redo", RedoEvent{})
	}

	// debug purposes
	if rl.IsKeyPressed(rl.KeyD) {
		fmt.Println("SCRIBBLE", b.Me.Scribbles[len(b.Me.Scribbles)-1].BoundingBox)
	}

}

// Draw

func (b *Board) DrawUIMode(serverButton Button, clientButton Button) {
	rl.ClearBackground(rl.White)
	serverButton.Draw()
	serverButton.Click(func() {
		go StartServer()
		go b.StartClient()
		b.UiMode = false
	})

	clientButton.Draw()
	clientButton.Click(func() {
		go b.StartClient()
		b.UiMode = false
	})
}

func (b *Board) Draw(target rl.RenderTexture2D) {
	rl.ClearBackground(rl.White)

	b.DrawBoard()

	// rl.BeginBlendMode(rl.BlendAlpha)
	b.DrawCache()
	// rl.EndBlendMode()

	if b.SelectedBoundingBox != nil {
		b.SelectedBoundingBox.Draw()
	}

	if b.ColorPickerOpened {
		b.ColorPicker.Draw()
	}

	rl.DrawCircleLines(rl.GetMouseX(), rl.GetMouseY(), b.PixelSize, rl.Black)
	rl.DrawFPS(b.Width-200, 20)

	mouseXText := fmt.Sprintf("Mouse X: %d", int(rl.GetMousePosition().X))
	mouseYText := fmt.Sprintf("Mouse Y: %d", int(rl.GetMousePosition().Y))
	rl.DrawText(mouseXText, b.Width-200, 40, 20, rl.Black)
	rl.DrawText(mouseYText, b.Width-200, 60, 20, rl.Black)

	pencilSizeText := fmt.Sprintf("Pencil size: %d", int(b.PixelSize))
	rl.DrawText(pencilSizeText, 10, 10, 20, b.CONFIG_COLOR)

	rl.DrawText("Selected color: ", 10, 40, 20, b.CONFIG_COLOR)
	rl.DrawCircle(180, 50, 10, b.SelectedColor)
}

func (b *Board) DrawBoard() {
	if b.Changed {
		for _, player := range b.Client.Players {
			if player.Drawing {
				currentlyDrawingArray := Last(player.Scribbles)
				cache := b.GetCache(player, len(player.CachedScribbles)-1)
				if cache == nil {
					panic("Cache nil")
				}
				texture := cache.RenderTexture2D
				DrawScribble(currentlyDrawingArray.Pixels, *texture)
			} else if player.JustJoined {
				for i := range len(player.Scribbles) {
					scribble := player.Scribbles[i]

					// to remember:
					// I initialized cache by iterating over scribbles,
					// so now cache array of each player is the same size as scribbles.
					cache := b.GetCache(player, i)
					if cache == nil {
						panic("Cache nil")
					}
					texture := cache.RenderTexture2D
					DrawScribble(scribble.Pixels, *texture)
					cache.Drawing = false
				}
				player.JustJoined = false
			}
		}
	}
	b.Changed = false
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

func (b *Board) DrawCache() {
	for _, cache := range b.Client.CacheArray {
		if !cache.Empty {
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

func (b *Board) HandlePainting() {
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		mousePos := rl.GetMousePosition()
		fmt.Println("pixelSize", b.PixelSize)
		newPixel := Pixel{
			mousePos,
			b.PixelSize,
			b.SelectedColor,
		}

		// avoid send redundant events, otherwise, drawing will be true
		// as long as the player hold the mouse button
		// so it would send these events again
		if !b.Me.Drawing {
			b.Client.EnqueueEvent(b.Me.Id, "started", StartedEvent{})
		} else if newPixel.Center != b.LastMousePos {
			b.Client.EnqueueEvent(b.Me.Id, "drawing", DrawingEvent{Pixel: &newPixel})
			b.LastMousePos = mousePos
		}
	} else {
		if b.Me.Drawing {
			b.Client.EnqueueEvent(b.Me.Id, "done", DoneEvent{})
		}
	}
}

func (b *Board) HandleColorPicker() {
	if rl.IsKeyPressed(rl.KeyC) {
		b.ColorPicker.LastMousePositionBeforeClick = rl.GetMousePosition()
	}

	if rl.IsKeyDown(rl.KeyC) {
		b.ColorPicker.Center = b.ColorPicker.LastMousePositionBeforeClick
		b.ColorPickerOpened = true
	}

	if rl.IsKeyReleased(rl.KeyC) {
		if b.ColorPicker.IsHovering() {
			currentMousePosition := rl.GetMousePosition()
			dx := float64(currentMousePosition.X - b.ColorPicker.Center.X)
			dy := float64(currentMousePosition.Y - b.ColorPicker.Center.Y)
			angle := math.Atan2(dy, dx) * (180 / math.Pi)
			if angle < 0 {
				// 0, 360
				angle += 360
			}
			sectorSize := 360 / len(b.ColorPicker.Colors)
			index := int(angle) / sectorSize
			b.SelectedColor = b.ColorPicker.Colors[index]
		}
		b.ColorPickerOpened = false
	}
}

// client utils

func (b *Board) GetCache(player *Player, index int) *Cache {
	if len(player.CachedScribbles) == 0 {
		return nil
	}

	cache := player.CachedScribbles[index]

	if cache.Empty {
		texture := rl.LoadRenderTexture(b.Width, b.Height)
		cache.RenderTexture2D = &texture
		cache.Empty = false
	}

	return cache
}

func Interpolate(from float32, to float32, percent float32) float32 {
	difference := to - from
	return from + (difference * percent)
}

func (b *Board) IsMouseClickOnScribble(clickPositon rl.Vector2) {
	// TO DO:
	// Implement spatial hashing to increase perfomance

	// Create a client pixel type that has a reference to it's parent.
	// That way I can find the pixel using spatial hashing and find it's parent

	for _, player := range b.Client.Players {
		for _, scribble := range player.Scribbles {
			for i := range len(scribble.Pixels) - 1 {
				x1 := scribble.Pixels[i].Center.X
				x2 := scribble.Pixels[i+1].Center.X
				y1 := scribble.Pixels[i].Center.Y
				y2 := scribble.Pixels[i+1].Center.Y

				for j := range 100 {
					k := float32(j) / 100.0
					xa := Interpolate(x1, x2, k)
					ya := Interpolate(y1, y2, k)

					radius := scribble.Pixels[0].Radius
					xHoveringLine := clickPositon.X >= xa-radius && clickPositon.X <= xa+radius
					yHoveringLine := clickPositon.Y >= ya-radius && clickPositon.Y <= ya+radius
					hoveringLine := xHoveringLine && yHoveringLine
					if hoveringLine {
						// adjusting padding
						scribble.BoundingBox.Min.X -= 10 + LINE_THICK
						scribble.BoundingBox.Max.X += 10 + LINE_THICK
						scribble.BoundingBox.Min.Y -= 10 + LINE_THICK
						scribble.BoundingBox.Max.Y += 10 + LINE_THICK

						b.SelectedBoundingBox = &BoundingBox{
							Scribble:  &scribble,
							Min:       scribble.BoundingBox.Min,
							Max:       scribble.BoundingBox.Max,
							LineThick: 5,
						}
					}

					xInsideBoundingBox := b.SelectedBoundingBox != nil && clickPositon.X > b.SelectedBoundingBox.Min.X && clickPositon.X < b.SelectedBoundingBox.Max.X
					yInsideBoundingBox := b.SelectedBoundingBox != nil && clickPositon.Y > b.SelectedBoundingBox.Min.Y && clickPositon.Y < b.SelectedBoundingBox.Max.Y
					insideBoundingBox := xInsideBoundingBox && yInsideBoundingBox
					if insideBoundingBox {
						fmt.Println("Inside")
						return
					}

					if !insideBoundingBox {
						b.SelectedBoundingBox = nil
					}
				}
			}
		}
	}
}

func GetMinAndMax(min rl.Vector3, max rl.Vector3, pixel *Pixel) (rl.Vector3, rl.Vector3) {
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

	return min, max
}
