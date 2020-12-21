package common

// import (
// 	"bytes"
// 	"compress/gzip"
// 	"errors"
// 	"io/ioutil"
// )

// // GZipFromBytes receives bytes array
// // and compresses that bytes array using gzip
// func GZipFromBytes(src []byte) ([]byte, error) {
// 	if len(src) == 0 {
// 		return []byte{}, errors.New("input to gzip compress is empty")
// 	}
// 	var b bytes.Buffer
// 	gz := gzip.NewWriter(&b)
// 	if _, err := gz.Write(src); err != nil {
// 		return nil, err
// 	}
// 	if err := gz.Flush(); err != nil {
// 		return nil, err
// 	}
// 	if err := gz.Close(); err != nil {
// 		return nil, err
// 	}
// 	return b.Bytes(), nil
// }

// // GZipToBytes receives bytes array which is compressed data using gzip
// // returns decompressed bytes array
// func GZipToBytes(src []byte) ([]byte, error) {
// 	var br bytes.Buffer
// 	br.Write(src)
// 	gz, err := gzip.NewReader(&br)
// 	if err != nil {
// 		return nil, err
// 	}
// 	resultBytes, err := ioutil.ReadAll(gz)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err := gz.Close(); err != nil {
// 		return nil, err
// 	}
// 	return resultBytes, nil
// }
