package globals

type Response struct {
	Status  string      `json:"status" yaml:"status"`
	Message string      `json:"message" yaml:"message"`
	Data    interface{} `json:"data"	yaml:"data"`
}

type EntryRequest struct {
	Database string              `json:"database"`
	Entries  []EntryRequestEntry `json:"entries"`
}

type EntryRequestEntry struct {
	Table string                 `json:"table"`
	Data  map[string]interface{} `json:"data"`
}

type EntryCreationResponse struct {
	EntryReceipts []EntryCreationResponseEntryReceipt `json:"receipts"`
}

type EntryCreationResponseEntryReceipt struct {
	EntryID      string `json:"entryID"`
	Table        string `json:"table"`
	RequestIndex int    `json:"requestIndex"`
}

type AuthRequestBody struct {
	Database string `json:"database,omitempty" yaml:"database,omitempty"`
	Data     struct {
		ID       string                 `json:"id,omitempty" yaml:"id,omitempty"`
		Name     string                 `json:"name,omitempty" yaml:"name,omitempty"`
		Password string                 `json:"password,omitempty" yaml:"password,omitempty"`
		JWT      string                 `json:"jwt,omitempty" yaml:"jwt,omitempty"`
		Roles    []string               `json:"roles,omitempty" yaml:"roles,omitempty"`
		Select   []string               `json:"__select,omitempty" yaml:"__select,omitempty"`
		Update   map[string]interface{} `json:"__update,omitempty" yaml:"__update,omitempty"`
	} `json:"data" yaml:"data"`
}
