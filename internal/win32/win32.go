// Package win32 wraps the small set of Windows APIs this launcher needs.
//
// Everything here is implemented with the standard library only (syscall +
// unsafe), so `go build` works offline with no third-party modules and no cgo.
package win32

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procCreateToolhelp32Snapshot   = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW            = kernel32.NewProc("Process32FirstW")
	procProcess32NextW             = kernel32.NewProc("Process32NextW")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procTerminateProcess           = kernel32.NewProc("TerminateProcess")
	procWaitForSingleObject        = kernel32.NewProc("WaitForSingleObject")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procGetModuleFileNameW         = kernel32.NewProc("GetModuleFileNameW")
	procGetPackagesByPackageFamily = kernel32.NewProc("GetPackagesByPackageFamily")
	procGetPackagePathByFullName   = kernel32.NewProc("GetPackagePathByFullName")
	procPackageFamilyNameFromFull  = kernel32.NewProc("PackageFamilyNameFromFullName")

	procMessageBoxW = user32.NewProc("MessageBoxW")

	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCLSIDFromString  = ole32.NewProc("CLSIDFromString")
)

// Win32 constants.
const (
	TH32CS_SNAPPROCESS = 0x00000002

	PROCESS_TERMINATE                 = 0x0001
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	SYNCHRONIZE                       = 0x00100000

	CLSCTX_ALL                = 0x17
	COINIT_APARTMENTTHREADED  = 0x2
	ERROR_SUCCESS             = 0
	ERROR_INSUFFICIENT_BUFFER = 122

	MB_OK              = 0x00000000
	MB_YESNO           = 0x00000004
	MB_ICONERROR       = 0x00000010
	MB_ICONINFORMATION = 0x00000040
	MB_SETFOREGROUND   = 0x00010000
	IDYES              = 6
)

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// Process is a lightweight snapshot of a running process.
type Process struct {
	PID  uint32
	Name string // lower-case executable file name, e.g. "chatgpt.exe"
	Path string // full path, empty when access was denied
}

// Processes enumerates all running processes.
func Processes() ([]Process, error) {
	snapshot, _, callErr := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", callErr)
	}
	defer CloseHandle(syscall.Handle(snapshot))

	var entry processEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	ok, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return nil, nil
	}

	result := make([]Process, 0, 256)
	for ok != 0 {
		p := Process{
			PID:  entry.ProcessID,
			Name: strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:])),
		}
		if path, err := ImagePath(entry.ProcessID); err == nil {
			p.Path = path
		}
		result = append(result, p)
		ok, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	return result, nil
}

// ImagePath returns the full executable path for pid.
func ImagePath(pid uint32) (string, error) {
	handle, err := OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, pid)
	if err != nil {
		return "", err
	}
	defer CloseHandle(handle)

	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	ok, _, callErr := procQueryFullProcessImageNameW.Call(
		uintptr(handle), 0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return "", callErr
	}
	return syscall.UTF16ToString(buffer[:size]), nil
}

// OpenProcess opens a handle with the given access rights.
func OpenProcess(access uint32, pid uint32) (syscall.Handle, error) {
	handle, _, callErr := procOpenProcess.Call(uintptr(access), 0, uintptr(pid))
	if handle == 0 {
		return 0, callErr
	}
	return syscall.Handle(handle), nil
}

// CloseHandle closes a kernel handle.
func CloseHandle(handle syscall.Handle) {
	_, _, _ = procCloseHandle.Call(uintptr(handle))
}

// Terminate kills pid and waits up to waitMs for it to exit.
func Terminate(pid uint32, waitMs uint32) error {
	handle, err := OpenProcess(PROCESS_TERMINATE|SYNCHRONIZE, pid)
	if err != nil {
		return err
	}
	defer CloseHandle(handle)

	ok, _, callErr := procTerminateProcess.Call(uintptr(handle), 1)
	if ok == 0 {
		return callErr
	}
	_, _, _ = procWaitForSingleObject.Call(uintptr(handle), uintptr(waitMs))
	return nil
}

// ModuleFileName returns the full path of the running executable.
func ModuleFileName() (string, error) {
	buffer := make([]uint16, 32768)
	n, _, callErr := procGetModuleFileNameW.Call(
		0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if n == 0 {
		return "", callErr
	}
	return syscall.UTF16ToString(buffer[:n]), nil
}

// MessageBox shows a modal dialog. Returns the pressed button id.
func MessageBox(text, caption string, flags uint32) int32 {
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return 0
	}
	captionPtr, err := syscall.UTF16PtrFromString(caption)
	if err != nil {
		return 0
	}
	result, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(flags))
	runtime.KeepAlive(textPtr)
	runtime.KeepAlive(captionPtr)
	return int32(result)
}

// GUID is the COM globally unique identifier layout.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// GUIDFromString parses the "{...}" textual form.
func GUIDFromString(text string) (GUID, error) {
	var guid GUID
	ptr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return guid, err
	}
	hr, _, _ := procCLSIDFromString.Call(
		uintptr(unsafe.Pointer(ptr)), uintptr(unsafe.Pointer(&guid)))
	runtime.KeepAlive(ptr)
	if int32(hr) < 0 {
		return guid, fmt.Errorf("CLSIDFromString(%s) HRESULT 0x%08X", text, uint32(hr))
	}
	return guid, nil
}

// CoInitializeEx initialises COM for the calling thread.
//
// COM apartments are per OS thread, so callers that use the returned object
// must keep the goroutine on that thread with runtime.LockOSThread.
func CoInitializeEx() error {
	hr, _, _ := procCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	switch code := uint32(hr); code {
	case 0x00000000, 0x00000001: // S_OK, S_FALSE
		return nil
	case 0x80010106: // RPC_E_CHANGED_MODE: already initialised, different model
		return nil
	default:
		return fmt.Errorf("CoInitializeEx HRESULT 0x%08X", code)
	}
}

// CoUninitialize tears down the COM apartment.
func CoUninitialize() {
	_, _, _ = procCoUninitialize.Call()
}

// comObject mirrors the leading vtable pointer of a COM interface.
type comObject struct {
	vtable *[comSlotCount]uintptr
}

// comSlotCount covers every vtable slot this package calls into.
const comSlotCount = 8

// CoCreateInstance creates a COM object and returns its interface pointer.
func CoCreateInstance(clsid, iid GUID) (unsafe.Pointer, error) {
	var object unsafe.Pointer
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsid)),
		0,
		CLSCTX_ALL,
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(&object)))
	runtime.KeepAlive(clsid)
	runtime.KeepAlive(iid)
	if int32(hr) < 0 {
		return nil, fmt.Errorf("CoCreateInstance HRESULT 0x%08X", uint32(hr))
	}
	return object, nil
}

// callVtbl invokes vtable slot index on a COM object.
func callVtbl(object unsafe.Pointer, index int, args ...uintptr) uintptr {
	if object == nil || index < 0 || index >= comSlotCount {
		return 0x80004003 // E_POINTER
	}
	fn := (*comObject)(object).vtable[index]

	callArgs := make([]uintptr, 0, len(args)+1)
	callArgs = append(callArgs, uintptr(object))
	callArgs = append(callArgs, args...)
	result, _, _ := syscall.SyscallN(fn, callArgs...)
	return result
}

// Release calls IUnknown::Release.
func Release(object unsafe.Pointer) {
	if object != nil {
		callVtbl(object, 2)
	}
}

// ActivateApplication starts a packaged app with arguments.
// It mirrors IApplicationActivationManager::ActivateApplication (vtable slot 3).
func ActivateApplication(object unsafe.Pointer, appUserModelID, arguments string) (uint32, error) {
	aumidPtr, err := syscall.UTF16PtrFromString(appUserModelID)
	if err != nil {
		return 0, err
	}
	argsPtr, err := syscall.UTF16PtrFromString(arguments)
	if err != nil {
		return 0, err
	}

	var pid uint32
	hr := callVtbl(object, 3,
		uintptr(unsafe.Pointer(aumidPtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		0,
		uintptr(unsafe.Pointer(&pid)))

	runtime.KeepAlive(aumidPtr)
	runtime.KeepAlive(argsPtr)

	if int32(hr) < 0 {
		return 0, fmt.Errorf("ActivateApplication HRESULT 0x%08X", uint32(hr))
	}
	return pid, nil
}

// PackageFullNames returns every installed version of a package family.
func PackageFullNames(family string) ([]string, error) {
	familyPtr, err := syscall.UTF16PtrFromString(family)
	if err != nil {
		return nil, err
	}

	var count, bufferLength uint32
	rc, _, _ := procGetPackagesByPackageFamily.Call(
		uintptr(unsafe.Pointer(familyPtr)),
		uintptr(unsafe.Pointer(&count)),
		0,
		uintptr(unsafe.Pointer(&bufferLength)),
		0)
	runtime.KeepAlive(familyPtr)

	if rc == ERROR_SUCCESS && count == 0 {
		return nil, nil
	}
	if rc != ERROR_INSUFFICIENT_BUFFER || count == 0 {
		return nil, fmt.Errorf("GetPackagesByPackageFamily probe failed rc=%d", rc)
	}

	buffer := make([]uint16, bufferLength)
	pointers := make([]unsafe.Pointer, count)
	rc, _, _ = procGetPackagesByPackageFamily.Call(
		uintptr(unsafe.Pointer(familyPtr)),
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&pointers[0])),
		uintptr(unsafe.Pointer(&bufferLength)),
		uintptr(unsafe.Pointer(&buffer[0])))
	runtime.KeepAlive(familyPtr)
	runtime.KeepAlive(pointers)
	runtime.KeepAlive(buffer)

	if rc != ERROR_SUCCESS {
		return nil, fmt.Errorf("GetPackagesByPackageFamily failed rc=%d", rc)
	}

	names := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		if pointers[i] == nil {
			continue
		}
		names = append(names, utf16At(pointers[i]))
	}
	return names, nil
}

// PackagePath returns the install root of a package full name.
func PackagePath(fullName string) (string, error) {
	fullPtr, err := syscall.UTF16PtrFromString(fullName)
	if err != nil {
		return "", err
	}

	var length uint32
	rc, _, _ := procGetPackagePathByFullName.Call(
		uintptr(unsafe.Pointer(fullPtr)), uintptr(unsafe.Pointer(&length)), 0)
	runtime.KeepAlive(fullPtr)

	if rc != ERROR_INSUFFICIENT_BUFFER || length == 0 {
		return "", fmt.Errorf("GetPackagePathByFullName probe failed rc=%d", rc)
	}

	buffer := make([]uint16, length)
	rc, _, _ = procGetPackagePathByFullName.Call(
		uintptr(unsafe.Pointer(fullPtr)),
		uintptr(unsafe.Pointer(&length)),
		uintptr(unsafe.Pointer(&buffer[0])))
	runtime.KeepAlive(fullPtr)
	runtime.KeepAlive(buffer)

	if rc != ERROR_SUCCESS {
		return "", fmt.Errorf("GetPackagePathByFullName failed rc=%d", rc)
	}
	return strings.TrimRight(syscall.UTF16ToString(buffer[:length]), "\x00"), nil
}

// PackageFamilyFromFullName converts a package full name to its family name.
func PackageFamilyFromFullName(fullName string) (string, error) {
	fullPtr, err := syscall.UTF16PtrFromString(fullName)
	if err != nil {
		return "", err
	}

	var length uint32
	rc, _, _ := procPackageFamilyNameFromFull.Call(
		uintptr(unsafe.Pointer(fullPtr)), uintptr(unsafe.Pointer(&length)), 0)
	runtime.KeepAlive(fullPtr)

	if rc != ERROR_INSUFFICIENT_BUFFER || length == 0 {
		return "", fmt.Errorf("PackageFamilyNameFromFullName probe failed rc=%d", rc)
	}

	buffer := make([]uint16, length)
	rc, _, _ = procPackageFamilyNameFromFull.Call(
		uintptr(unsafe.Pointer(fullPtr)),
		uintptr(unsafe.Pointer(&length)),
		uintptr(unsafe.Pointer(&buffer[0])))
	runtime.KeepAlive(fullPtr)
	runtime.KeepAlive(buffer)

	if rc != ERROR_SUCCESS {
		return "", fmt.Errorf("PackageFamilyNameFromFullName failed rc=%d", rc)
	}
	return strings.TrimRight(syscall.UTF16ToString(buffer[:length]), "\x00"), nil
}

// utf16At reads a NUL-terminated UTF-16 string at the given address.
func utf16At(base unsafe.Pointer) string {
	units := make([]uint16, 0, 64)
	for offset := uintptr(0); ; offset += 2 {
		unit := *(*uint16)(unsafe.Add(base, offset))
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units))
}
