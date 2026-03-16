import Contacts
import Foundation

// MARK: - JSON Output Helpers

func jsonOutput(_ dict: [String: Any]) {
    guard let data = try? JSONSerialization.data(withJSONObject: dict, options: [.sortedKeys]),
          let str = String(data: data, encoding: .utf8) else {
        print("{\"error\":\"json encoding failed\",\"code\":\"internal\"}")
        return
    }
    print(str)
}

func errorOutput(_ message: String, code: String) {
    jsonOutput(["error": message, "code": code])
}

// MARK: - Contacts

func fetchContacts() {
    let store = CNContactStore()

    switch CNContactStore.authorizationStatus(for: .contacts) {
    case .authorized:
        break
    case .notDetermined:
        let semaphore = DispatchSemaphore(value: 0)
        var granted = false
        store.requestAccess(for: .contacts) { g, _ in
            granted = g
            semaphore.signal()
        }
        semaphore.wait()
        if !granted {
            errorOutput("Contacts access denied. Grant in System Settings > Privacy & Security > Contacts.", code: "permission_denied")
            Foundation.exit(1)
        }
    default:
        errorOutput("Contacts access denied. Grant in System Settings > Privacy & Security > Contacts.", code: "permission_denied")
        Foundation.exit(1)
    }

    let keys: [CNKeyDescriptor] = [
        CNContactGivenNameKey as CNKeyDescriptor,
        CNContactFamilyNameKey as CNKeyDescriptor,
        CNContactNicknameKey as CNKeyDescriptor,
        CNContactPhoneNumbersKey as CNKeyDescriptor,
        CNContactEmailAddressesKey as CNKeyDescriptor,
    ]

    var contacts: [[String: Any]] = []
    let request = CNContactFetchRequest(keysToFetch: keys)

    do {
        try store.enumerateContacts(with: request) { contact, _ in
            var phones: [String] = []
            for phone in contact.phoneNumbers {
                phones.append(phone.value.stringValue)
            }
            var emails: [String] = []
            for email in contact.emailAddresses {
                emails.append(email.value as String)
            }
            contacts.append([
                "firstName": contact.givenName,
                "lastName": contact.familyName,
                "nickname": contact.nickname,
                "phones": phones,
                "emails": emails,
            ])
        }
    } catch {
        errorOutput("Failed to fetch contacts: \(error.localizedDescription)", code: "fetch_failed")
        Foundation.exit(1)
    }

    jsonOutput(["contacts": contacts])
}

// MARK: - Notes

func fetchNotes() {
    let script = """
    tell application "Notes"
        set noteIDs to id of every note
        set noteNames to name of every note
        set noteDates to modification date of every note
        set noteCreated to creation date of every note
        set noteProtected to password protected of every note
        set notePlain to plaintext of every note

        set output to ""
        repeat with i from 1 to count of noteIDs
            set nID to item i of noteIDs
            set nName to item i of noteNames
            set nMod to item i of noteDates
            set nCreated to item i of noteCreated
            set nProt to item i of noteProtected
            set nText to item i of notePlain
            set output to output & nID & "%%FIELD%%" & nName & "%%FIELD%%" & nMod & "%%FIELD%%" & nCreated & "%%FIELD%%" & nProt & "%%FIELD%%" & nText & "%%RECORD%%"
        end repeat
        return output
    end tell
    """

    let folderScript = """
    tell application "Notes"
        set output to ""
        repeat with f in every folder
            set fID to id of f
            set fName to name of f
            try
                set parentClass to class of container of f
                if parentClass is folder then
                    set parentID to id of container of f
                else
                    set parentID to ""
                end if
            on error
                set parentID to ""
            end try
            try
                set acctName to name of container of f
            on error
                set acctName to ""
            end try
            set noteIDs to id of notes of f
            set noteIDStr to ""
            repeat with nID in noteIDs
                set noteIDStr to noteIDStr & nID & ","
            end repeat
            set output to output & fID & "%%FIELD%%" & fName & "%%FIELD%%" & parentID & "%%FIELD%%" & acctName & "%%FIELD%%" & noteIDStr & "%%RECORD%%"
        end repeat
        return output
    end tell
    """

    // Fetch notes
    guard let notesAS = NSAppleScript(source: script) else {
        errorOutput("Failed to create notes AppleScript", code: "internal")
        Foundation.exit(1)
    }
    var notesErr: NSDictionary?
    let notesResult = notesAS.executeAndReturnError(&notesErr)
    if let err = notesErr {
        let msg = (err[NSAppleScript.errorMessage] as? String) ?? "Notes access denied."
        errorOutput(msg, code: "permission_denied")
        Foundation.exit(1)
    }

    // Fetch folders
    guard let foldersAS = NSAppleScript(source: folderScript) else {
        errorOutput("Failed to create folders AppleScript", code: "internal")
        Foundation.exit(1)
    }
    var foldersErr: NSDictionary?
    let foldersResult = foldersAS.executeAndReturnError(&foldersErr)
    if let err = foldersErr {
        let msg = (err[NSAppleScript.errorMessage] as? String) ?? "Notes access denied."
        errorOutput(msg, code: "permission_denied")
        Foundation.exit(1)
    }

    let iso = ISO8601DateFormatter()
    iso.formatOptions = [.withInternetDateTime]

    // Parse notes
    let fieldSep = "%%FIELD%%"
    let recordSep = "%%RECORD%%"

    var notes: [[String: Any]] = []
    let notesStr = notesResult.stringValue ?? ""
    for record in notesStr.components(separatedBy: recordSep) {
        let rec = record.trimmingCharacters(in: .whitespacesAndNewlines)
        if rec.isEmpty { continue }
        let fields = rec.components(separatedBy: fieldSep)
        if fields.count < 6 { continue }

        let modDate = parseAppleScriptDate(fields[2].trimmingCharacters(in: .whitespacesAndNewlines))
        let createdDate = parseAppleScriptDate(fields[3].trimmingCharacters(in: .whitespacesAndNewlines))

        notes.append([
            "id": fields[0].trimmingCharacters(in: .whitespacesAndNewlines),
            "title": fields[1].trimmingCharacters(in: .whitespacesAndNewlines),
            "body": fields[5].trimmingCharacters(in: .whitespacesAndNewlines),
            "passwordProtected": fields[4].trimmingCharacters(in: .whitespacesAndNewlines) == "true",
            "createdAt": iso.string(from: createdDate),
            "modifiedAt": iso.string(from: modDate),
        ])
    }

    // Parse folders
    var folders: [[String: Any]] = []
    let foldersStr = foldersResult.stringValue ?? ""
    for record in foldersStr.components(separatedBy: recordSep) {
        let rec = record.trimmingCharacters(in: .whitespacesAndNewlines)
        if rec.isEmpty { continue }
        let fields = rec.components(separatedBy: fieldSep)
        if fields.count < 5 { continue }

        var noteIDs: [String] = []
        let noteIDsStr = fields[4].trimmingCharacters(in: .whitespacesAndNewlines)
        if !noteIDsStr.isEmpty {
            for nID in noteIDsStr.components(separatedBy: ",") {
                let trimmed = nID.trimmingCharacters(in: .whitespacesAndNewlines)
                if !trimmed.isEmpty {
                    noteIDs.append(trimmed)
                }
            }
        }

        folders.append([
            "id": fields[0].trimmingCharacters(in: .whitespacesAndNewlines),
            "name": fields[1].trimmingCharacters(in: .whitespacesAndNewlines),
            "parentId": fields[2].trimmingCharacters(in: .whitespacesAndNewlines),
            "account": fields[3].trimmingCharacters(in: .whitespacesAndNewlines),
            "noteIds": noteIDs,
        ])
    }

    jsonOutput(["notes": notes, "folders": folders])
}

func parseAppleScriptDate(_ s: String) -> Date {
    let formats = [
        "EEEE, MMMM d, yyyy 'at' h:mm:ss a",
        "EEEE, MMMM d, yyyy 'at' h:mm:ss\u{202f}a",
        "EEEE, d MMMM yyyy 'at' HH:mm:ss",
        "yyyy-MM-dd HH:mm:ss Z",
        "M/d/yyyy h:mm:ss a",
        "d/M/yyyy h:mm:ss a",
        "MMMM d, yyyy 'at' h:mm:ss a",
        "MMMM d, yyyy 'at' h:mm:ss\u{202f}a",
        "d MMMM yyyy 'at' HH:mm:ss",
    ]
    let df = DateFormatter()
    df.locale = Locale(identifier: "en_US_POSIX")
    for fmt in formats {
        df.dateFormat = fmt
        if let date = df.date(from: s) {
            return date
        }
    }
    return Date(timeIntervalSince1970: 0)
}

// MARK: - Check Permissions

func checkPermissions() {
    var result: [String: String] = [:]

    // Check Contacts
    let contactsStatus = CNContactStore.authorizationStatus(for: .contacts)
    switch contactsStatus {
    case .authorized:
        result["contacts"] = "authorized"
    case .denied:
        result["contacts"] = "denied"
    case .restricted:
        result["contacts"] = "restricted"
    case .notDetermined:
        result["contacts"] = "not_determined"
    @unknown default:
        result["contacts"] = "unknown"
    }

    // Check Notes — try running a trivial AppleScript
    guard let notesAS = NSAppleScript(source: "tell application \"Notes\" to count of notes") else {
        result["notes"] = "unknown"
        jsonOutput(result)
        return
    }
    var notesErr: NSDictionary?
    _ = notesAS.executeAndReturnError(&notesErr)
    result["notes"] = notesErr == nil ? "authorized" : "denied"

    jsonOutput(result)
}

// MARK: - Main

let args = CommandLine.arguments
guard args.count >= 2 else {
    errorOutput("Usage: obkmacos <contacts|notes|check>", code: "usage")
    Foundation.exit(1)
}

switch args[1] {
case "contacts":
    fetchContacts()
case "notes":
    fetchNotes()
case "check":
    checkPermissions()
default:
    errorOutput("Unknown command: \(args[1]). Use: contacts, notes, check", code: "usage")
    Foundation.exit(1)
}
