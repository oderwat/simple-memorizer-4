package components

import "github.com/maxence-charriere/go-app/v9/pkg/app"

type Exercises struct {
	app.Compo
}

// The Render method is where the component appearance is defined.
func (h *Exercises) Render() app.UI {
	return app.Div().Body(
		app.P().Body(
			app.A().Href("/").Text("Home"),
			app.Text(" | "),
			app.A().Href("/exercises").Text("Exercises"),
		),
	)
}
