package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"richtalk/api/internal/httpx"
)

const maxUploadSize = 25 << 20 // 25 MB

// mimeAllowList maps accepted MIME types to attachment categories.
// video/webm is included because some browsers report that for audio-only MediaRecorder streams.
var mimeAllowList = map[string]string{
	"image/jpeg":  "image",
	"image/png":   "image",
	"image/webp":  "image",
	"image/gif":   "image",
	"audio/webm":  "audio",
	"video/webm":  "audio",
	"audio/ogg":   "audio",
	"audio/mpeg":  "audio",
	"audio/mp4":   "audio",
	"audio/wav":   "audio",
}

var mimeToExt = map[string]string{
	"image/jpeg":  ".jpg",
	"image/png":   ".png",
	"image/webp":  ".webp",
	"image/gif":   ".gif",
	"audio/webm":  ".webm",
	"video/webm":  ".webm",
	"audio/ogg":   ".ogg",
	"audio/mpeg":  ".mp3",
	"audio/mp4":   ".m4a",
	"audio/wav":   ".wav",
}

type UploadHandler struct {
	dir     string
	baseURL string
}

func NewUploadHandler(dir, baseURL string) *UploadHandler {
	return &UploadHandler{dir: dir, baseURL: baseURL}
}

type uploadResponse struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file_too_large", "Файл слишком большой (макс. 25 МБ)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "no_file", "Поле file не найдено в запросе")
		return
	}
	defer file.Close()

	// Browser sets this correctly (e.g. "audio/webm;codecs=opus" for voice recordings).
	contentType := header.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	attType, ok := mimeAllowList[contentType]
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "unsupported_type", "Разрешены только изображения и аудио")
		return
	}

	ext := mimeToExt[contentType]
	filename := uuid.New().String() + ext
	dstPath := filepath.Join(h.dir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "storage_error", "Ошибка сохранения файла")
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(dstPath)
		httpx.Error(w, http.StatusInternalServerError, "storage_error", "Ошибка записи файла")
		return
	}

	name := header.Filename
	if name == "" {
		name = "file" + ext
	}

	url := strings.TrimRight(h.baseURL, "/") + "/" + filename
	httpx.JSON(w, http.StatusCreated, uploadResponse{URL: url, Type: attType, Name: name, Size: size})
}
