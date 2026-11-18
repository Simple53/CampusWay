package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/nnu-connect/nnu-connect/log"
)

type PasswordLogin struct {
	Username      string
	Password      string
	Domain        string
	GraphCodeFile string
}

func (m PasswordLogin) AuthType() string {
	return "auth/psw"
}

func (m PasswordLogin) LoginDomain() string {
	return m.Domain
}

func (m PasswordLogin) login(s *Session, info AuthInfo) error {
	return s.loginAuthPsw(m.Username, m.Password, info.LoginDomain, m.GraphCodeFile)
}

func (s *Session) loginAuthPsw(username, password, loginDomain, graphCodeFile string) error {
	process := func(graphCheckCode string) (int, error) {
		return s.pswImpl(username, password, loginDomain, graphCheckCode)
	}
	return s.withGraphCheckCode(process, graphCodeFile)
}

func (s *Session) pswImpl(username, password, loginDomain, graphCheckCode string) (int, error) {
	log.Println("Perform POST /passport/v1/auth/psw")

	N := new(big.Int)
	N.SetString(s.pubKey, 16)
	E, _ := strconv.Atoi(s.pubKeyExp)
	pub := &rsa.PublicKey{N: N, E: E}

	msg := []byte(password + "_" + s.antiReplayRand)
	cipherBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pub, msg)
	if err != nil {
		return 0, err
	}
	encryptedPwd := hex.EncodeToString(cipherBytes)

	data := map[string]interface{}{
		"username":    username + "@" + loginDomain,
		"password":    encryptedPwd,
		"rememberPwd": "0",
	}

	if graphCheckCode != "" {
		data["graphCheckCode"] = graphCheckCode
	}
	postBody, _ := json.Marshal(data)

	u := s.baseURL + "/passport/v1/auth/psw"
	req, _ := http.NewRequest("POST", u+"?"+WithSharedParams(nil).Encode(), bytes.NewReader(postBody))
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("x-csrf-token", s.csrfToken)
	req.Header.Set("x-sdp-env", s.env)
	req.Header.Set("x-sdp-traceid", s.randSdpId())

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)
	log.DebugPrintf("Received psw: %s", string(body))

	trimmedBody := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("网关认证请求错误 (HTTP %d): %s", resp.StatusCode, trimmedBody)
	}
	if strings.HasPrefix(trimmedBody, "<") {
		return 0, fmt.Errorf("网关返回了非预期页面 (可能是验证码超时或请求被拦截): %s", trimmedBody)
	}

	var re struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Ticket               string `json:"ticket"`
			GraphCheckCodeEnable int    `json:"graphCheckCodeEnable"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &re)
	if err != nil {
		return 0, err
	}
	if re.Code != 0 || re.Message != "" {
		log.Printf("Code: %d, Message: %s", re.Code, re.Message)
	}
	log.DebugPrintf("Parsed psw: %+v", re)

	s.ticket = re.Data.Ticket
	if re.Data.GraphCheckCodeEnable != 1 && (re.Code != 0 || s.ticket == "") {
		return 0, fmt.Errorf("password authentication rejected (code %d): %s", re.Code, re.Message)
	}

	return re.Data.GraphCheckCodeEnable, nil
}
