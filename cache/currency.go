package currency

import (
	"sync/atomic"
	"time"

	"github.com/5thwork/common/logger"
	"github.com/5thwork/wallet/database/mysql"
	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
)

const (
	BTC = 1
	ETH = 2
)

type CurrencyService struct {
	mysql *xorm.Engine
	data  atomic.Value
}

func NewCurrencyService(db *xorm.Engine) *CurrencyService {
	cs := CurrencyService{mysql: db}
	if err := cs.update(); err != nil {
		panic(err)
	}
	go cs.loop()

	return &cs
}

func (c *CurrencyService) GetPid(id int) (int, bool) {
	if id == 1 || id == 2 {
		return id, true
	}
	data := c.data.Load().(map[int]mysql.Currencies)
	one, ok := data[id]
	if !ok {
		return 0, false
	}

	return one.Pid, true
}

func (c *CurrencyService) GetContract(id int) (string, bool) {
	data := c.data.Load().(map[int]mysql.Currencies)
	one, ok := data[id]
	if !ok {
		return "", false
	}

	return one.ContractAddress, true
}

func (c *CurrencyService) GetSymbol(id int) (string, bool) {
	data := c.data.Load().(map[int]mysql.Currencies)
	one, ok := data[id]
	if !ok {
		return "", false
	}

	return one.Symbol, true
}

func (c *CurrencyService) GetCurrency(id int) (*mysql.Currencies, error) {
	data := c.data.Load().(map[int]mysql.Currencies)
	one, ok := data[id]
	if !ok {
		return nil, mysql.ErrNotFound
	}

	return &one, nil
}

func (c *CurrencyService) GetAllCurrency() map[int]mysql.Currencies {
	return c.data.Load().(map[int]mysql.Currencies)
}

func (c *CurrencyService) loop() {
	tk := time.NewTicker(time.Minute)
	defer tk.Stop()
	for {
		select {
		case <-tk.C:
			if err := c.update(); err != nil {
				logger.Error(err)
			}
		}
	}
}

func (c *CurrencyService) update() error {
	curs := make([]mysql.Currencies, 0, 1300)
	if err := c.mysql.Find(&curs); err != nil {
		return errors.Wrap(err, "currency find")
	}
	data := make(map[int]mysql.Currencies)
	for _, o := range curs {
		data[o.Id] = o
	}
	c.data.Store(data)

	return nil
}
