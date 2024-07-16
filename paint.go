package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	width           int32 = 1600
	height          int32 = 900
	buffer_to_paint []*[]*Pixel
	deleted            []*[]*Pixel
	last_mouse_pos    rl.Vector2
	mouse_first_click bool = true
	changed           bool = false
	last_pixel        *Pixel
	pixel_size        float32 = 10.0
)

type Pixel struct {
	center rl.Vector2
	radius float32
	color  rl.Color
}

func main() {
	rl.InitWindow(width, height, "Paint")
	rl.SetWindowState(rl.FlagVsyncHint)
	// rl.SetTargetFPS(0)

	defer rl.CloseWindow()

	target := rl.LoadRenderTexture(width, height)

	for !rl.WindowShouldClose() {

		HandlePainting(target)
		HandleInput()
		DrawIfChanged(target)

		rl.BeginDrawing()
		rl.ClearBackground(rl.White)
		rl.DrawTextureRec(target.Texture, rl.Rectangle{X: 0, Y: 0, Width: float32(target.Texture.Width), Height: -float32(target.Texture.Height)}, rl.Vector2{X: 0, Y: 0}, rl.White)
		rl.DrawCircleLines(rl.GetMouseX(), rl.GetMouseY(), pixel_size, rl.Black)
		rl.DrawFPS(width-200, 20)
		rl.EndDrawing()

	}
}

// Draws each pixel in the Texture layer after a change occurs
func DrawIfChanged(target rl.RenderTexture2D) {
	if changed {
		rl.BeginTextureMode(target)
		rl.ClearBackground(rl.White)
		var last_pixel_loop *Pixel
		for _, pixel_array := range buffer_to_paint {
			for i, pixel := range *pixel_array {
				rl.DrawCircleV(pixel.center, pixel.radius, pixel.color)
				// Draws a line between the last and newest pixel
				if i > 0 && last_pixel_loop != nil {
					rl.DrawLineEx(pixel.center, last_pixel_loop.center, pixel.radius*2, rl.Black)
				}
				last_pixel_loop = pixel
			}
		}

		rl.EndTextureMode()
		changed = false
	}
}

func HandlePainting(target rl.RenderTexture2D) {
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		mouse_position := rl.GetMousePosition()
		new_pixel := Pixel{mouse_position, pixel_size, rl.Black}

		// If it's the first click, creates a new list of pixels and append to buffer_to_paint
		if mouse_first_click {
			pixel_buffer := []*Pixel{}
			buffer_to_paint = append(buffer_to_paint, &pixel_buffer)
			mouse_first_click = false
		}

		if new_pixel.center != last_mouse_pos {
			len_buffer_to_paint := len(buffer_to_paint)
			*buffer_to_paint[len_buffer_to_paint-1] = append(*buffer_to_paint[len_buffer_to_paint-1], &new_pixel)
			last_mouse_pos = mouse_position

			rl.BeginTextureMode(target)
			rl.DrawCircleV(new_pixel.center, new_pixel.radius, new_pixel.color)
			if last_pixel != nil {
				rl.DrawLineEx(new_pixel.center, last_pixel.center, new_pixel.radius*2, rl.Black)
			}
			rl.EndTextureMode()

			last_pixel = &new_pixel
		}

	} else {
		// if !mouse_first_click {
		// 	changed = true
		// }
		mouse_first_click = true
		last_pixel = nil
	}
}

func HandleInput() {
	if rl.IsKeyPressed(rl.KeyS) {
		fmt.Println("")
		for _, pixel_array := range buffer_to_paint {
			fmt.Println(pixel_array)
		}
	}
	
	if rl.IsKeyPressed(rl.KeyR) {
		buffer_to_paint = []*[]*Pixel{}
		changed = true
	}
	if rl.IsKeyPressed(rl.KeyEqual) {
		pixel_size++
	}
	if rl.IsKeyPressed(rl.KeyMinus) {
		if pixel_size > 0 {
			pixel_size--
		}
	}

	// Undo the last paint
	if rl.IsKeyPressed(rl.KeyU) {
		if len(buffer_to_paint) > 0 {
			deleted_pixels := buffer_to_paint[len(buffer_to_paint)-1]
			buffer_to_paint = buffer_to_paint[:len(buffer_to_paint)-1]
			deleted = append(deleted, deleted_pixels)
			changed = true
		}
	}

	// Redo the last paint
	if rl.IsKeyPressed(rl.KeyI) {
		if len(deleted) > 0 {
			new_pixels := deleted[len(deleted)-1]
			deleted = deleted[:len(deleted)-1]
			buffer_to_paint = append(buffer_to_paint, new_pixels)
			changed = true
		}
	}
}