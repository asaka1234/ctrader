package conf

import (
	"bufio"
	"logtech.com/exchange/ltrader/entity"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

//金融工具(financial instruments)是指在金融市场中可交易的金融资产
//相当于要处理的所有pair

// global instrument map which is fully synchronized
var IMap instrumentMap

type instrumentMap struct {
	sync.RWMutex
	id       int64
	bySymbol map[string]entity.Instrument //value是一个结构
	byID     map[int64]entity.Instrument
}

//---------------------------------------------------------------

func (im *instrumentMap) GetBySymbol(symbol string) entity.Instrument {
	im.RLock()
	defer im.RUnlock()

	i, ok := im.bySymbol[symbol]
	if !ok {
		return nil
	}
	return i
}
func (im *instrumentMap) GetByID(id int64) entity.Instrument {
	im.RLock()
	defer im.RUnlock()

	i, ok := im.byID[id]
	if !ok {
		return nil
	}
	return i
}
func (im *instrumentMap) AllSymbols() []string {
	im.RLock()
	defer im.RUnlock()

	var symbols []string
	for k, _ := range im.bySymbol {
		symbols = append(symbols, k)
	}
	return symbols
}

// the put/nextID are only needed if dynamic instrument creation is added, or for test cases

func (im *instrumentMap) nextID() int64 {
	return atomic.AddInt64(&im.id, 1)
}
func (im *instrumentMap) Put(instrument entity.Instrument) {
	im.Lock()
	defer im.Unlock()

	im.bySymbol[instrument.Symbol()] = instrument
	im.byID[instrument.ID()] = instrument
}

// load the instrument map from a file, see configs/instruments.txt for the format
func (im *instrumentMap) Load(filepath string) error {
	inputFile, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "//") || strings.HasPrefix(s, "#") {
			continue
		}
		if s == "" {
			continue
		}
		parts := strings.Fields(s)
		id, _ := strconv.Atoi(parts[0])
		if len(parts) == 2 {
			i := entity.NewInstrument(int64(id), parts[1])
			im.Put(i)
		}
	}
	return nil
}

func init() {
	IMap.bySymbol = make(map[string]entity.Instrument)
	IMap.byID = make(map[int64]entity.Instrument)
}
