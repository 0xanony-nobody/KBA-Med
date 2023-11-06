
package contracts


import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric/common/util"
)

type PharmaChaincode struct {
	contractapi.Contract
}

type Medicine struct {
	Name           string    `json:"name"`
	Quantity       int       `json:"quantity"`
	ManufactureDate time.Time `json:"manufactureDate"`
	ExpiryDate     time.Time `json:"expiryDate"`
	Owner          string    `json:"owner"`
}

type MedicineHistory struct {
	TxID      string    `json:"txId"`
	Value     Medicine  `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type MedicineRequest struct {
	MedicineName string `json:"medicineName"`
	Requester    string `json:"requester"`
	Details      string `json:"details"`
}

func (c *PharmaChaincode) AddMedicine(ctx contractapi.TransactionContextInterface, name string, quantity int, manufactureDate string, expiryDate string) error {
	// Check if medicine with the same name already exists
	existingMedicine, err := ctx.GetStub().GetState(name)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingMedicine != nil {
		return fmt.Errorf("medicine with name %s already exists", name)
	}

	// Parse dates
	manufactureTime, err := time.Parse(time.RFC3339, manufactureDate)
	if err != nil {
		return fmt.Errorf("failed to parse manufacture date: %v", err)
	}

	expiryTime, err := time.Parse(time.RFC3339, expiryDate)
	if err != nil {
		return fmt.Errorf("failed to parse expiry date: %v", err)
	}

	// Get the submitting organization
	owner, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get submitting organization: %v", err)
	}

	// Create a new Medicine instance
	medicine := Medicine{
		Name:           name,
		Quantity:       quantity,
		ManufactureDate: manufactureTime,
		ExpiryDate:     expiryTime,
		Owner:          owner,
	}

	// Convert the Medicine instance to JSON
	medicineJSON, err := json.Marshal(medicine)
	if err != nil {
		return fmt.Errorf("failed to marshal medicine to JSON: %v", err)
	}

	// Put the Medicine instance to the world state
	err = ctx.GetStub().PutState(name, medicineJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %v", err)
	}

	return nil
}

func (c *PharmaChaincode) DeleteMedicine(ctx contractapi.TransactionContextInterface, name string) error {
	// Check if medicine exists
	existingMedicine, err := ctx.GetStub().GetState(name)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingMedicine == nil {
		return fmt.Errorf("medicine with name %s does not exist", name)
	}

	// Delete the medicine from the world state
	err = ctx.GetStub().DelState(name)
	if err != nil {
		return fmt.Errorf("failed to delete state: %v", err)
	}

	return nil
}

func (c *PharmaChaincode) ListMedicines(ctx contractapi.TransactionContextInterface) ([]*Medicine, error) {
	// Get all medicines from the world state
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get state by range: %v", err)
	}
	defer resultsIterator.Close()

	// Iterate through the results and unmarshal the medicines
	var medicines []*Medicine
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate over query results: %v", err)
		}

		var medicine Medicine
		err = json.Unmarshal(queryResponse.Value, &medicine)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal medicine JSON: %v", err)
		}

		medicines = append(medicines, &medicine)
	}

	// Sort the medicines by name in ascending order
	sort.Slice(medicines, func(i, j int) bool {
		return medicines[i].Name < medicines[j].Name
	})

	return medicines, nil
}

func (c *PharmaChaincode) ShowMedicineHistory(ctx contractapi.TransactionContextInterface, name string) ([]*MedicineHistory, error) {
	// Get the history of the medicine
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for key %s: %v", name, err)
	}
	defer resultsIterator.Close()

	// Iterate through the results and unmarshal the history
	var medicineHistory []*MedicineHistory
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate over history query results: %v", err)
		}

		var txValue Medicine
		err = json.Unmarshal(queryResponse.Value, &txValue)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal medicine JSON from history: %v", err)
		}

		historyEntry := &MedicineHistory{
			TxID:      queryResponse.TxId,
			Value:     txValue,
			Timestamp: queryResponse.Timestamp,
		}

		medicineHistory = append(medicineHistory, historyEntry)
	}

	return medicineHistory, nil
}

func (c *PharmaChaincode) RequestMedicine(ctx contractapi.TransactionContextInterface, name string, details string) error {
	// Check if medicine exists
	existingMedicine, err := ctx.GetStub().GetState(name)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingMedicine == nil {
		return fmt.Errorf("medicine with name %s does not exist", name)
	}

	// Get the submitting organization
	requester, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get submitting organization: %v", err)
	}

	// Define the allowed organizations for requests (adjust as needed)
	allowedOrgs := map[string]bool{
		"ProducerMSP": true,
		"SupplierMSP": true,
		// Add other allowed organizations
	}

	// Check if the submitting organization is allowed to make requests
	if !allowedOrgs[requester] {
		return fmt.Errorf("organization '%s' is not allowed to make requests", requester)
	}

	// Create a unique key for the request using the medicine name
	requestKey := fmt.Sprintf("request_%s_%s", requester, name)

	// Check if the request already exists
	existingRequest, err := ctx.GetStub().GetState(requestKey)
	if err != nil {
		return fmt.Errorf("failed to read request: %v", err)
	}

	if existingRequest != nil {
		return fmt.Errorf("request for medicine '%s' already exists", name)
	}

	// Create a new request
	request := MedicineRequest{
		MedicineName: name,
		Requester:    requester,
		Details:      details,
	}

	// Convert the request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request to JSON: %v", err)
	}

	// Save the request to the ledger
	err = ctx.GetStub().PutState(requestKey, requestJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %v", err)
	}

	return nil
}
