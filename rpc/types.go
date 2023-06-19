package ryrpc

type kms_encrypt_t struct {
	Uid  string            `json:"uid" cbor:"uid"`
	Vals map[string]string `json:"vals" cbor:"vals"`
}

type kms_history_t struct {
	Uid   string `json:"uid" cbor:"uid"`
	Hide  bool   `json:"hide" cbor:"hide"`
	Field string `json:"field" cbor:"field"`
}

type kms_decrypt_one_t struct {
	Uid   string   `json:"uid" cbor:"uid"`
	Hide  bool     `json:"hide" cbor:"hide"`
	Field []string `json:"field" cbor:"field"`
}

type kms_decrypt_all_t struct {
	Uids  []string `json:"uids" cbor:"uids"`
	Hide  bool     `json:"hide" cbor:"hide"`
	Field []string `json:"field" cbor:"field"`
}

type deposit_flow_check_resp_t struct {
	Ok     bool   `json:"ok" cbor:"ok"`
	Amount string `json:"amount" cbor:"amount"`
}
