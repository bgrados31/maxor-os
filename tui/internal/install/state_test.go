package install

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnswersNeedAHashedPassword(t *testing.T) {
	s := Default()
	s.Disk, s.Username, s.Password = "/dev/vda", "ana", "plain-password"
	if _, err := s.Answers(); err == nil {
		t.Fatal("without a hash the answers must not be produced")
	}
}

func TestAnswersFollowTheContractAndNeverCarrySecrets(t *testing.T) {
	s := Default()
	s.Disk, s.Username, s.Fullname, s.Hostname = "/dev/nvme0n1", "ana", "Ana Pérez", "portatil"
	s.Password, s.Passphrase = "plain-password", "disk-passphrase-123"
	s.PasswordHash = "$6$salt$hashed"
	s.Encrypt, s.SwapKind, s.SwapGiB = true, "file", 8
	s.Profiles = []string{"dev", "gaming"}
	s.Confirmed = "ERASE"
	b, err := s.Answers()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "plain-password") || strings.Contains(string(b), "disk-passphrase") {
		t.Fatalf("a secret reached the answers:\n%s", b)
	}
	var doc struct {
		Schema int `json:"schema"`
		Disk   struct {
			Device    string `json:"device"`
			Strategy  string `json:"strategy"`
			Confirmed string `json:"confirmed"`
			Encrypt   struct {
				Enabled bool `json:"enabled"`
			} `json:"encrypt"`
			Swap struct {
				Kind string `json:"kind"`
				GiB  int    `json:"gib"`
			} `json:"swap"`
			Region *struct{} `json:"region"`
		} `json:"disk"`
		User struct {
			Name         string `json:"name"`
			PasswordHash string `json:"password_hash"`
		} `json:"user"`
		Look struct {
			Profiles []string `json:"profiles"`
		} `json:"look"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Schema != 1 || doc.Disk.Device != "/dev/nvme0n1" || doc.Disk.Strategy != "whole" || doc.Disk.Confirmed != "ERASE" ||
		!doc.Disk.Encrypt.Enabled || doc.Disk.Swap.Kind != "file" || doc.Disk.Swap.GiB != 8 ||
		doc.User.Name != "ana" || doc.User.PasswordHash != "$6$salt$hashed" || len(doc.Look.Profiles) != 2 {
		t.Fatalf("unexpected answers: %s", b)
	}
	if doc.Disk.Region != nil {
		t.Fatal("a whole-disk install carries no region")
	}
}

func TestAlongsideCarriesTheRegionAndThePlanHash(t *testing.T) {
	s := Default()
	s.Disk, s.Username, s.PasswordHash = "/dev/vda", "ana", "$6$a$b"
	s.Strategy, s.Region, s.PlanHash, s.Confirmed = "alongside", Region{Start: 4000000, End: 200000000}, strings.Repeat("a", 64), "INSTALL"
	b, err := s.Answers()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"region"`, `4000000`, `200000000`, `"plan_hash"`, `"INSTALL"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s in\n%s", want, b)
		}
	}
}

func TestNoProfilesIsAnEmptyListNotNull(t *testing.T) {
	s := Default()
	s.Disk, s.Username, s.PasswordHash = "/dev/vda", "ana", "$6$a$b"
	b, _ := s.Answers()
	if !strings.Contains(string(b), `"profiles": []`) {
		t.Fatalf("the engine needs a list:\n%s", b)
	}
}

func TestHostnameRules(t *testing.T) {
	good := []string{"maxor", "my-laptop", "a", "pc01", strings.Repeat("a", 63)}
	bad := []string{"", "localhost", "Bad", "has_underscore", "-start", "end-", "with space", strings.Repeat("a", 64), "ñandú"}
	for _, h := range good {
		if err := ValidateHostname(h); err != nil {
			t.Errorf("%q should be valid: %v", h, err)
		}
	}
	for _, h := range bad {
		if ValidateHostname(h) == nil {
			t.Errorf("%q should be invalid", h)
		}
	}
}

func TestUsernameRules(t *testing.T) {
	for _, u := range []string{"ana", "ana_b", "a1", "_svc", "x-y"} {
		if err := ValidateUsername(u); err != nil {
			t.Errorf("%q should be valid: %v", u, err)
		}
	}
	for _, u := range []string{"", "Ana", "1ana", "root", "maxor", "nobody", "has space", "a:b", strings.Repeat("a", 33)} {
		if ValidateUsername(u) == nil {
			t.Errorf("%q should be invalid", u)
		}
	}
}

func TestFullnameCannotBreakPasswd(t *testing.T) {
	if ValidateFullname("Ana: root") == nil || ValidateFullname("a\nb") == nil || ValidateFullname(strings.Repeat("a", 65)) == nil {
		t.Fatal("a colon, a line break or an overlong name must be rejected")
	}
	if err := ValidateFullname("Ana \"la\" Pérez"); err != nil {
		t.Fatalf("quotes and accents are fine: %v", err)
	}
}

func TestPasswordNeedsLengthAndConfirmation(t *testing.T) {
	if ValidatePassword("short", "short") == nil {
		t.Fatal("too short")
	}
	if ValidatePassword("longenough1", "different11") == nil {
		t.Fatal("must match its confirmation")
	}
	if err := ValidatePassword("longenough1", "longenough1"); err != nil {
		t.Fatal(err)
	}
}

func TestPassphraseIsStricterThanAPassword(t *testing.T) {
	if ValidatePassphrase("password", "password") == nil {
		t.Fatal("an easy passphrase must be refused")
	}
	if ValidatePassphrase("aaaaaaaaaaaa", "aaaaaaaaaaaa") == nil {
		t.Fatal("a repeated character is not a passphrase")
	}
	if err := ValidatePassphrase("correct horse battery staple", "correct horse battery staple"); err != nil {
		t.Fatalf("a long passphrase is fine: %v", err)
	}
}

func TestStrengthOrdersPasswords(t *testing.T) {
	weak, mid, strong := Strength("abc12"), Strength("Abcdefg1234"), Strength("correct horse battery staple 42!")
	if !(weak < mid && mid <= strong) || strong < 3 {
		t.Fatalf("strength should grow: %d %d %d", weak, mid, strong)
	}
	if Strength("aaaaaaaaaaaaaaaaaaaaaa") != 0 {
		t.Fatal("a repeated character is never strong")
	}
}

func TestRegionMustBeBigEnough(t *testing.T) {
	big := Region{Start: 2048, End: 2048 + 50*2097152}
	if err := ValidateRegion(big, 40); err != nil {
		t.Fatal(err)
	}
	if ValidateRegion(Region{Start: 2048, End: 2048 + 10*2097152}, 40) == nil {
		t.Fatal("10 GiB is not enough")
	}
	if ValidateRegion(Region{Start: 5, End: 5}, 1) == nil {
		t.Fatal("an empty region")
	}
}
