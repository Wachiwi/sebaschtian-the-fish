//go:build linux

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/oto/v2"
	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/youpy/go-wav"
)

var otoCtx *oto.Context

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

	totalWeight := 0
	for _, p := range phrases {
		totalWeight += p.Weight
	}

	r := rand.Intn(totalWeight)
	for _, p := range phrases {
		r -= p.Weight
		if r < 0 {
			return p.Text
		}
	}
	return phrases[0].Text
}

func playAudioWithAnimation(myFish *fish.Fish, pcmData []byte) {
	if otoCtx == nil {
		fmt.Println("audio context not initialized")
		return
	}

	playbackReader := bytes.NewReader(pcmData)
	analysisReader := bytes.NewReader(pcmData)

	player := otoCtx.NewPlayer(playbackReader)
	player.Play()

	go func() {
		const sampleRate = 22050
		const bitDepth = 2
		const channels = 1
		const chunkDuration = 100 * time.Millisecond
		const amplitudeThreshold = 1500
		isMouthOpen := false
		chunkSize := int(float64(sampleRate) * chunkDuration.Seconds() * float64(bitDepth*channels))
		buffer := make([]byte, chunkSize)

		for {
			n, err := io.ReadFull(analysisReader, buffer)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				fmt.Println("Error reading audio for animation: %v", err)
				break
			}

			var sum int64
			var count int
			for i := 0; i < n; i += 2 {
				if i+1 >= n {
					break
				}
				sample := int16(binary.LittleEndian.Uint16(buffer[i : i+2]))
				if sample < 0 {
					sum += int64(-sample)
				} else {
					sum += int64(sample)
				}
				count++
			}

			if count == 0 {
				time.Sleep(chunkDuration)
				continue
			}

			avgAmplitude := sum / int64(count)

			myFish.Lock()
			if avgAmplitude > amplitudeThreshold && !isMouthOpen {
				myFish.OpenMouth()
				isMouthOpen = true
			} else if avgAmplitude <= amplitudeThreshold && isMouthOpen {
				myFish.CloseMouth()
				isMouthOpen = false
			}
			myFish.Unlock()
			time.Sleep(chunkDuration)
		}

		myFish.Lock()
		if isMouthOpen {
			myFish.CloseMouth()
			time.Sleep(1 * time.Second)
			myFish.StopMouth()
		}

		myFish.StopBody()
		myFish.StopMouth()

		myFish.Unlock()
	}()

	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}

	if err := player.Close(); err != nil {
		fmt.Println("failed to close player: %v", err)
	}
}

func say(myFish *fish.Fish, piperClient *piper.PiperClient, text string) {
	fmt.Println("saying '%s'...", text)
	wavData, err := piperClient.Synthesize(text)
	if err != nil {
		fmt.Println("failed to synthesize text: %v", err)
		return
	}

	wavReader := wav.NewReader(bytes.NewReader(wavData))
	pcmData, err := io.ReadAll(wavReader)
	if err != nil {
		fmt.Println("failed to read pcm data: %v", err)
		return
	}
	playAudioWithAnimation(myFish, pcmData)
	fmt.Println("finished saying '%s'.", text)
}

func sing(myFish *fish.Fish) {
	soundDir := "/sound-data"
	files, err := os.ReadDir(soundDir)
	if err != nil {
		fmt.Println("failed to read sound directory '%s': %v", soundDir, err)
		return
	}

	if len(files) == 0 {
		fmt.Println("no sound files to sing, skipping.")
		return
	}

	randomFile := files[rand.Intn(len(files))]
	filePath := filepath.Join(soundDir, randomFile.Name())
	fmt.Println("singing '%s'...", randomFile.Name())

	wavData, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("failed to read sound file '%s': %v", filePath, err)
		return
	}

	wavReader := wav.NewReader(bytes.NewReader(wavData))
	pcmData, err := io.ReadAll(wavReader)
	if err != nil {
		fmt.Println("failed to decode wav data from '%s': %v", randomFile.Name(), err)
		return
	}

	playAudioWithAnimation(myFish, pcmData)
	fmt.Println("finished singing '%s'.", randomFile.Name())
}

func main() {
	myFish, err := fish.NewFish("gpiochip0")
	if err != nil {
		fmt.Println("failed to initialize fish: %v", err)
	}
	defer myFish.Close()

	fmt.Println("Initializing audio context...")
	var ready chan struct{}
	otoCtx, ready, err = oto.NewContext(22050, 1, oto.FormatSignedInt16LE)
	if err != nil {
		fmt.Println("failed to create oto context: %v", err)
	}
	<-ready
	fmt.Println("Audio context ready")

	piperClient := piper.NewPiperClient("http://piper:5000")

	say(myFish, piperClient, "Hallo Ich bins! Bin wieder da und ready!")
	myFish.Lock()
	myFish.StopBody()
	myFish.StopMouth()
	myFish.Unlock()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		fmt.Println("Error loading location: %v", err)
	}

	c := cron.New(
		cron.WithLocation(loc),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)

	c.AddFunc("* * * * *", func() {
		// Randomly choose between saying a phrase and singing a song
		action := rand.Intn(2) // 0 for say, 1 for sing

		// --- Start Fish Animation ---
		fmt.Println("Raising body...")
		myFish.Lock()
		if err := myFish.RaiseBody(); err != nil {
			log.Printf("Error raising body: %v", err)
		}
		myFish.Unlock()
		time.Sleep(1 * time.Second)

		if action == 0 {
			phraseToSay := getWeightedRandomPhrase()
			say(myFish, piperClient, phraseToSay)
		} else {
			sing(myFish)
		}

		// --- End Fish Animation ---
		time.Sleep(1 * time.Second) // Small pause after audio
		fmt.Println("Stopping body...")
		myFish.Lock()
		if err := myFish.StopBody(); err != nil {
			log.Printf("Error stopping body: %v", err)
		}
		myFish.Unlock()
		time.Sleep(1 * time.Second)

		fmt.Println("Tail...")
		myFish.Lock()
		if err := myFish.RaiseTail(); err != nil {
			log.Printf("Error raising tail: %v", err)
		}
		myFish.Unlock()
		time.Sleep(1 * time.Second)

		fmt.Println("Stopping tail...")
		myFish.Lock()
		if err := myFish.StopBody(); err != nil {
			log.Printf("Error stopping tail: %v", err)
		}
		myFish.Unlock()
	})
	go c.Start()

	select {}
}
