package tunnel

type Request struct {
    ID      string            `json:"id"`
    Method  string            `json:"method"`
    Path    string            `json:"path"`
    Headers map[string]string `json:"headers"`
    Body    []byte            `json:"body"`
}

type Response struct {
    ID      string            `json:"id"`
    Status  int               `json:"status"`
    Headers map[string]string `json:"headers"`
    Body    []byte            `json:"body"`
}
