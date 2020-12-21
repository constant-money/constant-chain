package common

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	// "github.com/ethereum/go-ethereum/common"
	// "log"
	"math/big"
	"net"
	// "os"
	// "os/user"
	// "path/filepath"
	// "reflect"
	"runtime"
	"strconv"
	"strings"
	// "unicode"

	"github.com/pkg/errors"
)

// appDataDir returns an operating system specific directory to be used for
// storing application data for an application.  See AppDataDir for more
// details.  This unexported version takes an operating system argument
// primarily to enable the testing package to properly test the function by
// forcing an operating system that is not the currently one.
// func appDataDir(goos, appName string, roaming bool) string {
// 	if appName == "" || appName == "." {
// 		return "."
// 	}

// 	// The caller really shouldn't prepend the appName with a period, but
// 	// if they do, handle it gracefully by trimming it.
// 	appName = strings.TrimPrefix(appName, ".")
// 	appNameUpper := string(unicode.ToUpper(rune(appName[0]))) + appName[1:]
// 	appNameLower := string(unicode.ToLower(rune(appName[0]))) + appName[1:]

// 	// Get the OS specific home directory via the Go standard lib.
// 	var homeDir string
// 	usr, err := user.Current()
// 	if err == nil {
// 		homeDir = usr.HomeDir
// 	}

// 	// Fall back to standard HOME environment variable that works
// 	// for most POSIX OSes if the directory from the Go standard
// 	// lib failed.
// 	if err != nil || homeDir == "" {
// 		homeDir = os.Getenv("HOME")
// 	}

// 	switch goos {
// 	// Attempt to use the LOCALAPPDATA or APPDATA environment variable on
// 	// Windows.
// 	case "windows":
// 		// Windows XP and before didn't have a LOCALAPPDATA, so fallback
// 		// to regular APPDATA when LOCALAPPDATA is not set.
// 		appData := os.Getenv("LOCALAPPDATA")
// 		if roaming || appData == "" {
// 			appData = os.Getenv("APPDATA")
// 		}

// 		if appData != "" {
// 			return filepath.Join(appData, appNameUpper)
// 		}

// 	case "darwin":
// 		if homeDir != "" {
// 			return filepath.Join(homeDir, "Library",
// 				"Application Support", appNameUpper)
// 		}

// 	case "plan9":
// 		if homeDir != "" {
// 			return filepath.Join(homeDir, appNameLower)
// 		}

// 	default:
// 		if homeDir != "" {
// 			return filepath.Join(homeDir, "."+appNameLower)
// 		}
// 	}

// 	// Fall back to the current directory if all else fails.
// 	return "."
// }

// AppDataDir returns an operating system specific directory to be used for
// storing application data for an application.
//
// The appName parameter is the name of the application the data directory is
// being requested for.  This function will prepend a period to the appName for
// POSIX style operating systems since that is standard practice.  An empty
// appName or one with a single dot is treated as requesting the current
// directory so only "." will be returned.  Further, the first character
// of appName will be made lowercase for POSIX style operating systems and
// uppercase for Mac and Windows since that is standard practice.
//
// The roaming parameter only applies to Windows where it specifies the roaming
// application data profile (%APPDATA%) should be used instead of the local one
// (%LOCALAPPDATA%) that is used by default.
//
// Example results:
//  dir := AppDataDir("myapp", false)
//   POSIX (Linux/BSD): ~/.myapp
//   Mac OS: $HOME/Library/Application Support/Myapp
//   Windows: %LOCALAPPDATA%\Myapp
//   Plan 9: $home/myapp
// func AppDataDir(appName string, roaming bool) string {
// 	return appDataDir(runtime.GOOS, appName, roaming)
// }

// InterfaceSlice receives a slice which is a interface
// and converts it into slice of interface
// func InterfaceSlice(slice interface{}) []interface{} {
// 	s := reflect.ValueOf(slice)
// 	if s.Kind() != reflect.Slice {
// 		log.Println("InterfaceSlice() given a non-slice type")
// 		return nil
// 	}

// 	ret := make([]interface{}, s.Len())

// 	for i := 0; i < s.Len(); i++ {
// 		ret[i] = s.Index(i).Interface()
// 	}

// 	return ret
// }

// ParseListeners determines whether each listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects addresses which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func ParseListeners(addrs []string, netType string) ([]SimpleAddr, error) {
	simpleAddrs := make([]SimpleAddr, len(addrs))

	for i, addr := range addrs {
		simpleAddr, err := ParseListener(addr, netType)
		if err != nil {
			return []SimpleAddr{}, err
		}

		simpleAddrs[i] = *simpleAddr
	}

	return simpleAddrs, nil
}

// ParseListener determines whether the listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects address which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func ParseListener(addr string, netType string) (*SimpleAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// Shouldn't happen due to already being normalized.
		return nil, err
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 0 {
		return nil, errors.New("port is invalid")
	}

	var netAddr *SimpleAddr
	// Empty host or host of * on plan9 is both IPv4 and IPv6.
	if host == EmptyString || (host == "*" && runtime.GOOS == "plan9") {
		netAddr = &SimpleAddr{Net: netType + "4", Addr: addr}
		return netAddr, nil
	}

	// Strip IPv6 zone id if present since net.ParseIP does not
	// handle it.
	zoneIndex := strings.LastIndex(host, "%")
	if zoneIndex > 0 {
		host = host[:zoneIndex]
	}

	// Parse the IP.
	ip := net.ParseIP(host)
	if ip == nil {
		fmt.Printf("'%s' is not a valid IP address\n", host)
		return nil, errors.New("IP address is invalid")
	}

	// To4 returns nil when the IP is not an IPv4 address, so use
	// this determine the address type.
	if ip.To4() == nil {
		//netAddrs = append(netAddrs, simpleAddr{net: netType + "6", addr: addr})
	} else {
		netAddr = &SimpleAddr{Net: netType + "4", Addr: addr}
	}
	return netAddr, nil
}

// SliceExists receives a slice and a item in interface type
// checks whether the slice contain the item or not
// func SliceExists(slice interface{}, item interface{}) (bool, error) {
// 	s := reflect.ValueOf(slice)

// 	if s.Kind() != reflect.Slice {
// 		return false, errors.New("SliceExists() given a non-slice type")
// 	}

// 	for i := 0; i < s.Len(); i++ {
// 		interfacea := s.Index(i).Interface()
// 		if interfacea == item {
// 			return true, nil
// 		}
// 	}

// 	return false, nil
// }

// GetShardIDFromLastByte receives a last byte of public key and
// returns a corresponding shardID
func GetShardIDFromLastByte(b byte) byte {
	return byte(int(b) % MaxShardNumber)
}

// IndexOfStr receives a list of strings and a item string
// It checks whether a item is contained in list or not
// and returns the first index of the item in the list
// It returns -1 if the item is not in the list
func IndexOfStr(item string, list []string) int {
	for k, v := range list {
		if strings.Compare(item, v) == 0 {
			return k
		}
	}
	return -1
}

func IndexOfHash(item Hash, list []Hash) int {
	for k, v := range list {
		if item.IsEqual(&v) {
			return k
		}
	}
	return -1
}

// IndexOfByte receives a array of bytes and a item byte
// It checks whether a item is contained in array or not
// and returns the first index of the item in the array
// It returns -1 if the item is not in the array
func IndexOfByte(item byte, array []byte) int {
	for k, v := range array {
		if v == item {
			return k
		}
	}
	return -1
}

// IndexOfStrInHashMap receives a map[Hash]string and a value string
// It checks whether a value is contained in map or not
// It returns -1 if the item is not in the list and return 1 otherwise
func IndexOfStrInHashMap(v string, m map[Hash]string) int {
	for _, value := range m {
		if strings.Compare(value, v) == 0 {
			return 1
		}
	}
	return -1
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
// func CleanAndExpandPath(path string, defaultHomeDir string) string {
// 	// Expand initial ~ to OS specific home directory.
// 	if strings.HasPrefix(path, "~") {
// 		homeDir := filepath.Dir(defaultHomeDir)
// 		path = strings.Replace(path, "~", homeDir, 1)
// 	}

// 	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
// 	// but they variables can still be expanded via POSIX-style $VARIABLE.
// 	return filepath.Clean(os.ExpandEnv(path))
// }

// RandBigIntMaxRange generates a big int with maximum value
func RandBigIntMaxRange(max *big.Int) (*big.Int, error) {
	return rand.Int(rand.Reader, max)
}

// RandBytes generates random bytes with length
func RandBytes(length int) []byte {
	rbytes := make([]byte, length)
	rand.Read(rbytes)
	return rbytes
}

// CompareStringArray receives 2 arrays of string
// and check whether 2 arrays is the same or not
func CompareStringArray(src []string, dst []string) bool {
	if len(src) != len(dst) {
		return false
	}
	for idx, val := range src {
		if dst[idx] != val {
			return false
		}
	}
	return true
}

// BytesToInt32 converts little endian 4-byte array to int32 number
func BytesToInt32(b []byte) (int32, error) {
	if len(b) != Int32Size {
		return 0, errors.New("invalid length of input BytesToInt32")
	}

	return int32(binary.LittleEndian.Uint32(b)), nil
}

// Int32ToBytes converts int32 number to little endian 4-byte array
func Int32ToBytes(value int32) []byte {
	b := make([]byte, Int32Size)
	binary.LittleEndian.PutUint32(b, uint32(value))
	return b
}

// IntToBytes converts an integer number to 2-byte array in big endian
func IntToBytes(n int) []byte {
	if n == 0 {
		return []byte{0, 0}
	}

	a := big.NewInt(int64(n))

	if len(a.Bytes()) > 2 {
		return []byte{}
	}

	if len(a.Bytes()) == 1 {
		return []byte{0, a.Bytes()[0]}
	}

	return a.Bytes()
}

// BytesToInt reverts an integer number from 2-byte array
func BytesToInt(bytesArr []byte) int {
	if len(bytesArr) != 2 {
		return 0
	}

	numInt := new(big.Int).SetBytes(bytesArr)
	return int(numInt.Int64())
}

// BytesToUint32 converts big endian 4-byte array to uint32 number
func BytesToUint32(b []byte) (uint32, error) {
	if len(b) != Uint32Size {
		return 0, errors.New("invalid length of input BytesToUint32")
	}
	return binary.BigEndian.Uint32(b), nil
}

// Uint32ToBytes converts uint32 number to big endian 4-byte array
func Uint32ToBytes(value uint32) []byte {
	b := make([]byte, Uint32Size)
	binary.BigEndian.PutUint32(b, value)
	return b
}

// BytesToUint64 converts little endian 8-byte array to uint64 number
func BytesToUint64(b []byte) (uint64, error) {
	if len(b) != Uint64Size {
		return 0, errors.New("invalid length of input BytesToUint64")
	}
	return binary.LittleEndian.Uint64(b), nil
}

// Uint64ToBytes converts uint64 number to little endian 8-byte array
func Uint64ToBytes(value uint64) []byte {
	b := make([]byte, Uint64Size)
	binary.LittleEndian.PutUint64(b, value)
	return b
}

// Int64ToBytes converts int64 number to little endian 8-byte array
func Int64ToBytes(value int64) []byte {
	return Uint64ToBytes(uint64(value))
}

// BoolToByte receives a value in bool
// and returns a value in byte
func BoolToByte(value bool) byte {
	var bitSetVar byte
	if value {
		bitSetVar = 1
	}
	return bitSetVar
}

// AddPaddingBigInt adds padding to big int to it is fixed size
// and returns bytes array
func AddPaddingBigInt(numInt *big.Int, fixedSize int) []byte {
	numBytes := numInt.Bytes()
	lenNumBytes := len(numBytes)
	zeroBytes := make([]byte, fixedSize-lenNumBytes)
	numBytes = append(zeroBytes, numBytes...)
	return numBytes
}

// AppendSliceString is a variadic function,
// receives some lists of array of strings
// and appends them to one list of array of strings
func AppendSliceString(arrayStrings ...[][]string) [][]string {
	res := [][]string{}
	for _, arrayString := range arrayStrings {
		res = append(res, arrayString...)
	}
	return res
}

type ErrorSaver struct {
	err error
}

func (s *ErrorSaver) Save(errs ...error) error {
	if s.err != nil {
		return s.err
	}
	for i, err := range errs {
		if err != nil {
			s.err = errors.WithMessagef(err, "errSaver #%d", i)
			return s.err
		}
	}
	return nil
}

func (s *ErrorSaver) Get() error {
	return s.err
}

// CheckError receives a list of errors
// returns the first error which is not nil
func CheckError(errs ...error) error {
	errSaver := &ErrorSaver{}
	return errSaver.Save(errs...)
}

// GetENV to get environment variable by key
// func GetENV(key, fallback string) string {
// 	value, ok := os.LookupEnv(key)
// 	if !ok {
// 		return fallback
// 	}
// 	return value
// }

func GetValidStaker(committees []string, stakers []string) []string {
	validStaker := []string{}
	for _, staker := range stakers {
		flag := false
		for _, committee := range committees {
			if strings.Compare(staker, committee) == 0 {
				flag = true
				break
			}
		}
		if !flag {
			validStaker = append(validStaker, staker)
		}
	}
	return validStaker
}

func GetShardChainKey(shardID byte) string {
	return ShardChainKey + "-" + strconv.Itoa(int(shardID))
}

func Uint16ToBytes(v uint16) [2]byte {
	var res [2]byte
	res[0] = uint8(v >> 8)
	res[1] = uint8(v & 0xff)
	return res
}

func BytesToUint16(b [2]byte) uint16 {
	return uint16(b[0])<<8 + uint16(b[1])
}

func BytesSToUint16(b []byte) (uint16, error) {
	if len(b) != 2 {
		return 0, errors.New("Cannot convert BytesSToUint16: length of byte is not 2")
	}
	var bytes [2]byte
	copy(bytes[:], b[:2])
	return BytesToUint16(bytes), nil
}

// func IsPortalToken(tokenIDStr string) bool {
// 	isExisted, _ := SliceExists(PortalSupportedIncTokenIDs, tokenIDStr)
// 	return isExisted
// }

// func IsPortalExchangeRateToken(tokenIDStr string) bool {
// 	return IsPortalToken(tokenIDStr) || tokenIDStr == PRVIDStr
// }

// CopyBytes returns an exact copy of the provided bytes.
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// Hex2Bytes returns the bytes represented by the hexadecimal string str.
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

// FromHex returns the bytes represented by the hexadecimal string s.
// s may be prefixed with "0x".
func FromHex(s string) []byte {
	if has0xPrefix(s) {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

// AssertAndConvertStrToNumber asserts and convert a passed input to uint64 number
func AssertAndConvertStrToNumber(numStr interface{}) (uint64, error) {
	assertedNumStr, ok := numStr.(string)
	if !ok {
		return 0, errors.New("Could not assert the passed input to string")
	}
	return strconv.ParseUint(assertedNumStr, 10, 64)
}

func IndexOfUint64(target uint64, arr []uint64) int {
	for i, v := range arr {
		if v == target {
			return i
		}
	}
	return -1
}

// func TokenHashToString(h *Hash) string {
// 	var propertyID [HashSize]byte
// 	copy(propertyID[:], h[:])
// 	propID := common.Hash(propertyID)
// 	return propID.String()
// }

func TokenStringToHash(s string) (*Hash, error) {
	return Hash{}.NewHashFromStr(s)
}