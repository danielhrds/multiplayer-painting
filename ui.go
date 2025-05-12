package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Button struct {
	Rectangle       rl.Rectangle
	BackgroundColor rl.Color
	Text            string
	FontSize        int32
}

func (b *Button) Draw() {
	rl.DrawRectangle(b.Rectangle.ToInt32().X, b.Rectangle.ToInt32().Y, b.Rectangle.ToInt32().Width, b.Rectangle.ToInt32().Height, b.BackgroundColor)
	textRec := rl.MeasureTextEx(rl.GetFontDefault(), b.Text, float32(b.FontSize), 0)
	rl.DrawText(b.Text, (b.Rectangle.ToInt32().X+b.Rectangle.ToInt32().Width/2)-int32(textRec.X)/2, b.Rectangle.ToInt32().Y+(b.Rectangle.ToInt32().Height/2)-int32(textRec.Y)/2, 40, rl.White)
}

func (b *Button) IsHovering() bool {
	mousePosition := rl.GetMousePosition()
	minX := b.Rectangle.X
	maxX := b.Rectangle.X + b.Rectangle.Width
	minY := b.Rectangle.Y
	maxY := b.Rectangle.Y + b.Rectangle.Height
	isHovering := mousePosition.X >= minX && mousePosition.X <= maxX && mousePosition.Y >= minY && mousePosition.Y <= maxY
	return isHovering
}

type Callback func()

func (b *Button) Click(callback Callback) {
	if b.IsHovering() {
		if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			callback()
		}
	}
}

func NewButton(x int, y int, width int, height int, backgroundColor rl.Color, text string, fontSize int32) Button {
	return Button{Rectangle: rl.Rectangle{
		X:      float32(x) - (float32(width) / 2),
		Y:      float32(y) - (float32(height) / 2),
		Width:  float32(width),
		Height: float32(height),
	},
		BackgroundColor: backgroundColor,
		Text:            text,
		FontSize:        fontSize,
	}

}

type ColorPicker struct {
	Colors []rl.Color
	Center rl.Vector2
	Radius float32
}

func (c *ColorPicker) Draw() {
	spacing := 360 / len(c.Colors)
	for i, color := range c.Colors {
		start := float32(i * spacing)
		end := float32((i * spacing) + spacing)
		rl.DrawCircleSector(c.Center, c.Radius, start, end, 15, color)
	}
}

func (c *ColorPicker) IsHovering() bool {
	mousePosition := rl.GetMousePosition()
	p1 := math.Pow(float64(mousePosition.X)-float64(c.Center.X), 2)
	p2 := math.Pow((float64(mousePosition.Y) - float64(c.Center.Y)), 2)
	distanceFromCenter := math.Sqrt(p1 + p2)
	return distanceFromCenter <= float64(c.Radius)
}

type BoundingBox struct {
	BoundingBox rl.BoundingBox
	LineThick   float32
	Scribble    []*Pixel
}

func (b *BoundingBox) Draw() {
	rl.DrawRectangleLinesEx(
		rl.Rectangle{
			X:      b.BoundingBox.Min.X,
			Y:      b.BoundingBox.Min.Y,
			Width:  b.BoundingBox.Max.X - b.BoundingBox.Min.X,
			Height: b.BoundingBox.Max.Y - b.BoundingBox.Min.Y,
		},
		b.LineThick,
		CONFIG_COLOR,
	)
}
