package bitcoin

import (
	"context"
	"time"

	"github.com/5thwork/wallet/database/mysql"
	"github.com/5thwork/wallet/pkg/codes"
	"github.com/5thwork/wallet/pkg/status"
	"github.com/5thwork/wallet/pkg/wallet"
	"github.com/5thwork/wallet/pkg/worker"
	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
)

type UTXOService struct {
	pool      *worker.Pool
	mysql     *xorm.Engine
	btcWallet *BTCWallet
}

func NewUTXOService(mysql *xorm.Engine, btcWallet *BTCWallet) *UTXOService {
	svc := UTXOService{
		mysql:     mysql,
		btcWallet: btcWallet,
	}
	svc.pool = worker.NewPool(5)
	svc.pool.Start()

	return &svc
}

func (us *UTXOService) GetUTXO(addr, deviceId string) ([]CurrencyUTXO, error) {
	var (
		taskCount int
		doneCount int
	)
	list := make([]CurrencyUTXO, 0, 10)
	session := mysql.NewEngine(us.mysql)
	defer session.Close()

	m, err := session.GetMember(deviceId)
	if err != nil {
		return nil, status.Newf(codes.DBQuery, "get member by device %s: %s", deviceId, err)
	}

	addrs, err := session.GetBTCWalletAddress(addr, m.Id)
	if err != nil {
		if err == mysql.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "GetBTCWalletAddress")
	}
	taskCount = len(addrs)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	taskCtx := TaskCtx{
		btcWallet: us.btcWallet,
		utxoChan:  make(chan CurrencyUTXO),
		errChan:   make(chan error),
		ctx:       ctx,
	}

	//分发任务
	go us.dispatch(&taskCtx, addrs)

	for {
		select {
		case item := <-taskCtx.utxoChan:
			if item.UTXOList != nil {
				list = append(list, item)
			}
			doneCount++
			if doneCount == taskCount {
				return list, nil
			}
		case err := <-taskCtx.errChan:
			return nil, errors.Wrap(err, "errChan")
		case <-ctx.Done():
			return nil, status.FromCode(codes.TimeOut)
		}
	}
}

func (us *UTXOService) dispatch(ctx *TaskCtx, list []mysql.BtcAddress) {
	for _, o := range list {
		select {
		case <-ctx.ctx.Done():
			return
		default:
		}
		task := newUtxoTask(ctx, CurrencyUTXO{Cid: 1, Address: o.Address})
		if err := us.pool.AddJob(task); err != nil {
			task.AfterDo(err)
		}
	}
}

type CurrencyUTXO struct {
	Cid      int           `json:"cid"`
	Address  string        `json:"addr"`
	UTXOList []wallet.UTXO `json:"list"`
}

type TaskCtx struct {
	btcWallet *BTCWallet
	utxoChan  chan CurrencyUTXO
	errChan   chan error
	ctx       context.Context
}

type utxoTask struct {
	ctx  *TaskCtx
	data CurrencyUTXO
}

func newUtxoTask(ctx *TaskCtx, data CurrencyUTXO) *utxoTask {
	return &utxoTask{
		ctx:  ctx,
		data: data,
	}
}

func (ut *utxoTask) Do() error {
	utxo, err := ut.ctx.btcWallet.GetUTXO(ut.data.Address)
	if err != nil {
		return errors.Wrapf(err, "get utxo for address %s", ut.data.Address)
	}
	if len(utxo) == 0 {
		return nil
	}
	ut.data.UTXOList = utxo

	return nil
}

func (ut *utxoTask) AfterDo(err error) {
	select {
	case <-ut.ctx.ctx.Done():
		return
	default:
	}
	if err != nil {
		ut.ctx.errChan <- err
	} else {
		ut.ctx.utxoChan <- ut.data
	}
}
