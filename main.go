package main

import (
	"log"
        "github.com/0xanony-nobody/KBA-Med/contracts/"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	medContract := new(contracts.PharmaChaincode)
	

	chaincode, err := contractapi.NewChaincode(PharmaChaincode)

	if err != nil {
		log.Panicf("Could not create chaincode." + err.Error())
	}

	err = chaincode.Start()

	if err != nil {
		log.Panicf("Failed to start chaincode. " + err.Error())
	}
}
