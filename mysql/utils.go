package mysql

import (
	"crypto/sha1"
	"math/rand"
	"strings"
	"time"
)

// sha1(password) ^ sha1(salt + sha1(sha1(password)))
func Sha1Password(password string, salt []byte) []byte {
	crypt := sha1.New()

	_, _ = crypt.Write([]byte(password))
	shaPass1 := crypt.Sum(nil)

	crypt.Reset()
	_, _ = crypt.Write(shaPass1)
	shaPass2 := crypt.Sum(nil)

	crypt.Reset()
	_, _ = crypt.Write(salt)
	_, _ = crypt.Write(shaPass2)
	sha3 := crypt.Sum(nil)

	for k, v := range shaPass1 {
		sha3[k] = v ^ sha3[k]
	}

	return sha3
}

func RandomSalt(size int) []byte {
	buf := make([]byte, size)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		buf[i] = byte(rand.Intn(127))
		if buf[i] == 0 || buf[i] == byte('$') {
			buf[i]++
		}
	}
	return buf
}

//判断是否写操作
func IsWrite(sql string) bool {
	sql = strings.ToUpper(sql)
	if strings.HasPrefix(sql, "INSERT ") {
		return true
	}
	if strings.HasPrefix(sql, "UPDATE ") {
		return true
	}
	if strings.HasPrefix(sql, "DELETE ") {
		return true
	}
	if strings.HasPrefix(sql, "DROP ") {
		return true
	}
	if strings.HasSuffix(sql, " FOR UPDATE") {
		return true
	}
	//事务
	if strings.HasPrefix(sql, "BEGIN") {
		return true
	}
	if strings.HasPrefix(sql, "COMMIT") {
		return true
	}
	if strings.HasPrefix(sql, "ROLLBACK") {
		return true
	}
	if strings.HasPrefix(sql, "START TRANSACTION") {
		return true
	}
	if strings.HasPrefix(sql, "SET AUTOCOMMIT") {
		return true
	}

	return false
}
