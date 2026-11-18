package files

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"client/config"
	"client/crypto"
	"client/request"
)

type GetAvatarRequest struct {
	LoginForSearch string
}

func SetAvatar(login, password, deviceInfo, key string, avatar []byte) string {
	resp, err := request.SendRequest(login, password, deviceInfo, key, config.ForServerReq+config.MediumSizeDataServicePort+"/", "setavatar", nil)
	if err != nil {
		return "$ не удалось отправить реквест " + err.Error()
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			msg := scanner.Text()

			switch {
			case strings.Contains(msg, "Error"):
				return "$ " + msg

			case msg == "Success":
				return "Успешно отправлен"

			default:
				postReqBody := FilesRequest{
					File:          avatar,
					ReceiverLogin: "",
				}

				body, err := json.Marshal(&postReqBody)
				if err != nil {
					return "$ не удалось создать запрос для загрузки файла"
				}

				resp, err := http.Post(config.FilesService+msg, "", bytes.NewReader(body))
				if err != nil {
					return "$ не удалось отправить реквест на загрузку файла"
				}
				defer resp.Body.Close()

				body, _ = io.ReadAll(resp.Body)
				if resp.StatusCode != http.StatusOK {
					return "Ошибка при отправке файла"
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return "$ ошибка чтения ответа: " + err.Error()
		}
	case http.StatusBadRequest:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "Не удается обработать ответ сервера"
		}
		resp, _ := crypto.StringDecrypt(string(body), key)
		return "$ " + string(resp)
	case http.StatusForbidden:
		return "$ Не правильный логин или пароль"
	case http.StatusInternalServerError:
		return "$ Внутренняя ошибка сервера"
	default:
		return "$ Неизвестная ошибка"
	}

	return "$ Произошла неизвестная ошибка"
}

func GetAvatar(login, password, loginForSearch, deviceInfo, key string) []byte {
	req := GetAvatarRequest{
		LoginForSearch: loginForSearch,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return []byte("$ не удалось преобразовать запрос в JSON" + err.Error())
	}

	resp, err := request.SendRequest(login, password, deviceInfo, key, config.ForServerReq+config.MediumSizeDataServicePort+"/", "getavatar", reqBody)
	if err != nil {
		return []byte("$ не удалось отправить реквест " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte("$ ошибка чтения ответа")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		storagePath, err := crypto.StringDecrypt(string(body), key)
		if err != nil {
			return []byte{}
		}

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		respAvatarBytes, err := client.Get(config.FilesService + "/" + storagePath)
		if err != nil {
			return []byte{}
		}
		defer respAvatarBytes.Body.Close()

		respBody, err := io.ReadAll(respAvatarBytes.Body)
		if err != nil {
			return []byte{}
		}

		return respBody
	case http.StatusNoContent:
		return []byte{}
	case http.StatusBadRequest:
		return []byte{}
	case http.StatusInternalServerError:
		return []byte{}
	default:
		return []byte{}
	}
}
