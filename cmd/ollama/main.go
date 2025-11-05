package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ollama/ollama/api"
)

func main() {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	model := "deepseek-r1:7b"
	ctx := context.Background()
	pullRequest := &api.PullRequest{
		Model: model,
	}
	progressFunc := func(resp api.ProgressResponse) error {
		fmt.Printf("Progress: status=%v, total=%v, completed=%v\n", resp.Status, resp.Total, resp.Completed)
		return nil
	}

	err = client.Pull(ctx, pullRequest, progressFunc)
	if err != nil {
		log.Fatal(err)
	}
	menuData := `
	<item>OC Plate | Tiroler Gröstl | Frühlingsgemüse | Kartoffeln | Pilze | Kräuter-Mandelcreme</item>
	<item>Claudis Hackbraten | Karotten-Blumenkohlgemüse | Kartoffellstampf | Rahmsauce</item>
	<item>Burgunderbraten | Rind | Muskatspätzle | Blaukraut | Preiselbeeren</item>
	<item>Haselnusspuddin</item>
	<item>Pizza Coca de Verdura  | Aubergine | Champignon | Paprika | Zucchini | Mozzarella</item>
	<item>TEMPURA Bowl | Gebackenes Gemüse | Shiitake | Pilze | Salatmix | Miso Mayo | Limetten- Joghurt Dressing</item>
`
	messages := []api.Message{
		{
			Role:    "system",
			Content: "Du bist ein Menü-Ansager. Deine Aufgabe ist es, eine strukturierte Liste von Menüpunkten in einen einzigen, natürlichen, fließenden Absatz gesprochenen Text auf Deutsch umzuwandeln\n\n- Wandle Formatierungszeichen wie '|' in gesprochene Wörter um (z. B. 'mit', 'und', 'oder', 'dazu').\n- Kombiniere Gerichte und ihre Zutaten zu beschreibenden Sätzen.\n- Ignoriere Preise oder nicht gesprochene Codes (wie 'OC Plate', es sei denn, es ist Teil des Gerichtsnamens).\n- Die Antwort darf *nur* der fertige, saubere Absatz auf Deutsch sein, ohne jegliche Einleitung.",
		},
		{
			Role:    "user",
			Content: "Bitte wandle die folgenden Menüpunkte in eine einzige, sprechbare Ansage um:\n\n" + menuData,
		},
	}

	chatRequest := &api.ChatRequest{
		Model:    model,
		Messages: messages,
	}

	respFunc := func(resp api.ChatResponse) error {
		fmt.Print(resp.Message.Content)
		return nil
	}

	err = client.Chat(ctx, chatRequest, respFunc)
	if err != nil {
		log.Fatal(err)
	}
}
