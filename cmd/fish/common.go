package main

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
)

type Phrase struct {
	Text   string
	Weight int
}

func getWeightedRandomPhrase() string {
	now := time.Now()
	hour := now.Hour()
	day := now.Day()

	phrases := []Phrase{
		{Text: "Bald ist Mittag", Weight: 10},
		{Text: "Bald ist Feierabend", Weight: 10},
		{Text: "Es ist spät, Zeit für Magic!", Weight: 10},
		{Text: "Feierabend, wie das duftet. Kräftig, deftig, würzig gut!", Weight: 10},
		{Text: "Es ist Mittwoch, meine Kerle.", Weight: 50},
		{Text: "Freitag ab eins macht jeder seins!", Weight: 50},
		{Text: "WOCHENENDE! SAUFEN!", Weight: 50},
		{Text: "Komm in die Gruppe! Hinterbüro ist beste!", Weight: 50},
		{Text: "Hallo, I bims. Vong Fisch Sprache her.", Weight: 50},
		{Text: "Der Gerät wird nie müde. Der Gerät schläft nie ein. Der Gerät ist immer vor die Chef im Geschäft.", Weight: 40},
		{Text: "Haben wir noch Peps da?", Weight: 50},
		{Text: "Läuft bei uns. Ich mach nix, bin aber auch nicht billable.", Weight: 50},
		{Text: "Was habt ihr heute gemacht? Ich hab gaar nix gemacht! Ich hab gar nix gemacht!", Weight: 50},
		{Text: "Ich hab Polizei! Ich hab Polizei!", Weight: 50},
		{Text: "Es gibt keine Experten! Keine Experten, außer John! Quanten-John!", Weight: 50},
		{Text: "Bruder, muss los! Ab ins Wasser!", Weight: 50},
		{Text: "IHR FILMT MICH INS GESICHT! DAS DÜRFEN SIE NICHT!", Weight: 50},
		{Text: "Halt, Stopp! Es bleibt alles so, wie es ist!", Weight: 50},
		{Text: "Unsere Schreibtische müssen verdichtet sein! Genaus wie die Kranplätze!", Weight: 50},
		{Text: "Technik, die begeistert. Das bin ich!", Weight: 50},
		{Text: "Arbeit?! Gönnt euch.", Weight: 50},
		{Text: "Ich küss dein Auge Habibi! ", Weight: 50},
		{Text: "DynamoDB?! Nein danke! Da ist die Tür!", Weight: 50},
		{Text: "Einfach mal machen!", Weight: 50},
		{Text: "Rüdiger keine Kapriolen!", Weight: 50},
		{Text: "Schauen wir mal was wird. Was wird.", Weight: 50},
		{Text: "Hey was machst du den hier? Das wolltest du wohl klauen?! ALARM!", Weight: 50},
		{Text: "Was ist denn mit Thorsten los?", Weight: 50},
		{Text: "ROOOOOOOOOBERT!!!", Weight: 50},
		{Text: "Meine Mama hat gesagt ich darf Fortnite spielen!", Weight: 50},
		{Text: "Was hast du denn da gekauft?! Coca Cola Light?? Ich wollte doch eine ZERROO!!", Weight: 50},
		{Text: "Was guckst du? Schau weg!", Weight: 50},
		{Text: "Lass mich in Ruhe!", Weight: 50},
		{Text: "Still hier. Sus.", Weight: 50},
		{Text: "Lügen darf man nicht sagen.", Weight: 50},
		{Text: "Ich muss raus. Ich muss rauuuus!", Weight: 50},
		{Text: "EGAL!", Weight: 50},
		{Text: "Ich bin der Uwe, ich bin auch dabei.", Weight: 50},
		{Text: "Warum liegt hier Stroh?", Weight: 50},
		{Text: "Dunkel war′s, der Mond schien helle,\n\nschneebedeckt die grüne Flur,\n\nals ein Wagen blitzesschnelle\n\nlangsam um die Ecke fuhr.\n\n \n\nDrinnen saßen stehend Leute\n\nschweigend ins Gespräch vertieft\n\nals ein totgeschossner Hase\n\nauf der Sandbank Schlittschuh lief.\n\n \n\nUnd der Wagen fuhr im Trabe\n\nrückwärts einen Berg hinauf.\n\nDroben zog ein alter Rabe\n\ngrade eine Turmuhr auf.\n\n \n\nRingsumher herrscht tiefes Schweigen\n\nund mit fürchterlichem Krach\n\nspielen in des Grases Zweigen\n\nzwei Kamele lautlos Schach.\n\n \n\nUnd auf einer roten Bank,\n\ndie blau angestrichen war\n\nsaß ein blondgelockter Jüngling\n\nmit kohlrabenschwarzem Haar.\n\n \n\nNeben ihm ne alte Schrulle,\n\ndie kaum siebzehn Jahr alt war,\n\nin der Hand ne Butterstulle,\n\ndie mit Schmalz bestrichen war.\n\n \n\nOben auf dem Apfelbaume,\n\nder sehr süße Birnen trug,\n\nhing des Frühlings letzte Pflaume\n\nund an Nüssen noch genug.\n\n \n\nVon der regennassen Straße\n\nwirbelte der Staub empor.\n\nUnd ein Junge bei der Hitze\n\nmächtig an den Ohren fror.\n\n \n\nBeide Hände in den Taschen\n\nhielt er sich die Augen zu.\n\nDenn er konnte nicht ertragen,\n\nwie nach Veilchen roch die Kuh.\n\n \n\nUnd zwei Fische liefen munter\n\ndurch das blaue Kornfeld hin.\n\nEndlich ging die Sonne unter\n\nund der graue Tag erschien.\n\n \n\nHolder Engel, süßer Bengel,\n\nfurchtbar liebes Trampeltier.\n\nDu hast Augen wie Sardellen,\n\nalle Ochsen gleichen Dir.\n\n \n\nEine Kuh, die saß im Schwalbennest\n\nmit sieben jungen Ziegen,\n\ndie feierten ihr Jubelfest\n\nund fingen an zu fliegen.\n\nDer Esel zog Pantoffeln an,\n\nist übers Haus geflogen,\n\nund wenn das nicht die Wahrheit ist,\n\nso ist es doch gelogen.", Weight: 20},
	}

	if hour >= 11 && hour <= 12 {
		phrases[0].Weight += 70
	} else {
		phrases[0].Weight = 0
	}
	if hour >= 15 && hour <= 17 {
		phrases[1].Weight += 70
		phrases[2].Weight += 70
	}
	if hour >= 17 && hour <= 19 {
		phrases[3].Weight += 70
	} else {
		phrases[3].Weight = 0
	}
	if day == 3 {
		phrases[4].Weight += 80
	} else {
		phrases[4].Weight = 0
	}
	if day == 5 {
		phrases[5].Weight += 80
		phrases[6].Weight += 80
	} else {
		phrases[5].Weight = 0
		phrases[6].Weight = 0
	}

	// Make a copy to avoid modifying the original slice with time-based weights
	phrasesToConsider := make([]Phrase, len(phrases))
	copy(phrasesToConsider, phrases)

	playedItems, err := playlist.GetPlayedItems()
	if err != nil {
		log.Printf("Error getting played items: %v", err)
	} else {
		recentlyPlayed := make(map[string]bool)
		cutoff := time.Now().Add(-1 * time.Hour)
		for _, item := range playedItems {
			if item.Type == "text" && item.Timestamp.After(cutoff) {
				recentlyPlayed[item.Name] = true
			}
		}

		for i, p := range phrasesToConsider {
			if recentlyPlayed[p.Text] {
				phrasesToConsider[i].Weight = 0
			}
		}
	}

	totalWeight := 0
	for _, p := range phrasesToConsider {
		totalWeight += p.Weight
	}

	// If all eligible phrases have been played, reset to the original list
	if totalWeight == 0 && len(phrases) > 0 {
		log.Println("All phrases have been played recently. Resetting phrase playlist for this round.")
		phrasesToConsider = phrases // Reset to the list with original time-based weights
		totalWeight = 0
		for _, p := range phrasesToConsider {
			totalWeight += p.Weight
		}
	}

	if totalWeight == 0 {
		log.Println("No phrases to say after filtering and potential reset.")
		return ""
	}

	r := rand.Intn(totalWeight)
	for _, p := range phrasesToConsider {
		r -= p.Weight
		if r < 0 {
			item := playlist.PlayedItem{
				Name:      p.Text,
				Type:      "text",
				Timestamp: time.Now(),
			}
			if err := playlist.AddPlayedItem(item, 1*time.Hour); err != nil {
				log.Printf("Error adding played item: %v", err)
			}
			return p.Text
		}
	}

	// Fallback, should ideally not be reached if totalWeight > 0
	if len(phrases) > 0 {
		return phrases[0].Text
	}
	return ""
}

func sing(myFish *fish.Fish, soundDir string) {
	allFiles, err := os.ReadDir(soundDir)
	if err != nil {
		log.Printf("failed to read sound directory '%s': %v", soundDir, err)
		return
	}

	var audioFiles []os.DirEntry
	for _, file := range allFiles {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !file.IsDir() && (ext == ".wav" || ext == ".mp3") {
			audioFiles = append(audioFiles, file)
		}
	}

	var availableFiles []os.DirEntry
	playedItems, err := playlist.GetPlayedItems()
	if err != nil {
		log.Printf("Error getting played items: %v", err)
		availableFiles = audioFiles // Play from all if we can't read playlist
	} else {
		recentlyPlayed := make(map[string]bool)
		cutoff := time.Now().Add(-1 * time.Hour)
		for _, item := range playedItems {
			if item.Type == "song" && item.Timestamp.After(cutoff) {
				recentlyPlayed[item.Name] = true
			}
		}

		for _, file := range audioFiles {
			if !recentlyPlayed[file.Name()] {
				availableFiles = append(availableFiles, file)
			}
		}
	}

	// If all songs have been played recently, reset the available list to all songs.
	if len(availableFiles) == 0 && len(audioFiles) > 0 {
		log.Println("All songs have been played recently. Resetting song playlist for this round.")
		availableFiles = audioFiles
	}

	if len(availableFiles) == 0 {
		log.Println("no .wav or .mp3 files found to sing, skipping.")
		return
	}

	randomFile := availableFiles[rand.Intn(len(availableFiles))]
	myFish.PlaySoundFile(randomFile.Name())
}

func runFishCycle(myFish *fish.Fish, piperClient *piper.PiperClient, soundDir string, enableTTS bool) {
	log.Println("Raising body...")
	myFish.Lock()
	if err := myFish.RaiseBody(); err != nil {
		log.Printf("Error raising body: %v", err)
	}
	myFish.Unlock()
	time.Sleep(1 * time.Second)

	// Check for queued items first
	queueItem, err := playlist.GetNextQueueItem()
	if err != nil {
		log.Printf("Error checking queue: %v", err)
	}

	if queueItem != nil {
		log.Printf("Playing queued item: %s (type: %s)", queueItem.Name, queueItem.Type)
		switch queueItem.Type {
		case "song":
			myFish.PlaySoundFile(queueItem.Name)
		case "text":
			if enableTTS {
				myFish.Say(piperClient, queueItem.Name)
			} else {
				log.Printf("Would say: %s", queueItem.Name)
			}
		}
	} else {
		// No queued items, do random action
		action := rand.Intn(2)
		if action == 0 {
			phraseToSay := getWeightedRandomPhrase()
			if enableTTS {
				myFish.Say(piperClient, phraseToSay)
			} else {
				log.Printf("Would say: %s", phraseToSay)
			}
		} else {
			sing(myFish, soundDir)
		}
	}

	time.Sleep(1 * time.Second)
	log.Println("Stopping body...")
	myFish.Lock()
	if err := myFish.StopBody(); err != nil {
		log.Printf("Error stopping body: %v", err)
	}
	myFish.Unlock()
	time.Sleep(1 * time.Second)

	log.Println("Tail...")
	myFish.Lock()
	if err := myFish.RaiseTail(); err != nil {
		log.Printf("Error raising tail: %v", err)
	}
	myFish.Unlock()
	time.Sleep(1 * time.Second)

	log.Println("Stopping tail...")
	myFish.Lock()
	if err := myFish.StopBody(); err != nil {
		log.Printf("Error stopping tail: %v", err)
	}
	myFish.Unlock()
}
