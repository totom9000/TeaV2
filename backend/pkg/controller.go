package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	query "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/shopspring/decimal"

	"go.uber.org/zap"
)

var once sync.Once

type PKDeleteRequest struct {
	Address string `json:"address"`
}

func (c *Controller) DeletePrivateKey(w http.ResponseWriter, r *http.Request) {
	pkdr := &PKDeleteRequest{}
	err := json.NewDecoder(r.Body).Decode(pkdr)
	if err != nil {
		c.Log.Error("error decoding pk delete request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// list the keys in the local file system
	files, err := ioutil.ReadDir("data/keys")
	if err != nil {
		c.Log.Error("error reading keys directory", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// find the key to delete
	for _, f := range files {
		if strings.Contains(f.Name(), pkdr.Address) {
			// delete the private key from the local file system
			// "data/keys/" + pk.Chain + "-" + pk.Address + ".txt"
			err = os.Remove("data/keys/" + f.Name())
			if err != nil {
				c.Log.Error("error deleting private key", zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode("success")
			return
		}
	}

	c.Log.Error("could not find the key to delete for address", zap.String("address", pkdr.Address))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode("could not find the key to delete")
}

func (c *Controller) RootHandler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		kdp := path.Join(os.Getenv("KO_DATA_PATH"))
		if kdp == "" {
			kdp = "kodata"
		}

		if !strings.HasSuffix(kdp, "/") {
			kdp = kdp + "/"
		}
		c.rootHandler = http.FileServer(http.Dir(kdp))
	})

	c.rootHandler.ServeHTTP(w, r)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *Controller) SocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.Log.Error("error during WS connection upgradation", zap.Error(err))
		return
	}

	c.WSHandler(conn)
}

// Sell is a http route handler that accepts a sell order
// sell orders are stored in an on prem MongoDB database
func (c *Controller) Sell(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.Log.Debug("received sell order")

	sellOrder := &SellOrder{}
	err := json.NewDecoder(r.Body).Decode(sellOrder)
	if err != nil {
		c.Log.Error("error decoding sell order", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: verify that sellOrder.Amount != nil
	// verify that all fields are not empty
	if sellOrder.Currency == "" || sellOrder.TradeAsset == "" {
		c.Log.Error("error: currency or trade asset is empty on sell order ID " + sellOrder.TXID)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("invalid sell order, currency or trade asset is empty")
		return
	}

	// generate a random string to be used as the transaction ID
	sellOrder.TXID = uuid.New().String()
	// add the NKN address to the sell order
	sellOrder.SellerNKNAddress = c.NKNClient.Address()
	// prepare sellOrder to send in the http request
	sellOrderJSON, err := json.Marshal(sellOrder)
	if err != nil {
		c.Log.Error("error marshalling sell order", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	io := bytes.NewBuffer(sellOrderJSON)
	req, err := http.NewRequest("POST", c.SAASAddress+"/submittrade", io)
	if err != nil {
		c.Log.Error("error creating http request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("error creating http request")
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.Log.Error("error sending sell order to Party", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("error sending sell order to Party")
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.Log.Error("error reading response body from Party", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("error reading response body from Party")
		return
	}

	if resp.StatusCode != 202 {
		c.Log.Error("error: status code is not 202")
		c.Log.Info("response body from Party: ", zap.String("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("error: resposne code from Party was not a 202.. Please check tea's backend service logs for more information. Response from Party was: " + string(body))
		return
	}

	// return accepted to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode("sell order accepted")
}

type QueryAllTradeOrdersResponse struct {
	TradeOrders []TradeOrders       `protobuf:"bytes,1,rep,name=tradeOrders,proto3" json:"tradeOrders"`
	Pagination  *query.PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type TradeOrders struct {
	Index              string `protobuf:"bytes,1,opt,name=index,proto3" json:"index,omitempty"`
	TradeAsset         string `protobuf:"bytes,2,opt,name=tradeAsset,proto3" json:"tradeAsset,omitempty"`
	Price              string `protobuf:"bytes,3,opt,name=price,proto3" json:"price,omitempty"`
	Currency           string `protobuf:"bytes,4,opt,name=currency,proto3" json:"currency,omitempty"`
	Amount             string `protobuf:"bytes,5,opt,name=amount,proto3" json:"amount,omitempty"`
	SellerShippingAddr string `protobuf:"bytes,6,opt,name=sellerShippingAddr,proto3" json:"sellerShippingAddr,omitempty"`
	SellerNknAddr      string `protobuf:"bytes,7,opt,name=sellerNknAddr,proto3" json:"sellerNknAddr,omitempty"`
	RefundAddr         string `protobuf:"bytes,8,opt,name=refundAddr,proto3" json:"refundAddr,omitempty"`
}

func (c *Controller) ListOrders(w http.ResponseWriter, r *http.Request) {
	// http get request to the SAAS to get all the orders
	resp, err := http.Get(c.SAASAddress + "/listorders")
	if err != nil {
		c.Log.Error("error getting orders from Party", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("fetching orders from Party failed. Please check tea's backend service logs for more information. Error from Party request: " + err.Error())
	}
	defer resp.Body.Close()

	var orders *QueryAllTradeOrdersResponse
	json.NewDecoder(resp.Body).Decode(&orders)
	if orders == nil {
		orders = &QueryAllTradeOrdersResponse{}
	}

	// convert to a []SellOrder
	var sellOrders []SellOrder

	for _, order := range orders.TradeOrders {
		// convert string to big.Int
		d, err := decimal.NewFromString(order.Price)
		if err != nil {
			c.Log.Error("Error converting price to decimal")
			return
		}
		biPrice := d.BigInt()

		dA, err := decimal.NewFromString(order.Amount)
		if err != nil {
			c.Log.Error("Error converting amount to decimal")
			return
		}
		biAmount := dA.BigInt()

		sellOrder := SellOrder{
			TradeAsset:            order.TradeAsset,
			Price:                 biPrice,
			Currency:              order.Currency,
			Amount:                biAmount,
			TXID:                  order.Index,
			SellerShippingAddress: order.SellerShippingAddr,
			SellerNKNAddress:      order.SellerNknAddr,
			RefundAddress:         order.RefundAddr,
		}
		sellOrders = append(sellOrders, sellOrder)
	}

	if len(sellOrders) == 0 {
		sellOrders = []SellOrder{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sellOrders)
}

func (c *Controller) Buy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	buyOrder := &BuyOrder{}
	err := json.NewDecoder(r.Body).Decode(buyOrder)
	if err != nil {
		c.Log.Error("error decoding buy order", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	buyOrder.BuyerNKNAddress = c.NKNClient.Address()
	buyOrderJSON, err := json.Marshal(buyOrder)
	if err != nil {
		c.Log.Error("error marshalling sell order", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	io := bytes.NewBuffer(buyOrderJSON)
	req, err := http.NewRequest("POST", c.SAASAddress+"/buy", io)
	if err != nil {
		c.Log.Error("creating http request sending a buy order to Party", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.Log.Error("sending buy order to Party", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.Log.Error("did not get a 200 status code from Party")
		c.Log.Info("response body from Party: ", zap.String("body", fmt.Sprint(resp.Body)))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("did not get a 200 status code from Party. Please check tea's backend service logs for more information. Response from Party was: " + fmt.Sprint(resp.Body))
		return
	}

	// return body to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode("buy order accepted")
}

func (c *Controller) StartNKNConnection() {
	c.Log.Info("listening on " + c.NKNClient.Address())
	<-c.NKNClient.OnConnect.C
	for {
		select {
		case msg := <-c.NKNClient.OnMessage.C:
			c.Log.Info("message: " + string(msg.Data) + " from " + msg.Src)
			writeLog("Receive message " + string(msg.Data) + " from " + msg.Src)

			nknNotification := &NKNNotification{}
			err := json.Unmarshal(msg.Data, nknNotification)
			if err != nil {
				c.Log.Error("error decoding nkn notification", zap.Error(err))
				manager.send("error decoding nkn notification: " + err.Error())
				return
			}

			if nknNotification.PrivateKey != "" {
				c.Log.Info("saving private key to file system: " + nknNotification.PrivateKey)
				if err := savePKToFS(nknNotification); err != nil {
					c.Log.Error("error saving private key to file system", zap.Error(err))
					manager.send("error saving private key to file system: " + err.Error())
					return
				}
			}

			c.Log.Debug("sending notification to client")
			// send the notification to the client
			manager.send(string(msg.Data))
			msg.Reply([]byte("ok"))
		}
	}
}

// savePKToFS saves the private key to the file system
func savePKToFS(pk *NKNNotification) error {
	// create a new file and save the private key to it
	f, err := os.Create("data/keys/" + pk.Chain + "-" + pk.Address + ".txt")
	if err != nil {
		return err
	}
	defer f.Close()

	bte, err := json.Marshal(pk)
	if err != nil {
		return err
	}

	// write the entire `pk` struct to the file
	_, err = f.Write(bte)
	if err != nil {
		return err
	}

	return nil
}

func writeLog(msg string) error {
	if _, err := os.Stat("data/log.txt"); os.IsNotExist(err) {
		f, err := os.Create("data/log.txt")
		if err != nil {
			log.Println(err)
			return err
		}
		defer f.Close()
	}

	f, err := os.OpenFile("data/log.txt", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(msg + "\n"); err != nil {
		return err
	}

	return nil
}

func (c *Controller) GetNKNAddress(w http.ResponseWriter, r *http.Request) {
	if c.NKNClient == nil {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "NKN client is not initialized", http.StatusBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c.NKNClient.Address())
}

// func (c *Controller) FetchOpenOrderByNKN(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
// 	w.Header().Set("Content-Type", "application/json")

// 	oor := &OpenOrderRequest{}
// 	err := json.NewDecoder(r.Body).Decode(oor)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode("error decoding request body: " + err.Error())
// 		return
// 	}

// 	oorJson, err := json.Marshal(oor)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode("error marshalling open order request: " + err.Error())
// 		return
// 	}

// 	io := bytes.NewBuffer(oorJson)
// 	req, err := http.NewRequest("POST", c.SAASAddress+"/fetchopenorderbynkn", io)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode("error creating request: " + err.Error())
// 		return
// 	}

// 	req.Header.Set("Content-Type", "application/json")
// 	res, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode("error sending request: " + err.Error())
// 		return
// 	}

// 	defer res.Body.Close()
// 	body, err := ioutil.ReadAll(res.Body)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode("error reading response body: " + err.Error())
// 		return
// 	}

// 	NKNNotification := &[]NKNNotification{}
// 	err = json.Unmarshal(body, NKNNotification)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode("error unmarshalling response body: " + err.Error())
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(NKNNotification)
// }

func (c *Controller) GetPrivateKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// read the keys stored in the keys/ directory
	files, err := ioutil.ReadDir("data/keys/")
	if err != nil {
		c.Log.Error("error reading keys directory: " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("error reading keys directory: " + err.Error())
		return
	}

	var privateKeys []NKNNotification
	for _, f := range files {
		file, err := os.Open("data/keys/" + f.Name())
		if err != nil {
			c.Log.Error("error opening Private Key file: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("error opening Private Key file: " + err.Error())
			return
		}

		bte, err := ioutil.ReadAll(file)
		if err != nil {
			c.Log.Error("reading file: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("reading file: " + err.Error())
			return
		}

		pk := &NKNNotification{}
		err = json.Unmarshal(bte, pk)
		if err != nil {
			c.Log.Error("error unmarshalling private key: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("unmarshalling private key: " + err.Error())
			return
		}

		privateKeys = append(privateKeys, *pk)
		file.Close()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(privateKeys)
}
