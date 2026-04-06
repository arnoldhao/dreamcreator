package dto

type LibraryOperationsToolRequest struct {
	LibraryID string   `json:"libraryId,omitempty"`
	Status    []string `json:"status,omitempty"`
	Kinds     []string `json:"kinds,omitempty"`
	Query     string   `json:"query,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
}

type LibraryFilesToolRequest struct {
	LibraryID string `json:"libraryId,omitempty"`
}

type LibraryOperationStatusRequest struct {
	OperationID string `json:"operationId"`
}
