package main

import (
	"encoding/hex"
	"fmt"
	"github.com/duminghui/go-rpcclient"
	"github.com/duminghui/go-rpcclient/cmdjson"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

const (
	numBase = 100000000.0
	feeFix  = 5000000.0
)

var log = logrus.New()

func getInputsAndSendAmount(c *rpcclient.Client, addresses []string, limitAmount float64, maxSumAmount float64) ([]cmdjson.TransactionInput, float64, error) {
	maxAmountFix := maxSumAmount * numBase
	amountSumFix := 0.0
	limitMaxConf := 1500000
	minConf := 10
	maxConf := 10000
	confSize := 10000
	var inputs []cmdjson.TransactionInput
OUT:
	for {
		unspentList, err := c.ListUnspent(cmdjson.Int(minConf), cmdjson.Int(maxConf), &addresses)
		if err != nil {
			return nil, 0, err
		}
		for _, unspent := range unspentList {
			if unspent.Spendable && unspent.Amount <= limitAmount {
				amountSumFix += unspent.Amount * numBase
				input := cmdjson.TransactionInput{
					Txid: unspent.TxID,
					Vout: unspent.Vout,
				}
				inputs = append(inputs, input)
				if amountSumFix >= maxAmountFix {
					break OUT
				}
			}
		}
		log.Infof("sum:%17.8f, inputs:%v, minConf:%v, maxConf:%v", amountSumFix/numBase, len(inputs), minConf, maxConf)
		if maxConf >= limitMaxConf {
			break
		}
		minConf = maxConf + 1
		maxConf = minConf + confSize - 1
	}
	sendAmount := (amountSumFix - feeFix) / numBase
	if sendAmount <= 0 {
		err := fmt.Errorf("send Amount<=0: %017.8f", sendAmount)
		return nil, 0, err
	}
	log.Infof("ListUnspent Use: %v, minConf:%v, maxConf:%v", len(inputs), minConf, maxConf)
	return inputs, sendAmount, nil
}

type info struct {
	User      string `yaml:"user"`
	Pass      string `yaml:"pass"`
	FromAddr1 string `yaml:"fromAddr1"`
	FromAddr2 string `yaml:"fromAddr2"`
	SendAddr  string `yaml:"sendAddr"`
	PrivKey1  string `yaml:"privKey1"`
	PrivKey2  string `yaml:"privKey2"`
}

func sendAmount() {
	data, err := ioutil.ReadFile("./data.yml")
	if err != nil {
		log.Infoln("读取配置文件出错;%v", err)
		return
	}
	info := &info{}
	err = yaml.Unmarshal(data, info)
	if err != nil {
		log.Infoln("解析配置文件出错;%v", err)
		return
	}
	fmt.Println("user", info.User)
	fmt.Println("pass", info.Pass)
	fmt.Println("addr1", info.FromAddr1)
	fmt.Println("addr2", info.FromAddr2)
	fmt.Println("sendAddr", info.SendAddr)
	fmt.Println("key1:", info.PrivKey1)
	fmt.Println("key2:", info.PrivKey2)
	log.Infoln("Type Key:")
	var key string
	_, err = fmt.Scanln(&key)
	if err != nil {
		return
	}

	user, err := toDecryptItem(info.User, key)
	if err != nil {
		log.Infoln("Decrypt User Error:%v", err)
		return
	}
	pass, err := toDecryptItem(info.Pass, key)
	if err != nil {
		log.Infoln("Decrypt Pass Error:%v", err)
		return
	}
	addr1, err := toDecryptItem(info.FromAddr1, key)
	if err != nil {
		log.Infoln("Decrypt Addr1 Error:%v", err)
		return
	}
	addr2, err := toDecryptItem(info.FromAddr2, key)
	if err != nil {
		log.Infoln("Decrypt Addr2 Error:%v", err)
		return
	}
	sendAddr, err := toDecryptItem(info.SendAddr, key)
	if err != nil {
		log.Infoln("Decrypt SendAddr Error:%v", err)
		return
	}
	privKey1, err := toDecryptItem(info.PrivKey1, key)
	if err != nil {
		log.Infoln("Decrypt PrivKey1 Error:%v", err)
		return
	}
	privKey2, err := toDecryptItem(info.PrivKey2, key)
	if err != nil {
		log.Infoln("Decrypt PrivKey2 Error:%v", err)
		return
	}
	log.Infoln("User:", user)
	log.Infoln("Pass:", pass)
	log.Infoln("Addr1:", addr1)
	log.Infoln("Addr2:", addr2)
	log.Infoln("SendAddr:", sendAddr)
	log.Infoln("privKey1:", privKey1)
	log.Infoln("privKey2:", privKey2)
	if true {
		return
	}
	config := &rpcclient.ConnConfig{
		Name:    "DST",
		Host:    "127.0.0.1:51479",
		User:    user,
		Pass:    pass,
		LogJSON: false,
	}
	client := rpcclient.New(config)
	client.Start()
	defer client.Shutdown()
	addresses := []string{addr1, addr2}
	inputs, sendAmount, err := getInputsAndSendAmount(client, addresses, 5.0, 3000.0)
	if err != nil {
		log.Errorf("getInputsAndSendAmount Error: %v", err)
		return
	}
	log.Infof("Send Amount: %017.8f", sendAmount)
	amounts := map[string]float64{
		sendAddr: sendAmount,
	}
	hexTx, err := client.CreateRawTransaction(inputs, amounts, nil)
	if err != nil {
		log.Errorf("CreateRawTransaction Error: %v", err)
		return
	}
	privKeys := *new([]string)
	privKeys = append(privKeys, privKey1)
	privKeys = append(privKeys, privKey2)
	signTxResult, err := client.SignRawTransaction(hexTx, nil, &privKeys, nil)
	if err != nil {
		log.Errorf("SignRawTransaction Error 1: %v", err)
		return
	}
	if !signTxResult.Complete {
		log.Errorf("SignRawTransaction Error 2: %v", signTxResult.Complete)
		return
	}
	txId, err := client.SendRawTransaction(signTxResult.Hex, cmdjson.Bool(false))
	if err != nil {
		logrus.Infof("SendRawTransaction Error: %v", err)
		return
	}
	log.Infof("SendRawTransaction: %v", txId)
}

func toEncrypt() {
	key := ""
	user := ""
	pass := ""
	addr1 := ""
	addr2 := ""
	privKey1 := ""
	privKey2 := ""
	fmt.Println("User", toEncryptItem(user, key))
	fmt.Println("Pass", toEncryptItem(pass, key))
	fmt.Println("Addr1", toEncryptItem(addr1, key))
	fmt.Println("Addr2", toEncryptItem(addr2, key))
	fmt.Println("privKey1", toEncryptItem(privKey1, key))
	fmt.Println("privKey2", toEncryptItem(privKey2, key))

}

func toEncryptItem(data string, key string) string {
	keyBytes := []byte(key)
	keyBytes = pkcs5Padding(keyBytes, 32)
	encryptItem := aesEncryptCFB([]byte(data), keyBytes)
	encryptStr := hex.EncodeToString(encryptItem)
	//encryptStr := base64.StdEncoding.EncodeToString(encryptItem)
	return encryptStr
}

func toDecryptItem(encryptData string, key string) (string, error) {
	bytes, err := hex.DecodeString(encryptData)
	//bytes, err := base64.StdEncoding.DecodeString(encryptData)
	if err != nil {
		log.Errorf("Decrypt Item Error:%v", err)
		return "", err
	}
	keyBytes := []byte(key)
	keyBytes = pkcs5Padding(keyBytes, 32)
	return string(aesDecryptCFB(bytes, keyBytes)), nil
}

func main() {
	//toEncrypt()
	sendAmount()
}

func init() {
	log.SetFormatter(&TextFormatter{})
}
