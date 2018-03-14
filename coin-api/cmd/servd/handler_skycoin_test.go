package servd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/skycoin/services/coin-api/internal/model"
)

const (
	rawTxID  = "bff13a47a98402ecf2d2eee40464959ad26e0ed6047de5709ffb0c0c9fc1fca5"
	rawTxStr = "dc00000000a8558b814926ed0062cd720a572bd67367aa0d01c0769ea4800adcc89cdee524010000008756e4bde4ee1c725510a6a9a308c6a90d949de7785978599a87faba601d119f27e1be695cbb32a1e346e5dd88653a97006bf1a93c9673ac59cf7b5db7e07901000100000079216473e8f2c17095c6887cc9edca6c023afedfac2e0c5460e8b6f359684f8b020000000060dfa95881cdc827b45a6d49b11dbc152ecd4de640420f00000000000000000000000000006409744bcacb181bf98b1f02a11e112d7e4fa9f940f1f23a000000000000000000000000"
)

func TestHandlerMulti(t *testing.T) {
	e := echo.New()
	handler := newHandlerMulti("127.0.0.1", 6430)

	t.Run("TestGenerateKeyPair", func(t *testing.T) {
		req := httptest.NewRequest(echo.POST, "/keys", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		ctx := e.NewContext(req, rec)
		if err := handler.generateKeys(ctx); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusCreated {
			t.Fatalf("wrong status, expected %d  got %d", rec.Code, http.StatusCreated)
		}
		rsp := struct {
			Status string              `json:"status"`
			Code   int                 `json:"code"`
			Result *model.KeysResponse `json:"result"`
		}{
			Result: &model.KeysResponse{},
		}
		err := json.Unmarshal(rec.Body.Bytes(), &rsp)
		if err != nil {
			t.Fatal(err)
		}
		if len(rsp.Result.Private) == 0 || len(rsp.Result.Public) == 0 {
			t.Fatal("key cannot be empty")
		}

		t.Run("TestGenerateAddress", func(t *testing.T) {
			req := httptest.NewRequest(echo.POST, fmt.Sprintf("/address/%s", rsp.Result.Public), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.SetParamNames("key")
			ctx.SetParamValues(rsp.Result.Public)
			err := handler.generateSeed(ctx)
			if err != nil {
				t.Fatal(err.Error())
			}
			rsp := struct {
				Status string                 `json:"status"`
				Code   int                    `json:"code"`
				Result *model.AddressResponse `json:"result"`
			}{
				Result: &model.AddressResponse{},
			}
			if rec.Code != http.StatusCreated {
				t.Fatalf("wrong status, expected %d  got %d", rec.Code, http.StatusCreated)
			}

			err = json.Unmarshal(rec.Body.Bytes(), &rsp)
			if err != nil {
				t.Fatal(err)
			}

			if len(rsp.Result.Address) == 0 {
				t.Fatal("key cannot be empty")
				// return
			}

			t.Run("checkBalance", func(t *testing.T) {
				req := httptest.NewRequest(echo.POST, fmt.Sprintf("/address/%s", rsp.Result.Address), nil)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				recorder := httptest.NewRecorder()
				ctx := e.NewContext(req, recorder)
				ctx.SetParamNames("address")
				ctx.SetParamValues(rsp.Result.Address)
				err := handler.checkBalance(ctx)
				if err != nil {
					t.Fatal(err)
				}
				rspBalance := struct {
					Status string                 `json:"status"`
					Code   int                    `json:"code"`
					Result *model.BalanceResponse `json:"result"`
				}{
					Result: &model.BalanceResponse{},
				}
				err = json.Unmarshal(rec.Body.Bytes(), &rspBalance)
				if err != nil {
					t.Fatalf("error unmarshalling response: %v", err)
				}
				if len(rspBalance.Result.Address) == 0 {
					t.Fatalf("address can't be nil")
				}
			})
		})
		t.Run("signTransaction", func(t *testing.T) {
			sourceTrans := rawTxStr
			signid := rsp.Result.Private
			req := httptest.NewRequest(echo.POST, fmt.Sprintf("/transaction/sign/%s", rawTxStr), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()
			ctx := e.NewContext(req, recorder)
			ctx.SetParamNames("signid")
			ctx.SetParamValues(signid)

			ctx.SetParamNames("sourceTrans")
			ctx.SetParamValues(sourceTrans)

			err := handler.signTransaction(ctx)
			rspTrans := struct {
				Status string                 `json:"status"`
				Code   int                    `json:"code"`
				Result *model.TransactionSign `json:"result"`
			}{
				Result: &model.TransactionSign{},
			}
			err = json.Unmarshal(recorder.Body.Bytes(), &rspTrans)
			if err != nil {
				t.Fatalf("error unmarshalling response: %v", err)
			}
			if len(rspTrans.Result.Signid) == 0 {
				t.Fatalf("rspTrans.Result.Signid cannot be zero length")
			}

			t.Run("injectTransaction", func(t *testing.T) {
				transid := rawTxStr
				req := httptest.NewRequest(echo.PUT, fmt.Sprintf("/transaction/%s", transid), nil)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				recorder := httptest.NewRecorder()
				ctx := e.NewContext(req, recorder)

				ctx.SetParamNames("transid")
				ctx.SetParamValues(transid)
				err := handler.injectTransaction(ctx)
				if err != nil {
					t.Fatalf("error injectin transaction %s", err.Error())
				}
				rspTrans := struct {
					Status string             `json:"status"`
					Code   int                `json:"code"`
					Result *model.Transaction `json:"result"`
				}{
					Result: &model.Transaction{},
				}
				err = json.Unmarshal(recorder.Body.Bytes(), &rspTrans)
				if err != nil {
					t.Fatalf("error unmarshalling response: %v", err)
				}
				if len(rspTrans.Result.Transid) > 0 {
					t.Fatal("rspTrans.Result.Transid cannot be zero lenght")
				}
			})
		})
	})

	t.Run("checkTransaction", func(t *testing.T) {
		transid := "transid"
		req := httptest.NewRequest(echo.GET, fmt.Sprintf("/transaction/%s", transid), nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()
		ctx := e.NewContext(req, recorder)
		ctx.SetParamNames("transid")
		ctx.SetParamValues(transid)
		err := handler.checkTransaction(ctx)
		if err != nil {
			t.Fatalf("error check transaction %s", err.Error())
		}

		rspTrans := struct {
			Status string                   `json:"status"`
			Code   int                      `json:"code"`
			Result *model.TransactionStatus `json:"result"`
		}{
			Result: &model.TransactionStatus{},
		}
		err = json.Unmarshal(recorder.Body.Bytes(), &rspTrans)
		if err != nil {
			t.Fatalf("error unmarshalling response: %v", err)
		}
		if len(rspTrans.Result.Transid) == 0 {
			t.Fatal("rspTrans.Result.Transid can't be zero length")
		}
	})
}
