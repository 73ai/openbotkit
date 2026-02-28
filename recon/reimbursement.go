package recon

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/priyanshujain/reimbursement/store"
)

// PackageFromDB reads reconciliation results from the database and creates a complete
// reimbursement package (reconciled items only):
// - output_dir/reconciled.csv
// - output_dir/attachments/ (invoice/receipt PDFs for reconciled items)
// - output_dir/statements/ (unlocked PDF statements for reconciled items)
// - output_dir.zip (everything zipped, ready to submit)
// Additionally writes unreconciled.csv alongside the folder (not in the zip) for reference.
func PackageFromDB(reconDB *store.ReconStore, pdfPasswords map[string]string) error {
	// Create timestamped output directory
	timestamp := time.Now().Format("2006-01-02")
	outputDir := filepath.Join("results", fmt.Sprintf("reimbursement_%s", timestamp))

	// Clean up any previous run
	os.RemoveAll(outputDir)
	os.Remove(outputDir + ".zip")

	attachDir := filepath.Join(outputDir, "attachments")
	stmtDir := filepath.Join(outputDir, "statements")

	if err := os.MkdirAll(attachDir, 0755); err != nil {
		return fmt.Errorf("create attachments dir: %w", err)
	}
	if err := os.MkdirAll(stmtDir, 0755); err != nil {
		return fmt.Errorf("create statements dir: %w", err)
	}

	// Fetch rows from DB
	reconciledRows, err := reconDB.GetByStatus(StatusReconciled)
	if err != nil {
		return fmt.Errorf("get reconciled rows: %w", err)
	}
	unreconciledRows, err := reconDB.GetByStatus(StatusUnreconciled)
	if err != nil {
		return fmt.Errorf("get unreconciled rows: %w", err)
	}

	// Track unique statement files to copy (only from reconciled rows)
	stmtFiles := make(map[string]bool)

	// Process reconciled rows — copy attachments & track statements
	reconciledCSV := toCSVRows(reconciledRows)
	for i, row := range reconciledCSV {
		reconciledCSV[i] = processRow(row, attachDir, stmtFiles)
	}

	// Copy and unlock statement PDFs (only those referenced by reconciled rows)
	for stmtPath := range stmtFiles {
		destName := sanitizeFileName(stmtPath)
		destPath := filepath.Join(stmtDir, destName)

		if unlocked := unlockPDF(stmtPath, destPath, pdfPasswords); !unlocked {
			if err := copyFile(stmtPath, destPath); err != nil {
				fmt.Printf("  Warning: could not copy statement %s: %v\n", stmtPath, err)
				continue
			}
		}

		localRef := filepath.Join("statements", destName)
		updateDestRef(reconciledCSV, stmtPath, localRef)
	}

	// Write reconciled.csv inside the package
	header := csvHeader()
	reconciledPath := filepath.Join(outputDir, "reconciled.csv")
	if err := writeCSVRows(reconciledPath, append([][]string{header}, reconciledCSV...)); err != nil {
		return fmt.Errorf("write reconciled csv: %w", err)
	}

	// Create zip (only reconciled items)
	zipPath := outputDir + ".zip"
	if err := zipDirectory(outputDir, zipPath); err != nil {
		return fmt.Errorf("create zip: %w", err)
	}

	// Write unreconciled.csv outside the package (for your reference, not submitted)
	unreconciledCSV := toCSVRows(unreconciledRows)
	unreconciledPath := filepath.Join("results", "unreconciled.csv")
	if err := writeCSVRows(unreconciledPath, append([][]string{header}, unreconciledCSV...)); err != nil {
		return fmt.Errorf("write unreconciled csv: %w", err)
	}

	fmt.Printf("Reimbursement package created:\n")
	fmt.Printf("  Folder: %s/\n", outputDir)
	fmt.Printf("  Zip:    %s (submit this)\n", zipPath)
	fmt.Printf("  Reconciled:   %d rows\n", len(reconciledCSV))
	fmt.Printf("  Attachments:  %d files\n", countFiles(attachDir))
	fmt.Printf("  Statements:   %d files\n", countFiles(stmtDir))
	if len(unreconciledCSV) > 0 {
		fmt.Printf("\n  Unreconciled: %s (%d rows, for your reference)\n", unreconciledPath, len(unreconciledCSV))
	}

	return nil
}

func csvHeader() []string {
	return []string{
		"Service", "Date", "Amount (Source)", "Amount (Dest)", "Status",
		"Description", "Account", "Source", "Destination",
		"Invoice/Proof", "Dest Statement", "Notes", "Amount (INR)",
	}
}

// Column indices in the CSV row (must match csvHeader order)
const (
	colAttachment = 9
	colDestRef    = 10
)

// toCSVRows converts store.ReconRow slices to string slices for CSV writing.
func toCSVRows(rows []store.ReconRow) [][]string {
	var result [][]string
	for _, r := range rows {
		row := []string{
			r.Service,
			r.Date.Format("2006-01-02"),
			r.SourceAmount,
			r.DestAmount,
			r.Status,
			r.Description,
			r.Account,
			r.Source,
			r.Destination,
			r.AttachmentPath,
			r.DestRef,
			r.Notes,
			r.AmountINR,
		}
		result = append(result, row)
	}
	return result
}

// processRow copies attachments to the package and tracks statement files.
func processRow(row []string, attachDir string, stmtFiles map[string]bool) []string {
	// Copy attachment if exists
	if colAttachment < len(row) {
		attachPath := row[colAttachment]
		if attachPath != "" {
			destName := sanitizeFileName(attachPath)
			destPath := filepath.Join(attachDir, destName)
			if err := copyFile(attachPath, destPath); err != nil {
				fmt.Printf("  Warning: could not copy attachment %s: %v\n", attachPath, err)
			} else {
				row[colAttachment] = filepath.Join("attachments", destName)
			}
		}
	}

	// Track statement files
	if colDestRef < len(row) {
		stmtPath := row[colDestRef]
		if stmtPath != "" && strings.HasSuffix(strings.ToLower(stmtPath), ".pdf") {
			stmtFiles[stmtPath] = true
		}
	}

	return row
}

// updateDestRef replaces original statement paths with local references in CSV rows.
func updateDestRef(rows [][]string, originalPath, localRef string) {
	for i := range rows {
		if colDestRef < len(rows[i]) && rows[i][colDestRef] == originalPath {
			rows[i][colDestRef] = localRef
		}
	}
}

func writeCSVRows(path string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// sanitizeFileName creates a flat filename from a path like "attachments/myhq.in/123.pdf" -> "myhq.in_123.pdf"
func sanitizeFileName(path string) string {
	name := strings.ReplaceAll(path, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimLeft(name, "_")
	return name
}

// unlockPDF attempts to create an unlocked copy of a PDF using qpdf.
func unlockPDF(srcPath, destPath string, passwords map[string]string) bool {
	if _, err := exec.LookPath("qpdf"); err != nil {
		return false
	}

	var password string
	for key, pw := range passwords {
		if strings.Contains(strings.ToLower(srcPath), strings.ToLower(key)) {
			password = pw
			break
		}
	}

	if password != "" {
		cmd := exec.Command("qpdf", "--password="+password, "--decrypt", srcPath, destPath)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	cmd := exec.Command("qpdf", "--decrypt", srcPath, destPath)
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}

func zipDirectory(srcDir, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
