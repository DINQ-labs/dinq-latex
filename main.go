package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8092"
	}

	http.HandleFunc("/compile", handleCompile)
	http.HandleFunc("/convert", handleConvert)
	http.HandleFunc("/health", handleHealth)

	log.Printf("[dinq-latex] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","engine":"tectonic+libreoffice"}`))
}

func handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "empty LaTeX source", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "latex-compile-*")
	if err != nil {
		http.Error(w, "failed to create temp dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	texFile := filepath.Join(tmpDir, "input.tex")
	if err := os.WriteFile(texFile, body, 0644); err != nil {
		http.Error(w, "failed to write tex file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	start := time.Now()

	cmd := exec.Command("tectonic", "-X", "compile", "--outdir", tmpDir, texFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[compile] tectonic failed in %v: %s", time.Since(start), string(output))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		// Escape output for JSON
		fmt.Fprintf(w, `{"error":"compilation failed","log":%q}`, string(output))
		return
	}

	log.Printf("[compile] success in %v", time.Since(start))

	pdfFile := filepath.Join(tmpDir, "input.pdf")
	pdfBytes, err := os.ReadFile(pdfFile)
	if err != nil {
		http.Error(w, "failed to read PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.Write(pdfBytes)
}

// handleConvert converts Word (.doc/.docx) files to PDF via LibreOffice.
// Accepts the Word file as raw POST body.
// The caller should set the filename via query param ?filename=xxx.docx
func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "empty file", http.StatusBadRequest)
		return
	}

	// Determine input filename (need correct extension for LibreOffice)
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = "input.docx"
	}

	tmpDir, err := os.MkdirTemp("", "word-convert-*")
	if err != nil {
		http.Error(w, "failed to create temp dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	inputFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(inputFile, body, 0644); err != nil {
		http.Error(w, "failed to write input file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	start := time.Now()

	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", "--outdir", tmpDir, inputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[convert] libreoffice failed in %v: %s", time.Since(start), string(output))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, `{"error":"conversion failed","log":%q}`, string(output))
		return
	}

	log.Printf("[convert] success in %v", time.Since(start))

	// LibreOffice outputs PDF with same basename
	baseName := strings.TrimSuffix(filename, ext)
	pdfFile := filepath.Join(tmpDir, baseName+".pdf")
	pdfBytes, err := os.ReadFile(pdfFile)
	if err != nil {
		http.Error(w, "failed to read converted PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.Write(pdfBytes)
}
