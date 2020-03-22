// Package settings - caricamento, reload e salvataggio impostazioni.
package settings

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Settings - struttura oggetto direttamente utilizzabile.
type Settings struct {
	verbose  bool
	filename string

	lock      sync.Mutex
	timerSave *time.Timer

	Data interface{} // Punta ad una struttura dati custom
}

func (set *Settings) init(filename string, data interface{}, verbose bool) error {
	if filename[0] != '/' {
		ex, err := os.Executable()
		if err != nil {
			return err
		}
		set.filename = filepath.Dir(ex) + "/" + filename
	} else {
		set.filename = filename
	}

	set.Data = data

	set.verbose = verbose

	if verbose {
		log.Println("Settings filename", set.filename)
	}

	return nil
}

// LoadSettings - carica le impostazioni
func (set *Settings) LoadSettings() error {
	if set.Data == nil {
		return errors.New("settings data struct not set")
	}

	set.lock.Lock()
	defer set.lock.Unlock()

	if set.verbose {
		log.Println("Load settings")
	}

	fileContent, err := ioutil.ReadFile(set.filename)
	if err != nil {
		return err
	}

	data, isMap := set.Data.(map[string]interface{})
	if isMap {
		// mappa che contiene strutture; deserializza ogni singola struttura, anche se in modo inefficiente
		var temp map[string]interface{}
		err = json.Unmarshal([]byte(fileContent), &temp)
		if err != nil {
			return err
		}

		for key, item := range temp {
			tempj, _ := json.Marshal(item)

			err := json.Unmarshal(tempj, data[key])
			if err != nil {
				return err
			}
		}
	} else {
		err = json.Unmarshal([]byte(fileContent), &set.Data)
		if err != nil {
			return err
		}
	}

	//fmt.Println(set)

	if set.Data == nil {
		return errors.New("null settings in file")
	}

	return nil
}

// SaveSettings - salva le impostazioni
func (set *Settings) SaveSettings() error {
	if set.Data == nil {
		return errors.New("data not set")
	}

	set.lock.Lock()
	defer set.lock.Unlock()

	if set.verbose {
		log.Println("Save settings")
	}

	f, err := os.Create(set.filename)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.MarshalIndent(set.Data, "", "\t")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(set.filename, b, 0666)

	return err
}

// SaveSettingsDebounce - salva le impostazioni dopo un certo ritardo dall'ultima invocazione.
// Ogni invocazione resetta il il conteggio del timeout.
func (set *Settings) SaveSettingsDebounce(saveAfter time.Duration) {
	if set.timerSave != nil {
		set.timerSave.Reset(saveAfter)
	} else {
		saveFunc := func() {
			set.SaveSettings()
		}

		set.timerSave = time.AfterFunc(saveAfter, saveFunc)
	}
}

// New - restituisce un gestore inizializzato per dati custom
// - filename: se vuoto viene automaticamente settato a settings.json
// nel medesimo path dell'eseguibile.
// - data: passare il puntatore ad una struct contenente i dati
// da caricare e salvare.
func New(filename string, data interface{}, verbose bool) (*Settings, error) {
	ctrl := Settings{}
	err := ctrl.init(filename, data, verbose)
	if err != nil {
		return nil, err
	}

	return &ctrl, nil
}
