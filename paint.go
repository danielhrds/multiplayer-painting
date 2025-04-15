package main

import (
	"fmt"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	width             int32 = 1600
	height            int32 = 900
	buffer_to_paint   []*[]*Pixel
	last_mouse_pos    rl.Vector2
	changed           bool = false
	last_pixel        *Pixel
	pixelSize        float32 = 10.0
	FPS         			int32 = 60
	wg 								sync.WaitGroup
	uiMode 						bool = true
	choose 						string
	initiated					bool = false
)

func main() {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(width, height, "Paint")
	rl.SetWindowState(rl.FlagVsyncHint)
	rl.SetTargetFPS(FPS)

	defer rl.CloseWindow()

	target := rl.LoadRenderTexture(width, height)
	halfScreenW := rl.GetScreenWidth() / 2
	halfScreenH :=  rl.GetScreenHeight() / 2
	buttonWidth := 400
	buttonHeight := 100
	serverButton := NewButton(halfScreenW, halfScreenH - (buttonHeight / 2) - 20, buttonWidth, buttonHeight, rl.Black, "Host", 40)
	clientButton := NewButton(halfScreenW, halfScreenH + (buttonHeight / 2) + 20, buttonWidth, buttonHeight, rl.Black, "Enter", 40)

	for !rl.WindowShouldClose() {
		// ui mode: choose if you're gonna host or enter
		// initiated: init server/client based on your choice
		// else: start paint screen
		if uiMode {
			rl.BeginDrawing()
			rl.ClearBackground(rl.White)
			serverButton.Draw()
			serverButton.Click(func() {
				choose = "host"
				uiMode = false
			})
			clientButton.Draw()
			clientButton.Click(func() {
				choose = "client"
				uiMode = false
			})
			rl.EndDrawing()
		} else if !initiated {
				if choose == "host" {
					go StartServer()
					go StartClient()
				} else {
					go StartClient()
				}
				initiated = true
		} else {
			HandlePainting(target)
			HandleInput()
			DrawIfChanged(target)
			
			rl.BeginDrawing()
			rl.ClearBackground(rl.White)
			rl.DrawTextureRec(target.Texture, rl.Rectangle{X: 0, Y: 0, Width: float32(target.Texture.Width), Height: -float32(target.Texture.Height)}, rl.Vector2{X: 0, Y: 0}, rl.White)
			rl.DrawCircleLines(rl.GetMouseX(), rl.GetMouseY(), pixelSize, rl.Black)
			rl.DrawFPS(width-200, 20)
			rl.EndDrawing()
		}
	}

	// close the application window so they left
	clientEventsToSend <- &Event{
		PlayerId: me.Id,
		Kind: "left",
		InnerEvent: LeftEvent{},
	}
	wg.Wait()
}

func Init() {
	
}

// Draws each pixel in the Texture layer after a change occurs
func DrawIfChanged(target rl.RenderTexture2D) {
	rl.BeginTextureMode(target)
	
	if changed {
		rl.ClearBackground(rl.White)
		var lastPixelLoop *Pixel
		for _, client := range players {
			for _, pixelArray := range client.Scribbles {
				for i, pixel := range pixelArray {
					rl.DrawCircleV(pixel.Center, pixel.Radius, pixel.Color)
					// Draws a line between the last and newest pixel
					if i > 0 && lastPixelLoop != nil {
						rl.DrawLineEx(pixel.Center, lastPixelLoop.Center, pixel.Radius*2, rl.Black)
					}
					lastPixelLoop = pixel
				}
			}
			lastPixelLoop = &Pixel{}
		}
	}
	

	rl.EndTextureMode()
	changed = false
}

func HandlePainting(target rl.RenderTexture2D) {
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		mouse_position := rl.GetMousePosition()
		new_pixel := Pixel{mouse_position, pixelSize, rl.Black}

		// avoid send redundant events, otherwise, drawing will be true
		// as long as the player hold the mouse button
		// so it would send again these events
		if !me.Drawing {
			clientEventsToSend <- &Event{
				PlayerId: me.Id,
				Kind: "started",
				InnerEvent: StartedEvent{},
			}
		} else {
			if new_pixel.Center != last_mouse_pos {
				clientEventsToSend <- &Event{
					PlayerId: me.Id,
					Kind: "drawing",
					InnerEvent: DrawingEvent{
						Pixel: &new_pixel,
					},
				}
			}
		}	

		if new_pixel.Center != last_mouse_pos {
			last_mouse_pos = mouse_position
			last_pixel = &new_pixel
		}

	} else {
		if me.Drawing {
			clientEventsToSend <- &Event{
				PlayerId: me.Id,
				Kind: "done",
				InnerEvent: DoneEvent{},
			}
		}
		last_pixel = nil
	}
}

func HandleInput() {
	if rl.IsKeyPressed(rl.KeyS) {
		for _, pixel_array := range buffer_to_paint {
			fmt.Println(pixel_array)
		}
	}

	// if rl.IsKeyPressed(rl.KeyR) {
	// 	clear = true
	// }
	
	if rl.IsKeyPressed(rl.KeyEqual) {
		pixelSize++
	}
	
	if rl.IsKeyPressed(rl.KeyMinus) {
		if pixelSize > 0 {
			pixelSize--
		}
	}

	// Undo the last paint
	if rl.IsKeyPressed(rl.KeyU) {
		clientEventsToSend <- &Event{
			PlayerId: me.Id,
			Kind: "undo",
			InnerEvent: UndoEvent{},
		}
	}

	// Redo the last paint
	if rl.IsKeyPressed(rl.KeyR) {
		clientEventsToSend <- &Event{
			PlayerId: me.Id,
			Kind: "redo",
			InnerEvent: RedoEvent{},
		}
	}
}
