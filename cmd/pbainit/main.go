package main

import (
	"bufio"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"

	// "encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	tcg "github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/drive"
	"github.com/bluecmd/go-tcg-storage/pkg/locking"
	"github.com/u-root/u-root/pkg/libinit"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/u-root/pkg/ulog"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var (
	GitSource = "https://github.com/felixstorm/elx-pba"
)

func main() {
	fmt.Printf("\n")
	l, _ := base64.StdEncoding.DecodeString(logo)
	fmt.Println(string(l))
	if GitInfo == "" {
		GitInfo = "(no hash)"
	}
	fmt.Printf("Welcome to Elastx PBA!\nSource: %s, %s\n\n", GitSource, GitInfo)
	log.SetPrefix("elx-pba: ")

	if _, err := mount.Mount("proc", "/proc", "proc", "", 0); err != nil {
		log.Fatalf("Mount(proc): %v", err)
	}
	if _, err := mount.Mount("sysfs", "/sys", "sysfs", "", 0); err != nil {
		log.Fatalf("Mount(sysfs): %v", err)
	}
	if _, err := mount.Mount("efivarfs", "/sys/firmware/efi/efivars", "efivarfs", "", 0); err != nil {
		log.Fatalf("Mount(efivars): %v", err)
	}

	log.Printf("Starting system...")

	if err := ulog.KernelLog.SetConsoleLogLevel(ulog.KLogNotice); err != nil {
		log.Printf("Could not set log level: %v", err)
	}

	libinit.SetEnv()
	libinit.CreateRootfs()
	libinit.NetInit()

	defer func() {
		log.Printf("Starting emergency shell...")
		for {
			Execute("/bbin/elvish")
		}
	}()

	dmi, err := readDMI()
	if err != nil {
		log.Printf("Failed to read SMBIOS/DMI data: %v", err)
		return
	}

	log.Printf("System UUID:            %s", dmi.SystemUUID)
	log.Printf("System serial:          %s", dmi.SystemSerialNumber)
	log.Printf("Baseboard manufacturer: %s", dmi.BaseboardManufacturer)
	log.Printf("Baseboard product:      %s", dmi.BaseboardProduct)
	log.Printf("Baseboard serial:       %s", dmi.BaseboardSerialNumber)
	log.Printf("Chassis serial:         %s", dmi.ChassisSerialNumber)

	sysblk, err := ioutil.ReadDir("/sys/class/block/")
	if err != nil {
		log.Printf("Failed to enumerate block devices: %v", err)
		return
	}

	startEmergencyShell := true
	password := ""
	for _, fi := range sysblk {
		devname := fi.Name()
		if _, err := os.Stat(filepath.Join("sys/class/block", devname, "device")); os.IsNotExist(err) {
			continue
		}
		devpath := filepath.Join("/dev", devname)
		if _, err := os.Stat(devpath); os.IsNotExist(err) {
			majmin, err := ioutil.ReadFile(filepath.Join("/sys/class/block", devname, "dev"))
			if err != nil {
				log.Printf("Failed to read major:minor for %s: %v", devname, err)
				continue
			}
			parts := strings.Split(strings.TrimSpace(string(majmin)), ":")
			major, _ := strconv.ParseInt(parts[0], 10, 8)
			minor, _ := strconv.ParseInt(parts[1], 10, 8)
			if err := unix.Mknod(filepath.Join("/dev", devname), unix.S_IFBLK|0600, int(major<<16|minor)); err != nil {
				log.Printf("Mknod(%s) failed: %v", devname, err)
				continue
			}
		}

		d, err := drive.Open(devpath)
		if err != nil {
			log.Printf("drive.Open(%s): %v", devpath, err)
			continue
		}
		defer d.Close()
		identity, err := d.Identify()
		if err != nil {
			log.Printf("drive.Identify(%s): %v", devpath, err)
		}
		dsn, err := d.SerialNumber()
		if err != nil {
			log.Printf("drive.SerialNumber(%s): %v", devpath, err)
		}
		d0, err := tcg.Discovery0(d)
		if err != nil {
			if err != tcg.ErrNotSupported {
				log.Printf("tcg.Discovery0(%s): %v", devpath, err)
			}
			continue
		}
		if d0.Locking != nil && d0.Locking.Locked {
			log.Printf("Drive %s is locked", identity)
			if d0.Locking.MBREnabled && !d0.Locking.MBRDone {
				log.Printf("Drive %s has active shadow MBR", identity)
			}
			unlocked := false
			for !unlocked {
				// reuse-existing password for multiple drives
				if password == "" {
					password = getDrivePassword()
				}
				if err := unlock(d, password, dsn); err != nil {
					log.Printf("Failed to unlock %s: %v", identity, err)
					// clear password to be queried again
					password = ""
				} else {
					unlocked = true
				}
			}
			bd, err := block.Device(devpath)
			if err != nil {
				log.Printf("block.Device(%s): %v", devpath, err)
				continue
			}
			if err := bd.ReadPartitionTable(); err != nil {
				log.Printf("block.ReadPartitionTable(%s): %v", devpath, err)
				continue
			}
			log.Printf("Drive %s has been unlocked", devpath)
			startEmergencyShell = false
		} else {
			log.Printf("Considered drive %s, but drive is not locked", identity)
		}
	}

	if startEmergencyShell {
		log.Printf("No drives changed state to unlocked, starting shell for troubleshooting")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	abort := make(chan bool)
	go func() {
		fmt.Println("")
		log.Printf("Starting 'boot' in 5 seconds, press Enter to start shell instead")
		select {
		case <-abort:
			return
		case <-time.After(5 * time.Second):
			// pass
		}
		// Work-around for systems which are known to fail during boot/kexec - these
		// systems keep the drives in an unlocked state during software triggered reboots,
		// which means that the "real" kernel and rootfs should be booted afterwards
		if dmi.BaseboardManufacturer == "Supermicro" && strings.HasPrefix(dmi.BaseboardProduct, "X12") {
			log.Printf("Work-around: Rebooting system instead of utilizing 'boot'")
			Execute("/bbin/shutdown", "reboot")
		} else {
			Execute("/bbin/boot")
		}
	}()

	reader.ReadString('\n')
	abort <- true
}

func getDrivePassword() string {
	fmt.Printf("Enter OPAL drive password (prefix with 'chubbyant ' to use 500000*SH512): ")
	bytePassword, err := term.ReadPassword(0)
	fmt.Println()
	if err != nil {
		log.Printf("terminal.ReadPassword(0): %v", err)
		return ""
	}
	return string(bytePassword)
}

func unlock(d tcg.DriveIntf, pass string, driveserial []byte) error {
	// Same format as used by sedutil for compatibility
	salt := fmt.Sprintf("%-20s", string(driveserial))
	var pin []byte
	// y and z are switched on US English vs. German keyboard layout
	chubbyAntRegexp := regexp.MustCompile(`^chubb(y|z)ant `)
	if chubbyAntRegexp.MatchString(pass) {
		pass = chubbyAntRegexp.ReplaceAllLiteralString(pass, "")
		// github.com/ChubbyAnt/sedutil
		pin = pbkdf2.Key([]byte(pass), []byte(salt[:20]), 500000, 32, sha512.New)
	} else {
		// github.com/Drive-Trust-Alliance/sedutil
		pin = pbkdf2.Key([]byte(pass), []byte(salt[:20]), 75000, 32, sha1.New)
	}
	log.Printf("Password length: %v", len(pass))
	// log.Printf("Password: %s, hash: %s", hex.EncodeToString([]byte(pass)), hex.EncodeToString(pin))

	cs, lmeta, err := locking.Initialize(d)
	if err != nil {
		return fmt.Errorf("locking.Initialize: %v", err)
	}
	defer cs.Close()
	l, err := locking.NewSession(cs, lmeta, locking.DefaultAuthority(pin))
	if err != nil {
		return fmt.Errorf("locking.NewSession: %v", err)
	}
	defer l.Close()

	for i, r := range l.Ranges {
		if err := r.UnlockRead(); err != nil {
			log.Printf("Read unlock range %d failed: %v", i, err)
		}
		if err := r.UnlockWrite(); err != nil {
			log.Printf("Write unlock range %d failed: %v", i, err)
		}
	}

	if l.MBREnabled && !l.MBRDone {
		if err := l.SetMBRDone(true); err != nil {
			return fmt.Errorf("SetMBRDone: %v", err)
		}
	}
	return nil
}

func Execute(name string, args ...string) {
	environ := append(os.Environ(), "USER=root")
	environ = append(environ, "HOME=/root")
	environ = append(environ, "TZ=UTC")

	cmd := exec.Command(name, args...)
	cmd.Dir = "/"
	cmd.Env = environ
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Setsid = true
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to execute: %v", err)
	}
}
