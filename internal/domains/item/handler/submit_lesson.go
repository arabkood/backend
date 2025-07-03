package handler

import (
	"encoding/base64"
	"strconv"

	appError "github.com/arabkood/backend/pkg/errors"
)

func handleLesson(data DataType) (int, error) {
	encoded, ok := data["_$"].(string)
	if !ok {
		return 0, appError.ErrorInvalidInput()
	}
	/*
	 * Lore Time:
	 *
	 * This is a coding teaching website and since hacking can be considered coding too
	 * I wanna make it possible to cheat in the lesson quizes and get full score without answering,
	 * but we should not make it that easy, therefore the obfuscation
	 */
	salt := 69
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, err
	}

	raw, err := strconv.Atoi(string(decodedBytes))
	if err != nil {
		return 0, err
	}

	percent := raw / salt

	// Clamp to 0–100 range: above 100 or below 0 is dangerous and not allowed :)
	if percent > 100 {
		percent = 100
	} else if percent < 0 {
		percent = 0
	}

	return percent, nil
}
