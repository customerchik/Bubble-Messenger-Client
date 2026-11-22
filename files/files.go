package files

import (
	"bufio"
	"bytes"
	"client/config"
	"client/crypto"
	"client/request"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type SendFileRequest struct {
	Receiver string
	FileName string
}

type GetFileRequest struct {
	StorageFilepath string
}

type FilesRequest struct {
	File          []byte
	ReceiverLogin string
}

type DelFileRequest struct {
	Filename string
}

func SendFile(login, password, receiverLogin, fileName, deviceInfo, key string, file []byte) string {
	req := SendFileRequest{
		Receiver: receiverLogin,
		FileName: fileName,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "$ не удалось преобразовать запрос в JSON" + err.Error()
	}

	resp, err := request.SendRequest(login, password, deviceInfo, key, config.ForServerReq+config.MediumSizeDataServicePort+"/", "sendfile", reqBody)
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
					File:          file,
					ReceiverLogin: receiverLogin,
				}

				body, err := json.Marshal(&postReqBody)
				if err != nil {
					return "$ не удалось создать запрос для загрузки файла"
				}

				body, _ = crypto.StringEncryptByte(body, key)

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
		return "$ Проверьте правильность ввода данных "
	case http.StatusForbidden:
		return "$ Не правильный логин или пароль"
	case http.StatusInternalServerError:
		body, _ := io.ReadAll(resp.Body)
		decryptResp, _ := crypto.StringDecrypt(string(body), key)
		return "$ Внутренняя ошибка сервера" + decryptResp
	default:
		return "$ Неизвестная ошибка"
	}

	return "$ Произошла неизвестная ошибка"
}

func GetFile(login, password, fileStoragePath, deviceInfo, key string) []byte {
	req := GetFileRequest{
		StorageFilepath: fileStoragePath,
	}

	reqBody, err := json.Marshal(&req)
	if err != nil {
		return []byte("$ marshall json" + err.Error())
	}

	resp, err := request.SendRequest(login, password, deviceInfo, key, config.ForServerReq+config.MainServicePort+"/", "getfile", reqBody)
	if err != nil {
		return []byte("$ Не удачный реквест" + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte("$ Не удается обработать ответ сервера")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		tempPath, _ := crypto.StringDecrypt(string(body), key)

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		respFilesBytes, err := client.Get(config.FilesService + "/" + tempPath)
		if err != nil {
			return []byte{}
		}
		defer respFilesBytes.Body.Close()

		respBody, err := io.ReadAll(respFilesBytes.Body)
		if err != nil {
			return []byte{}
		}

		respBody, _ = crypto.StringDecryptByte(respBody, key)

		return respBody
	case http.StatusBadRequest:
		return []byte("$ Проверьте правильность ввода данных")
	case http.StatusForbidden:
		return []byte("$ Не правильный логин или пароль")
	case http.StatusInternalServerError:
		resp, _ := crypto.StringDecrypt(string(body), key)
		return []byte("$ Внутренняя ошибка сервера" + resp)
	default:
		return []byte("$ Неизвестная ошибка")
	}
}

func DelFile(login, password, fileName, deviceInfo, key string) string {
	req := DelFileRequest{
		Filename: fileName,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "$ Не удалось преобразовать запрос в JSON" + err.Error()
	}

	resp, err := request.SendRequest(login, password, deviceInfo, key, config.ForServerReq+config.MediumSizeDataServicePort+"/", "delfile", reqBody)
	if err != nil {
		return "$ не удалось отправить реквест " + err.Error()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "$ Не удается обработать ответ сервера"
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return "Успешно удалено"
	case http.StatusBadRequest:
		return "$ Проверьте правильность ввода данных " + string(body)
	case http.StatusForbidden:
		return "$ Не правильный логин или пароль" + string(body)
	case http.StatusInternalServerError:
		resp, _ := crypto.StringDecrypt(string(body), key)
		return "$ Внутренняя ошибка сервера " + resp
	default:
		return "$ Неизвестная ошибка" + string(body)
	}
}
