package common

import (
	"context"
	"fmt"
	"maps"
	"net/http"
)

type BtnSz int

const (
	BtnSmall BtnSz = iota
	BtnLarge
	BtnLong
)

func (f BtnSz) String() string {
	switch f {
	case BtnSmall:
		return "btn-sm"
	case BtnLarge:
		return "btn-lg"
	case BtnLong:
		return "btn-long"
	default:
		return fmt.Sprintf("btn-size--%d-", f)
	}
}

type BtnColor int

const (
	BtnPrimary BtnColor = iota
	BtnSecondary
	BtnTertiary
	BtnDanger
	BtnOutline
	BtnGhost
	BtnLink
)

func (f BtnColor) String() string {
	switch f {
	case BtnPrimary:
		return "btn-primary"
	case BtnSecondary:
		return "btn-secondary"
	case BtnTertiary:
		return "btn-tertiary"
	case BtnDanger:
		return "btn-danger"
	case BtnOutline:
		return "btn-outline"
	case BtnGhost:
		return "btn-ghost"
	case BtnLink:
		return "btn-link"
	default:
		return fmt.Sprintf("btn-color--%d-", f)
	}
}

type Button struct {
	Id           string
	Content      string
	Size         BtnSz
	Color        BtnColor
	Disabled     bool
	ExtraClasses map[string]bool
}

func BtnChain(id string, content string) Button {
	return Button{Id: id, Content: content}
}

func Btn(id string, content string, size BtnSz, color BtnColor, extras ...string) Button {
	classes := map[string]bool{}
	for _, extra := range extras {
		classes[extra] = true
	}
	return Button{
		Id:           id,
		Content:      content,
		Size:         size,
		Color:        color,
		ExtraClasses: classes,
	}
}

func (b Button) SetSize(sz BtnSz) Button {
	b.Size = sz
	return b
}

func (b Button) SetColor(c BtnColor) Button {
	b.Color = c
	return b
}

func (b Button) Disable() Button {
	b.Disabled = true
	return b
}

func (b Button) Enable() Button {
	b.Disabled = false
	return b
}

func (b Button) AddClass(class string) Button {
	if b.ExtraClasses == nil {
		b.ExtraClasses = map[string]bool{}
	}
	b.ExtraClasses[class] = true
	return b
}

func (b Button) DropClass(class string) Button {
	if b.ExtraClasses == nil {
		b.ExtraClasses = map[string]bool{}
	}
	b.ExtraClasses[class] = false
	return b
}

func (b Button) Classes() map[string]bool {
	ret := map[string]bool{}
	maps.Copy(ret, b.ExtraClasses)
	ret[b.Color.String()] = true
	ret[b.Size.String()] = true
	return ret
}

func (b Button) Render(ctx context.Context, w http.ResponseWriter) error {
	component := b.render()
	return component.Render(ctx, w)
}

type ButtonIcon struct {
	Id       string
	Icon     string
	Color    BtnColor
	Selected bool
	Disabled bool
}

func BtnIconChain(id string, name string) ButtonIcon {
	return ButtonIcon{
		Id:    id,
		Icon:  name,
		Color: BtnPrimary,
	}
}

func BtnIcon(id string, name string, color BtnColor, selected bool, disabled bool) ButtonIcon {
	return ButtonIcon{
		Id:       id,
		Icon:     name,
		Color:    color,
		Selected: selected,
		Disabled: disabled,
	}
}

func (b ButtonIcon) SetColor(color BtnColor) ButtonIcon {
	b.Color = color
	return b
}

func (b ButtonIcon) Select() ButtonIcon {
	b.Selected = true
	return b
}

func (b ButtonIcon) Deselect() ButtonIcon {
	b.Selected = false
	return b
}

func (b ButtonIcon) Disable() ButtonIcon {
	b.Disabled = true
	return b
}

func (b ButtonIcon) Enable() ButtonIcon {
	b.Disabled = false
	return b
}

func (b ButtonIcon) Button() Button {
	ret := Button{
		Color:    b.Color,
		Disabled: b.Disabled,
		Size:     BtnSmall,
	}
	if !b.Selected {
		ret.Color = BtnGhost
	} else {
		ret.AddClass("rounded-lg")
	}
	return ret
}

func (b ButtonIcon) Render(ctx context.Context, w http.ResponseWriter) error {
	component := b.render()
	return component.Render(ctx, w)
}
