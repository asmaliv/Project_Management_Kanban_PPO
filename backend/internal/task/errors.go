package task

import "errors"

var (
	ErrTaskNotFound       = errors.New("task tidak ditemukan")
	ErrProjectNotFound    = errors.New("proyek tidak ditemukan")
	ErrColumnNotFound     = errors.New("kolom tidak ditemukan")
	ErrForbidden          = errors.New("anda tidak memiliki izin untuk melakukan aksi ini")
	ErrProjectArchived    = errors.New("proyek sudah diarsipkan, tidak dapat mengubah task")
	ErrVersionConflict    = errors.New("task sudah diubah oleh pengguna lain. Muat ulang papan.")
	ErrWIPLimitExceeded   = errors.New("wip limit pada kolom tujuan telah tercapai")
	ErrMakerCheckerDone   = errors.New("hanya Project Manager yang dapat memindahkan task ke atau dari kolom Done.")
	ErrAssigneeNotMember  = errors.New("assignee harus merupakan member atau owner proyek")
	ErrInvalidInput       = errors.New("input tidak valid")
	ErrCommentBodyEmpty   = errors.New("komentar tidak boleh kosong")
	ErrCommentBodyTooLong = errors.New("komentar maksimal 5000 karakter")
)
