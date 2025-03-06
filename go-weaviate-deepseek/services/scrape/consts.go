package scrape

const (
// entryURL = "https://www.gcores.com"
// entryURL = "https://www.showmebug.com"
// entryURL = "https://eggman.tv"
)

// var continueDomains = []string{"gcores.com"}
// var continueDomains = []string{"showmebug.com"}
// var continueDomains = []string{"eggman.tv"}

var skipURLStrings = []string{
	".mp3",
	".wav",
	".aac",
	".wma",
	".flac",
	".mp4",
	".avi",
	".mov",
	".wmv",
	".flv",
	".jpg",
	".jpeg",
	".png",
	".gif",
	".bmp",
	".tiff",
	".pdf",
	".doc",
	".docx",
	".xls",
	".xlsx",
	".ppt",
	".pptx",
	".zip",
	".exe",
	".iso",
	".css",
}
var skipMIMETypes = []string{
	"application/octet-stream",
	"application/pdf",
	"application/zip",
	"application/vnd.ms-excel",
	"application/vnd.ms-powerpoint",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/vnd.android.package-archive",
	"image/png",
	"image/jpeg",
	"image/gif",
	"audio/mpeg",
	"video/mp4",
	"video/x-msvideo",
	"video/quicktime",
}
