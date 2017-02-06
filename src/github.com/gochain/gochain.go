/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"strconv"
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type UsedTimesMarshal struct {
	Holder string
	Times [][]string
}

type UsedTimesUnMarshal struct {
	Holder string `json:"holder"`
	Times [][]string `json:"times"`
}

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("########### gochain Init ###########")
	_, args := stub.GetFunctionAndParameters()
	var entity string    // entity
	var time int // time
	var err error

	if (len(args) % 2) != 0 {
		return shim.Error("Incorrect number of arguments. Expecting an even number")
	}

	// Initialize the chaincode
	for i := 0; i < len(args); i += 1 {
		entity = args[i]
		if (i % 2) == 0 {
			time, err = strconv.Atoi(args[i+1])
			if err != nil {
				return shim.Error("Expecting integer value for asset holding")
			}
			fmt.Printf("user = %c, time = %d\n", entity, time)

			// Write the state to the ledger
			result := t.storeUsedTime(stub,
				[][]string{
					[]string{strconv.Itoa(time), ""}}, entity)
			if result == false {
				return shim.Error(err.Error())
			}

		}
	}

	return shim.Success(nil)
}

func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("Unknown supported call")
}

// Transaction makes payment of X units from A to B
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("########### gochain Invoke ###########")
	function, args := stub.GetFunctionAndParameters()

	if function != "invoke" {
		return shim.Error("Unknown function call")
	}

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting at least 2")
	}

	if args[0] == "delete" {
		// Deletes an entity from its state
		return t.delete(stub, args)
	}

	if args[0] == "query" {
		// queries an entity state
		return t.query(stub, args)
	}

	if args[0] == "queryAll" {
		// queries an entity state
		return t.queryAll(stub, args)
	}

	if args[0] == "move" {
		// Deletes an entity from its state
		return t.move(stub, args)
	}
	return shim.Error("Unknown action, check the first argument, must be one of 'delete', 'query', or 'move'")
}

func (t *SimpleChaincode) move(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// must be an invoke
	var From, To string    // Entities
	var FromTimeRange, ToTimeRange [][]string // current ownership time
	var time int          // time
	var err error

	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4, function followed by 2 names and 1 value")
	}

	From = args[1]
	To = args[2]

	// Get the time
	time, err = strconv.Atoi(args[3])
	if err != nil {
		return shim.Error("Invalid transaction amount, expecting a integer value")
	}

	// Get the state from the ledger
	FromTimeBytes, err := stub.GetState(From)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if FromTimeBytes == nil {
		return shim.Error("Entity not found")
	}
	FromTimeRange = t.getUsedTime(stub, FromTimeBytes).Times

	ToTimeBytes, err := stub.GetState(To)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if ToTimeBytes == nil {
		ToTimeRange = [][]string{}
	} else {
		ToTimeRange = t.getUsedTime(stub, ToTimeBytes).Times
	}

	// store the new state
	ToTimeRange = append(ToTimeRange, []string{strconv.Itoa(time), ""})
	FromTimeRange[len(FromTimeRange)-1][1] = strconv.Itoa(time)
	fmt.Println(FromTimeRange);
	fmt.Println(ToTimeRange);

	// Write the state back to the ledger
	resultFrom := t.storeUsedTime(stub, FromTimeRange, From)
	if resultFrom == false {
		return shim.Error(err.Error())
	}

	resultTo := t.storeUsedTime(stub, ToTimeRange, To)
	if resultTo == false {
		return shim.Error(err.Error())
	}

	return shim.Success(nil);
}

// Deletes an entity from state
func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	A := args[1]

	// Delete the key from the state in ledger
	err := stub.DelState(A)
	if err != nil {
		return shim.Error("Failed to delete state")
	}

	return shim.Success(nil)
}

// Query callback representing the query of a chaincode
func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	var A string // Entities
	var err error

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	A = args[1]

	// Get the state from the ledger
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	if Avalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(Avalbytes)
}

func (t *SimpleChaincode) queryAll(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	var A, B string // Entities
	var err error
	var States []byte

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	A = args[1]
	B = args[2]

	// Get the state from the ledger
	Iterator, err := stub.RangeQueryState(A, B)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "-" + B + "\"}"
		return shim.Error(jsonResp)
	}

	for Iterator.HasNext() {
		Key, Valbytes, err := Iterator.Next()
		if err != nil {
			jsonResp := "{\"Error\":\"Failed to get state for " + Key + "\"}"
			return shim.Error(jsonResp)
		}
		States = append(States, Valbytes...)
	}

	return shim.Success(States)
}

func (t *SimpleChaincode) storeUsedTime(stub shim.ChaincodeStubInterface, time [][]string, entity string) bool {
	usedTimesMarshal := &UsedTimesMarshal{Times: time, Holder: entity}
	usedTimes, err := json.Marshal(usedTimesMarshal)
	if err != nil {
		return false
	}

	err = stub.PutState(entity, usedTimes)
	if err != nil {
		return false
	}

	return true
}

func (t *SimpleChaincode) getUsedTime(stub shim.ChaincodeStubInterface, timeBytes []byte) UsedTimesUnMarshal {
	usedTimesUnMarshal := UsedTimesUnMarshal{}
	json.Unmarshal(timeBytes, &usedTimesUnMarshal)

	return usedTimesUnMarshal
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
