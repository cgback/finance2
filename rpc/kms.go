package ryrpc

import "errors"

func KmsHistory(uid, field string, hide bool) (string, error) {

	var ret string
	vv := kms_history_t{
		Uid:   uid,
		Hide:  hide,
		Field: field,
	}

	err := client.Call("/kms/history", vv, &ret)
	return ret, err
}

func KmsEncrypt(uid string, data map[string]string) (string, error) {

	var ret string
	vv := kms_encrypt_t{
		Uid:  uid,
		Vals: data,
	}

	err := client.Call("/kms/encrypt", vv, &ret)
	return ret, err
}

func KmsDecryptOne(uid string, hide bool, field []string) (map[string]string, error) {

	ret := map[string]string{}
	vv := kms_decrypt_one_t{
		Uid:   uid,
		Hide:  hide,
		Field: field,
	}

	err := client.Call("/kms/decrypt/one", vv, &ret)
	return ret, err
}

func KmsDecryptAll(uids []string, hide bool, field []string) (map[string]map[string]string, error) {

	var ret map[string]map[string]string

	if len(uids) == 0 {
		return ret, errors.New("uids = nil")
	}

	vv := kms_decrypt_all_t{
		Uids:  uids,
		Hide:  hide,
		Field: field,
	}

	err := client.Call("/kms/decrypt/all", vv, &ret)
	return ret, err
}
