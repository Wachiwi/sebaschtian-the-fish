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

	model := "tinyllama"
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
"HOT \u0026 COLD | Winterliche Maronensuppe \u0026 The SMALL Green ",
"OC Plate | Tiroler Gröstl | Frühlingsgemüse | Kartoffeln | Pilze | Kräuter-Mandelcreme ",
"Claudis Hackbraten | Karotten-Blumenkohlgemüse | Kartoffellstampf | Rahmsauce ",
"Burgunderbraten | Rind | Muskatspätzle | Blaukraut | Preiselbeeren ",
"Haselnusspudding   ",
"Salatbuffet \"The BIG Green\" | The SMALL Green 4,35 €  ",
"Pizza Coca de Verdura  | Aubergine | Champignon | Paprika | Zucchini | Mozzarella ",
"TEMPURA Bowl | Gebackenes Gemüse | Shiitake | Pilze | Salatmix | Miso Mayo | Limetten- Joghurt Dressing "
`
	messages := []api.Message{
		{
			Role:    "system",
			Content: "Du bist ein Menü-Ansager. Deine Aufgabe ist es, eine strukturierte Liste von Menüpunkten in einen einzigen, natürlichen, fließenden Absatz gesprochenen Text auf Deutsch umzuwandeln, der für ein Text-to-Speech (TTS)-System geeignet ist.\n\n- Wandle Formatierungszeichen wie '|' in gesprochene Wörter um (z. B. 'mit', 'und', 'oder', 'dazu').\n- Korrigiere Zeichencodes wie '\\u0026' (ersetze es durch 'und').\n- Kombiniere Gerichte und ihre Zutaten zu beschreibenden Sätzen.\n- Ignoriere Preise oder nicht gesprochene Codes (wie 'OC Plate', es sei denn, es ist Teil des Gerichtsnamens).\n- Die Antwort darf *nur* der fertige, saubere Absatz auf Deutsch sein, ohne jegliche Einleitung.",
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
