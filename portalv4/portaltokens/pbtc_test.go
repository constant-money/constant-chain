package portaltokens

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindClosestUTXO(t *testing.T) {
	type testCase struct {
		targetAmount uint64
		res          uint64
		err          error
	}
	tcs := []testCase{
		{
			targetAmount: 100,
			res:          100,
			err:          nil,
		},
		{
			targetAmount: 1000,
			res:          1000,
			err:          nil,
		},
		{
			targetAmount: 390,
			res:          500,
			err:          nil,
		},
		{
			targetAmount: 1100,
			res:          0,
			err:          errors.New("There is no utxo that has amount greater or equal target amount"),
		},
		{
			targetAmount: 50,
			res:          100,
			err:          nil,
		},
	}

	utxos := []*BtcUTXO{
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  100,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  250,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  300,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  500,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  1000,
		},
	}

	for _, tc := range tcs {
		res, err := findClosestUTXO(utxos, tc.targetAmount)
		assert.Equal(t, tc.err, err)
		if err == nil {
			assert.Equal(t, tc.res, res.Amount)
		}
	}
}

func TestChooseUTXOs(t *testing.T) {
	type testCase struct {
		amountTransfer uint64
		fee            uint64
		res            []*BtcUTXO
		isErr          bool
	}
	tcs := []testCase{
		{
			amountTransfer: 99,
			fee:            1,
			res: []*BtcUTXO{
				{
					TxID:    chainhash.Hash{},
					TxIndex: 0,
					Amount:  100,
				},
			},
			isErr: false,
		},
		{
			amountTransfer: 990,
			fee:            5,
			res: []*BtcUTXO{
				{
					TxID:    chainhash.Hash{},
					TxIndex: 0,
					Amount:  1000,
				},
			},
			isErr: false,
		},
		{
			amountTransfer: 1200,
			fee:            10,
			res: []*BtcUTXO{
				{
					TxID:    chainhash.Hash{},
					TxIndex: 0,
					Amount:  1000,
				},
				{
					TxID:    chainhash.Hash{},
					TxIndex: 0,
					Amount:  500,
				},
			},
			isErr: false,
		},
		{
			amountTransfer: 0,
			fee:            0,
			res:            []*BtcUTXO{},
			isErr:          true,
		},
		{
			amountTransfer: 3000,
			fee:            0,
			res:            []*BtcUTXO{},
			isErr:          true,
		},
	}

	utxos := []*BtcUTXO{
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  100,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  250,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  300,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  500,
		},
		&BtcUTXO{
			TxID:    chainhash.Hash{},
			TxIndex: 0,
			Amount:  1000,
		},
	}

	for i, tc := range tcs {
		res, err := chooseUTXOs(utxos, tc.amountTransfer, tc.fee)
		assert.Equal(t, tc.isErr, err != nil)
		if err == nil {
			assert.Equal(t, tc.res, res, i)
		}
	}
}
