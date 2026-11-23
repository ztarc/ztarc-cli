# Desain dan rencana implementasi ZTARC CLI

Tanggal: 23 November 2026  
Status: desain untuk proyek CLI terpisah  
Proyek tujuan: `ztarc/ztarc-cli`  
Proyek integrasi: `ztarc/ztarc-console`  
Baseline analisis console: commit `cc0742c`

ZTARC CLI akan dibangun dari source Pangolin CLI yang dipatok ke commit tertentu, direbrand menggunakan pola staging dan patch seperti ZTARC Windows, lalu didistribusikan melalui ZTARC Console. CLI melayani client pengguna, machine client tanpa sesi pengguna, dan Spoke Connector. Aktivasi Machines dasar pada console menjadi tahap A1 dengan direct grants dan lifecycle manual yang tercatat.

Implementasi CLI dikerjakan di repositori terpisah. Console menangani pemilihan release, distribusi installer, instruksi pemasangan, provisioning melalui boundary middleware, pengelolaan Machines, dan integrasi protokol native yang diperlukan. Penambahan dokumen ini tidak mengaktifkan CLI atau mengubah aplikasi maupun deployment.

## 1. Keputusan produk

| Keputusan | Target yang disepakati |
| --- | --- |
| Kemampuan | Client pengguna, machine client, dan runtime Spoke secara bertahap |
| Platform awal | Linux amd64 dan arm64, serta container untuk mode yang tervalidasi |
| Parity | Seluruh command Linux upstream menjadi target bertahap |
| Kebijakan tenant | RBAC, kuota, masa aktif tenant, dan pembatasan domain tetap berlaku |
| Fitur sensitif | Diaktifkan setelah adapter dan enforcement server tersedia |
| Default Spoke saat canary | Newt tetap default; ZTARC CLI merupakan pilihan tambahan |
| Default Spoke setelah acceptance | CLI menjadi default untuk instalasi baru pada platform yang lulus |
| Connector existing | Migrasi eksplisit oleh pemilik host; Newt tetap didukung |
| Update awal | Manual melalui release dan installer ZTARC |
| Machines dasar | Tahap A1: enrollment, direct grants, penanggung jawab/review akses, block/archive/delete, dan penggantian credential melalui enrollment baru |
| Machines lanjutan | Role mesin, expiry otomatis, dan rotasi credential dengan identitas tetap ditunda |
| Platform berikutnya | Windows companion dan macOS sesudah Linux |

Pengguna CLI mencakup pengguna yang mengakses private resources, administrator tenant yang memasang Spoke, dan operator yang memelihara instalasi. CLI pelanggan tidak menerima credential operator middleware atau Integration API key milik platform.

## 2. Kondisi existing dan baseline

### 2.1 ZTARC Console

Console belum menyediakan CLI pelanggan hasil rebrand Pangolin. File `middleware/src/cli.ts` merupakan alat administrasi operator untuk middleware; fungsinya berbeda dari CLI networking pelanggan.

| Komponen | Kondisi berdasarkan source dan bukti deployment tersimpan |
| --- | --- |
| Server | Pangolin EE 1.26.0; deployment server tercatat selesai 18 November 2026 |
| Adapter middleware | `ee-1.26.0`, schema `003` |
| Connector stable | Newt 1.16.0 |
| Device Agent stable | Olm 1.9.1 |
| Windows yang ditawarkan | ZTARC Windows 0.16.0.1 |
| Distribusi connector kandidat | Newt 1.20.0 dan Olm 1.10.1; belum dipromosikan |
| Connector Linux | `/usr/local/bin/ztarc-connector`, konfigurasi `/etc/ztarc/connector.json` |
| Service connector | `ztarc-connector.service` |
| Distribusi CLI | Belum ada release resolver, pin, installer, atau pilihan CLI pada UI |
| Login perangkat | Device authorization dan persetujuan browser sudah tersedia |
| Machines | UI list/enrollment/actions dan direct grants sudah ada; `FEATURES.machines` masih `false` |

Upgrade server tidak otomatis mengubah connector yang terpasang. Daftar di atas bukan inventaris ulang semua endpoint pengguna atau pemeriksaan langsung production pada tanggal dokumen.

Rujukan console: [manifest release](../deploy/pangolin-1.26.0-release.json), [bukti deployment akhir](../deploy/evidence/pangolin-1.26.0-production-final-20261118.json), dan [kontrak distribusi existing](../deploy/distribution/README.md).

### 2.2 Baseline CLI upstream

Gunakan kandidat awal berikut agar implementasi dapat diulang:

| Identitas | Nilai |
| --- | --- |
| Repository | `https://github.com/fosrl/cli` |
| Release | `0.23.0`, diterbitkan 19 November 2026 |
| Commit | `bb03d2c45ce79213c4c9b206bbd87e9846be5a9a` |
| Newt | `1.20.1` |
| Olm | `1.10.2` |
| Deklarasi Go | `go 1.26.0` |
| Toolchain upstream | `go1.26.10` |

Versi ini merupakan kandidat pengembangan, bukan pernyataan kompatibilitas production. Perubahan baseline memerlukan pembaruan metadata, audit patch dan dependency, serta pengujian kemampuan yang terdampak.

Rujukan: [release 0.23.0](https://github.com/fosrl/cli/releases/tag/0.23.0) dan [go.mod pada commit yang dipatok](https://github.com/fosrl/cli/blob/bb03d2c45ce79213c4c9b206bbd87e9846be5a9a/go.mod).

### 2.3 Acuan ZTARC Windows

[ZTARC Windows](https://github.com/ztarc/ztarc-windows) menyimpan upstream sebagai submodule dan menerapkan rebranding pada staging sebelum kompilasi. Adopsi pemisahan upstream, patch, override, audit hasil transformasi, dan metadata release.

Jangan menyalin substitusi Windows secara global. CLI memiliki namespace, service, updater, proses elevasi, serta kontrak engine sendiri. Build ZTARC perlu dapat ditelusuri ke source upstream dan patch yang benar.

## 3. Batas pekerjaan antarproyek

| Proyek | Tanggung jawab |
| --- | --- |
| `ztarc-cli` | Pin upstream; rebrand; credential storage dan handoff; namespace; service; installer; packaging; CI; release; source bundle; migrasi host |
| `ztarc-console` | Metadata release; pin aktif; endpoint installer; UI Linux, Spoke, dan Machines; instruksi; adapter protokol native; pengujian regresi |
| Middleware console | Validasi identitas/scope tenant, enforcement mutasi, operasi durable, audit, rekonsiliasi, dan metadata lifecycle Machines |
| Deployment | Mirror binary; registry image; signature/provenance; routing ingress; promosi dan rollback |
| ZTARC Windows | Companion dan menu instalasi CLI ketika pasangan versi telah diuji |
| Backend Pangolin | Perubahan server yang diperlukan untuk menutup jalur mutasi atau menambah authorization yang tidak bisa ditegakkan oleh adapter HTTP saja |

Workstream CLI mencakup mode machine dan handoff pemasangannya. Aktivasi UI Machines dasar dikerjakan pada console sebagai tahap A1 setelah gate terkait lulus; upgrade server otomatis dan migrasi massal connector existing tetap di luar cakupan. Kebutuhan perubahan backend harus mempunyai patch dan image yang dipatok serta jalur deployment tersendiri.

## 4. Struktur repositori CLI

```text
ztarc-cli/
  upstream/                    # submodule pada commit tetap
  brand/
    patches/                   # perubahan perilaku dengan context terarah
    overrides/                 # file kecil yang dimiliki ZTARC
    replacements/              # transformasi dengan target dan jumlah ekspektasi
    audit-exceptions.json      # identifier upstream yang sengaja dipertahankan
  release/
    version.json               # produk, upstream, engine, toolchain, provenance
    platforms.json             # platform dan status dukungan
    compatibility.json         # pasangan server dan kemampuan beserta hasil uji
  scripts/                     # stage, audit, build, package, verify, source bundle
  packaging/
    linux/
    docker/
  tests/
    integration/
    migration/
  docs/
  .github/workflows/
  LICENSE
  NOTICE
  THIRD_PARTY_NOTICES
```

Urutan pipeline: verifikasi commit dan metadata; staging dari upstream bersih; terapkan patch; jalankan substitusi terarah; salin override; format; audit; test; build; package; sign; verifikasi hasil signing; publish candidate; canary; promote.

Staging harus idempotent dan tidak menulis ke submodule. Patch atau substitusi yang tidak cocok menggagalkan build. Audit memeriksa hasil yang wajib ada, bukan hanya tidak ditemukannya kata Pangolin.

Build lokal dan CI menggunakan entrypoint yang sama. Release menolak source tree kotor dan mencatat commit ZTARC, commit upstream, versi engine, inventory patch, toolchain, platform, serta hash artefak.

## 5. Identitas produk dan kontrak kompatibilitas

| Permukaan | Desain |
| --- | --- |
| Command root dan executable | `ztarc-cli` |
| Login default | `https://console.ztarc.io` |
| Server eksplisit | Tetap didukung untuk staging atau deployment yang ditentukan pengguna |
| Konfigurasi pengguna Linux | `~/.config/ztarc-cli/` |
| Konfigurasi Spoke baru | `/etc/ztarc/site.json` |
| Konfigurasi machine client Linux | `/etc/ztarc/client.json`, owner root dan mode `0600` |
| Konfigurasi connector hasil migrasi | `/etc/ztarc/connector.json` tetap dipakai |
| Binary Linux | `/usr/local/bin/ztarc-cli` |
| Service baru | `ztarc-site.service` dan `ztarc-client.service` |
| Service hasil migrasi | `ztarc-connector.service` tetap dipertahankan |
| Image | `registry.ztarc.io/ztarc/cli` |
| Versi produk awal | `0.23.0-ztarc.1` |

Pisahkan versi produk, versi CLI upstream, versi Newt, dan versi Olm. Versi yang dikirim pada field negosiasi backend mengikuti kontrak upstream; versi produk ZTARC ditampilkan melalui help, output version, metadata release, dan provenance. Channel `candidate` atau `stable` adalah keputusan release terpisah, bukan hasil menafsirkan suffix versi.

Pertahankan import/module path upstream pada implementasi awal, API path, field JSON, cookie native, site type `newt`, agent protocol `cli`, serta identifier wire protocol. Prefix credential dan marker protokol bukan sasaran substitusi branding.

Linux CLI memakai autentikasi mandiri. Jangan otomatis membaca account, token, socket, atau konfigurasi aplikasi desktop Pangolin yang kebetulan terpasang. Namespace socket dan state CLI harus terpisah, dengan ownership yang membatasi akses ke pengguna yang berhak dan root.

## 6. Tahapan kemampuan

| Tahap | Kemampuan |
| --- | --- |
| A | Login/logout/auth status; account/org selection; config; client up/down/status/logs; reset DNS dan watchdog; site; service; version/provenance |
| A1 | Aktivasi Machines dasar di console; headless CLI/service/container; direct grants; lifecycle manual yang tercatat |
| B | Alias listing; SSH/sign/SCP; auth-daemon; exit-node/gateway selection |
| C | Blueprint; konfigurasi AI clients; kemampuan lain yang memerlukan integrasi kebijakan server |
| D | Windows companion, macOS, package manager, dan automatic updater ZTARC |

Target parity mencakup command Linux pada baseline upstream. Setiap command harus mempunyai status `supported`, `not_tested`, `unsupported`, atau `deferred` dalam matriks release. Command yang belum tersedia memberi pesan terarah sebelum melakukan mutasi; keberadaan source command tidak berarti kemampuan tersebut telah didukung layanan.

Status dukungan command, kemampuan server, dan hasil uji platform merupakan tiga hal terpisah. Cross-compilation tidak membuktikan tunnel, service, atau integrasi desktop bekerja.

### 6.1 Mode pengguna dan mode service

Login dijalankan oleh pengguna biasa. Runtime tunnel dapat memerlukan elevasi untuk TUN, route, dan DNS. Handoff ke proses elevated harus membawa snapshot akun, organisasi, credential, dan pengaturan yang sudah dipilih; jangan membaca ulang profil root sebagai pengganti profil pengguna.

Gunakan stdin atau IPC terproteksi untuk credential handoff. Hindari secret pada argv, command shell, log, dan file sementara yang dapat dibaca pengguna lain. Uji batas ukuran payload, proses parent yang mati, child gagal start, dan elevasi yang dibatalkan.

Service client untuk machine credentials dibedakan dari sesi pengguna interaktif. Mode machine memakai ID/secret Olm, endpoint, dan organisasi yang diprovisioning administrator; tidak memerlukan login browser atau `user-token`. Runtime ini sudah menjadi kemampuan upstream, sedangkan aktivasi enrollment di console mengikuti tahap A1 dan gate pada bagian 10.4.

Tambahkan input `--credentials-file` pada command client dan instalasi service ZTARC agar console tidak perlu mencetak secret pada argv. Ini tambahan ZTARC yang belum tersedia pada baseline upstream. File JSON machine memuat `id`, `secret`, `endpoint`, dan `org`; tidak memuat sesi pengguna. Tolak kombinasi file dengan flag credential eksplisit, file yang dapat dibaca group/other, ownership yang tidak sesuai, dan symlink. Service membaca credential dari file root-only, bukan meneruskannya sebagai argumen subprocess.

Contoh target setelah tambahan tersebut diimplementasikan adalah `sudo ztarc-cli service install client --credentials-file /etc/ztarc/client.json`. Installer binary tetap terpisah dari enrollment dan pembuatan file credential.

### 6.2 Penyimpanan dan service

Konfigurasi berisi token/secret dibuat owner-only sejak awal, ditulis atomik, dan tidak mengikuti symlink yang dapat menyebabkan overwrite file lain. Direktori credential tidak world-readable. Log dan diagnostic bundle melakukan redaksi credential.

Installer binary tidak otomatis login, membuat device, memasang service, atau mengganti runtime existing. Instalasi service menolak konflik kepemilikan unit. Semua service memakai path binary/config eksplisit dan mempunyai perilaku stop/restart yang teruji.

Tooling status/logs/uninstall harus mengenali unit `ztarc-connector` hasil migrasi. Jangan mengasumsikan semua Spoke CLI berjalan sebagai `ztarc-site`.

## 7. Autentikasi dan integrasi native console

### 7.1 Alur login

1. CLI meminta kode melalui device authorization Pangolin.
2. CLI membuka atau menampilkan URL `https://console.ztarc.io/auth/login/device?code=...`.
3. Console memverifikasi identitas dan meminta persetujuan memakai mekanisme sesi fresh existing.
4. CLI menerima sesi native melalui polling.
5. CLI memuat organisasi yang boleh diakses, memilih organisasi, lalu memperoleh credential perangkat sendiri.
6. Middleware menyelesaikan cleanup sesi persetujuan setelah exchange, mengikuti mekanisme existing.

CLI menggunakan sesi native Pangolin. Cookie browser console yang tersegel tidak diekspor menjadi credential CLI. Jalur enrollment UI Linux tidak membuat Olm tambahan sebelum CLI login.

Uji password, SSO, MFA, expiry kode, cancel, retry/429, sesi dirotasi, pergantian organisasi, approval pending, serta logout. Keberadaan alur Windows existing hanya menjadi dasar reuse, bukan bukti acceptance CLI.

### 7.2 Adapter native

Pertahankan URL dan envelope respons native yang dibutuhkan CLI. Buat daftar method/path eksplisit yang terpisah dari allowlist browser `/api/pg`. Setiap entri mencatat command pemakai, identitas/scope, query/body, field secret yang boleh dikembalikan, dan bukti authorization upstream.

Adapter memvalidasi sesi native, pengguna yang diminta, keanggotaan organisasi, masa aktif tenant yang relevan, serta permission resource. Propagasikan rotasi sesi tanpa meneruskan cookie yang tidak diperlukan. Mutasi menolak browser cross-origin sambil tetap menerima client native yang tidak mengirim Origin.

Routing Traefik hanya mengarahkan method/path yang sudah direview. Tidak membuka seluruh `/api/v1`, tidak mengubah `/api/pg` menjadi proxy token native, dan tidak membuka Integration API publik. Jalur Newt/Olm auth dan WebSocket existing dipertahankan.

Tahap A mengaudit identity/server-info, daftar organisasi pengguna, enrollment dan recovery Olm, lookup org/client, my-device, dan logout. Auth start/poll tetap menggunakan jalur existing yang sudah ditetapkan.

Machine tidak melewati persetujuan browser atau adapter sesi pengguna ketika menjalankan tunnel. Gunakan autentikasi Olm dan WebSocket yang sesuai dengan kontrak engine; provisioning dan perubahan grant tetap dilakukan administrator melalui console/middleware. Jangan memaksa endpoint machine menggunakan cookie pengguna atau membuka API manajemen native untuk memudahkan instalasi.

### 7.3 Pengaturan organisasi

CLI 0.23.0 meminta endpoint organization-managed client settings yang belum ditemukan pada source server 1.26.0 yang diperiksa. Catat kemampuan ini sebagai `unsupported` pada pasangan tersebut.

Jangan mengembalikan pengaturan kosong seolah-olah managed policy telah berhasil dimuat. Perilaku fallback lokal harus eksplisit dan diuji, termasuk perbedaan endpoint belum tersedia dengan kegagalan memuat policy yang sebelumnya wajib. Dokumen pengguna tidak boleh menjanjikan enforcement setting server sebelum kemampuan backend tersedia.

## 8. Fitur sensitif dan enforcement

### 8.1 SSH dan SCP

Audit alias/resource discovery, sertifikat SSH, auth-daemon, status pesan JIT, expiry sertifikat, dan otorisasi pengguna terhadap resource. Native SSH options tetap mengikuti perilaku upstream setelah dukungan dinyatakan lulus.

Handler status round-trip pada baseline server mencari `messageId` tanpa pemeriksaan kepemilikan di handler tersebut. Adapter harus mengikat pesan yang diterbitkan saat signing ke identitas/sesi dan organisasi peminta, menyimpan hubungan secara durable dengan expiry, dan menolak polling milik pihak lain. Routing tidak boleh menyediakan jalur bypass ke handler asli.

Rujukan: [handler status pesan](https://github.com/fosrl/pangolin/blob/1.26.0/server/routers/ws/checkRoundTripMessage.ts) dan [SSH signing](https://github.com/fosrl/pangolin/blob/1.26.0/server/routers/ssh/signSshKey.ts).

### 8.2 Blueprint

Blueprint merupakan mutasi yang dapat mengubah resources dan asosiasinya. Dukungan membutuhkan RBAC, pembatasan domain, masa aktif tenant, admission/kuota yang relevan, writer gate, idempotency, audit, dan rekonsiliasi kegagalan. Seluruh perubahan dilakukan melalui boundary yang dikendalikan ZTARC.

Backend juga menerima blueprint melalui pesan Newt WebSocket. Penutupan endpoint HTTP atau command pada CLI tidak menutup jalur tersebut. Sebelum fitur blueprint diaktifkan, enforcement harus mencakup HTTP dan WebSocket pada server. Menyembunyikan command atau menolak flag pada binary resmi hanya mengatur dukungan produk, bukan kontrol keamanan server.

Tahap A menolak input blueprint yang akan dikirim oleh mode site resmi, termasuk sumber file, flag, environment, atau konfigurasi yang diterima engine. Jalur server existing harus dicatat sebagai temuan enforcement sampai guard server tersedia; jangan menyatakan bahwa seluruh bypass sudah tertutup.

Rujukan: [handler blueprint Newt](https://github.com/fosrl/pangolin/blob/1.26.0/server/routers/newt/handleApplyBlueprintMessage.ts).

### 8.3 Konfigurasi AI clients

Command konfigurasi AI memerlukan resource discovery dan akses virtual key. Dukungan hanya membuka credential milik peminta terhadap resource yang diizinkan, tanpa membocorkan provider key atau credential tenant lain.

Perubahan file konfigurasi aplikasi lokal mempertahankan field yang tidak dimiliki CLI. Tampilkan file yang akan diubah, lakukan penulisan atomik, dan uji reset agar tidak menghapus konfigurasi lain. Integrasi ini tidak otomatis mengaktifkan administrasi AI Gateway di console.

## 9. Distribusi dan pembaruan

### 9.1 Kontrak distribusi console

Tambahkan kontrak release CLI tersendiri tanpa mengubah schema pin Newt/Olm/Windows existing.

| Interface baru | Fungsi |
| --- | --- |
| `GET /api/cli-release` | Metadata release aktif, versi produk/engine, platform, checksum, capability, image digest, dan immutable installer path |
| `GET /get-cli.sh` | Installer yang mengikuti pin CLI aktif |
| `GET /get-cli/<releaseId>.sh` | Installer untuk pilihan release immutable |
| `https://dl.ztarc.io/cli/<version>/...` | Binary dan metadata release yang tidak ditimpa |
| `registry.ztarc.io/ztarc/cli@sha256:...` | Image hasil build ZTARC yang dipatok digest |

Manifest release memuat provenance, hash/ukuran binary per arsitektur, hash installer, image digest, pasangan server, kemampuan, dan rujukan bukti acceptance. `releaseId` mengidentifikasi bytes manifest yang tepat. Bentuk final schema menjadi shared contract antara proyek CLI dan console; perubahan incompatible menggunakan versi schema baru.

Pin aktif CLI disimpan terpisah pada direktori distribusi runtime yang sama. Pemilihan memvalidasi manifest dan installer; tidak ada fallback `latest`. Metadata stabil memakai `no-store`, sedangkan bytes artefak yang benar-benar immutable boleh di-cache.

Setiap alur UI mengambil satu snapshot release. Command yang sudah diberikan tetap menunjuk pilihan immutable walaupun operator melakukan promosi/rollback. Menutup dan membuka ulang alur memilih pin aktif terbaru. Kegagalan metadata memblokir alokasi credential yang membutuhkan artefak tersebut.

Promosi memakai lock, backup pin, fsync, dan atomic replacement. Candidate dipakai melalui URL immutable untuk canary; stable hanya dipromosikan setelah bukti acceptance diverifikasi. Catatan boolean tanpa rujukan hasil uji tidak cukup menjadi bukti release.

### 9.2 Binary dan source release

Build artefak ZTARC sendiri; checksum upstream tidak membuktikan integritas binary modifikasi. Setiap release menyediakan checksum, signature/provenance, source bundle lengkap, license/notices, serta instruksi build.

Source bundle mencakup source upstream pada commit yang benar, patch ZTARC, metadata dependency, dan script build. Archive yang hanya memuat pointer submodule tidak cukup sebagai source bundle mandiri.

Gunakan signing artefak dengan identitas release ZTARC dan prosedur verifikasi yang didokumentasikan. Installer berbasis HTTPS dan pinned checksum memiliki batas kepercayaan terhadap jalur distribusi; jangan menyamakannya dengan updater otomatis yang memverifikasi manifest memakai trust anchor terpisah.

### 9.3 Kebijakan update awal

Nonaktifkan semua jalur download/apply upstream: command update, pemeriksaan versi saat startup/version, self-update site, dan mekanisme engine/client lain yang ditemukan pada dependency yang dipatok. Flag konfigurasi default saja tidak cukup.

Command update awal memberikan instruksi manual ZTARC. Update dilakukan setelah tunnel dihentikan secara eksplisit dan binary lama disimpan. Installer memverifikasi bytes sebelum replacement, tidak mengubah credential/service, serta tidak mengganti executable yang sedang dipakai tanpa penanganan yang teruji.

Container diperbarui melalui pull/recreate dengan digest yang dipilih, tanpa self-modification. Automatic updater baru masuk tahap D setelah signature manifest, trust anchor, rotasi kunci, anti-replay, replacement atomik, dan rollback tersedia.

Rujukan updater upstream: [Unix update](https://github.com/fosrl/cli/blob/bb03d2c45ce79213c4c9b206bbd87e9846be5a9a/cmd/update/update_unix.go) dan [self-update site](https://github.com/fosrl/cli/blob/bb03d2c45ce79213c4c9b206bbd87e9846be5a9a/cmd/up/site/site.go).

## 10. Pengalaman pengguna console

### 10.1 Add device

Tambahkan pilihan Linux CLI bersama Windows dan Docker existing. Tampilkan versi yang terpilih, arsitektur yang didukung, installer immutable, serta instruksi login dengan origin console dan organisasi yang dipilih.

Urutan pengguna: install binary; jalankan login; setujui lewat browser; jalankan koneksi; periksa status. Enrollment dilakukan oleh CLI setelah autentikasi. UI browser tidak membuat credential perangkat yang tidak dipakai.

Tampilkan kebutuhan sudo untuk tunnel dan arti status pending approval. Saat metadata tidak tersedia, sediakan retry dan jangan tampilkan command instalasi tanpa versi terverifikasi.

### 10.2 Create Spoke

Tambahkan pilihan runtime sebelum provisioning: Newt standalone atau ZTARC CLI. Pilihan runtime merupakan metode deployment, bukan site atau kapasitas baru. Site tetap memakai `type: newt` dan provisioning melalui middleware existing.

Instruksi CLI memuat installer yang dipatok, pembuatan config root-only, dan service `ztarc-site`. Untuk container, gunakan config read-only mount dan image digest; secret tidak ditanamkan ke argumen proses. Kebutuhan TUN, capability, dan network disesuaikan dengan mode yang diuji.

Kedua packaging harus menghasilkan kemampuan Spoke yang konsisten. Jangan menambahkan `disable-clients` atau `disable-ssh` pada satu packaging saja tanpa keputusan capability yang terlihat oleh pengguna.

### 10.3 Detail dan migrasi

Tampilkan versi/agent yang dilaporkan backend. Jangan menebak build ZTARC atau jenis runtime hanya dari nomor Newt. Metadata instalasi yang dicatat operator harus dibedakan dari status runtime live.

Console menyediakan panduan migrasi host dan rollback. Tidak menawarkan tombol migrasi jarak jauh karena console tidak mempunyai executor pada host pengguna.

### 10.4 Aktivasi Machines dasar pada tahap A1

#### Hubungan CLI dengan Machines

| Mode | Identitas | Cara memperoleh akses | Fungsi |
| --- | --- | --- | --- |
| User client | Perangkat dengan `userId` | Login pengguna; mengikuti akses pengguna | Laptop atau workstation |
| Machine client | Client dengan `userId = null` | ID/secret Olm; grant sendiri | Server, CI runner, backup agent, workload |
| Site atau Spoke | Site `type: newt` | Credential connector; menyediakan jalur ke resource | Menghubungkan jaringan lokasi |

Machine merupakan konsumen private resources, bukan pengganti Spoke yang menyediakan koneksi ke jaringan tujuan. CLI menjalankan keduanya melalui mode berbeda. Machines juga dapat memakai Olm standalone; aktivasi konsep Machines tidak bergantung pada tersedianya CLI, tetapi alur instalasi baru akan memakai distribusi CLI ZTARC yang tervalidasi.

Rujukan: [jenis client](https://docs.pangolin.net/manage/clients/understanding-clients), [machine clients dan direct grants](https://docs.pangolin.net/manage/clients/machine-clients), serta [service client pada baseline CLI](https://github.com/fosrl/cli/blob/bb03d2c45ce79213c4c9b206bbd87e9846be5a9a/cmd/service/client.go).

#### Penilaian kompleksitas

Aktivasi UI saja tergolong kecil karena halaman list, enrollment, actions, allowlist, dan grant list sudah tersedia. Rilis yang dapat dipertanggungjawabkan tergolong menengah: perlu merapikan gating, menguji kontrak EE 1.26.0, menghindari secret pada command, serta menyediakan lifecycle manual dan pembatasan tenant.

Fitur ini dinonaktifkan sejak 30 Agustus 2026 karena machine memiliki akses mandiri tanpa mekanisme penanggung jawab, expiry, rotasi, atau review yang memadai. Mengubah `FEATURES.machines` menjadi `true` saja tidak menyelesaikan alasan tersebut. Tahap A1 menambahkan penanggung jawab, review terjadwal secara administratif, revokasi, dan prosedur penggantian credential. Tahap A1 tidak menjanjikan expiry atau rotasi otomatis.

| Area | Kompleksitas relatif | Keputusan tahap A1 |
| --- | --- | --- |
| Navigasi, list, actions | Kecil | Gunakan kembali implementasi existing dengan regresi |
| Enrollment CLI dan secret handling | Menengah | File credential root-only; Linux service dan container |
| Direct resource grants | Kecil sampai menengah | Aktifkan grant langsung; verifikasi isolasi organisasi dan revokasi |
| Lifecycle dan boundary mutasi | Menengah | Metadata penanggung jawab/review, audit, writer gate, idempotency |
| Role inheritance machine | Tambahan menengah | Ditunda; jangan menyamakan dengan role pengelola client |
| Expiry otomatis dan rotasi identitas tetap | Tambahan menengah sampai besar | Workstream lanjutan; perlu enforcement, recovery, serta kontrak backend |

#### Cakupan fungsional pertama

1. Aktifkan menu Machines dan daftar machine per organisasi, termasuk status active, blocked, dan archived. Pisahkan dari user devices secara eksplisit melalui `userId = null`.
2. Administrator tenant melakukan enrollment, memilih penanggung jawab, tujuan penggunaan, dan tanggal review. Machine baru tidak diberi private-resource access secara otomatis.
3. Berikan akses melalui daftar machine langsung pada private service. Role inheritance machine belum tersedia pada tahap ini.
4. Sediakan detail dasar untuk metadata penanggung jawab, review, status runtime, dan direct grants. Gunakan data existing jika backend tidak melaporkan versi build CLI.
5. Sediakan block/unblock, archive/unarchive, dan delete dengan permission yang sesuai. Emergency revocation menggunakan Block; Archive tidak diperlakukan sebagai bukti pemutusan trafik tanpa pengujian.
6. Sediakan prosedur penggantian credential melalui machine baru: block machine lama dan buktikan akses terputus; buat machine baru; administrator mengonfirmasi direct grants; pasang credential baru; uji akses; simpan machine lama dalam keadaan blocked sampai decommission. Identitas berubah dan ada potensi downtime; bukan rotasi in-place.

Jangan menjanjikan regenerate secret berdasarkan dokumentasi upstream terbaru saja. Handler `updateClient` baseline 1.26.0 yang diperiksa hanya menerima `name` dan `niceId`. Dukungan rotasi dengan `clientId` tetap memerlukan kontrak endpoint yang diverifikasi dan pengujian sesi/koneksi lama.

Rujukan source: [createClient](https://github.com/fosrl/pangolin/blob/1.26.0/server/routers/client/createClient.ts), [updateClient](https://github.com/fosrl/pangolin/blob/1.26.0/server/routers/client/updateClient.ts), dan [blockClient](https://github.com/fosrl/pangolin/blob/1.26.0/server/routers/client/blockClient.ts).

#### Metadata lifecycle minimum

Simpan metadata pada schema middleware milik ZTARC, dengan key organisasi dan `clientId`: penanggung jawab berupa user organisasi, tujuan penggunaan, `nextReviewAt`, waktu/aktor review terakhir, waktu/aktor pembuatan, serta audit perubahan. Metadata ini bukan credential dan tidak mengubah `userId` Pangolin; machine tetap identitas mandiri.

Default review pertama adalah 90 hari setelah enrollment. Administrator dapat memajukan tanggalnya; perpanjangan review dilakukan dengan konfirmasi bahwa kebutuhan dan grant masih sesuai. List menampilkan overdue dan menyediakan filter machine yang perlu direview. Keterlambatan review tidak otomatis memutus tunnel pada tahap A1 dan tidak boleh ditampilkan sebagai expiry credential.

Penanggung jawab yang keluar dari organisasi ditandai perlu pengalihan tanggung jawab; tidak otomatis membuat machine menjadi perangkat orang lain. Sebelum perubahan hak yang memperluas akses pada machine tanpa penanggung jawab valid, administrator harus menetapkan pengganti. Penanggung jawab merupakan metadata akuntabilitas, bukan tambahan permission jaringan atau pemberian akses manajemen otomatis.

Machine existing diinventarisasi dan diberi metadata melalui proses adopsi; jangan membuat ulang identitasnya hanya untuk mengisi lifecycle. Status belum direview harus terlihat, dan tidak boleh dianggap otomatis sudah memenuhi policy baru.

#### Provisioning dan mutasi

Pindahkan orkestrasi enrollment dari alur browser dua langkah menjadi satu operasi middleware terautentikasi yang menerima nama dan metadata lifecycle. Middleware memverifikasi administrator/scope, tenant aktif, readiness dan lisensi; mengambil defaults; membuat client melalui API backend; mengikat metadata; dan mengembalikan credential satu kali. Alur tetap memakai ID/subnet/defaults native, bukan menulis tabel Pangolin secara langsung.

Gunakan writer gate dan idempotency existing. Simpan hasil sensitif hanya secara terenkripsi dengan TTL menurut mekanisme custody existing yang tervalidasi. Retry operasi yang sama tidak membuat machine kedua; kegagalan setelah create direkonsiliasi ke identitas yang sama atau ditandai perlu recovery. Jangan meninggalkan machine aktif yang tidak mempunyai metadata akuntabilitas dan tidak bisa dipulihkan oleh administrator.

Perubahan lifecycle dan mutasi machine serta direct grants melewati boundary yang memastikan tenant, permission, dan audit. Tetap izinkan tindakan pengurangan akses yang diperlukan saat tenant tidak aktif; create, unblock, restore, atau penambahan grant tidak boleh menjadi cara mengaktifkan kembali akses tenant. Permission upstream tetap berlaku, tetapi tidak menggantikan kebijakan ZTARC.

Daftar `clientIds` dalam request resource harus benar-benar machine pada organisasi yang sama. Jangan menerima user-device ID sebagai machine grant. Ingat bahwa penyimpanan grant existing dapat mengganti seluruh daftar: picker harus memuat atau mempertahankan grant yang sudah dipilih, termasuk di luar halaman pertama, agar penyimpanan tidak mencabut grant tersembunyi tanpa sengaja.

User dan site quotas existing tidak otomatis menjadi kuota machine. Adapter lisensi saat ini hanya memodelkan `maxUsers` dan `maxSites`; validasi ketentuan/limit Machines pada EE yang terpasang secara terpisah. Tahap A1 melaporkan jumlah machine sebagai kategori tersendiri, mengikuti limit backend yang terverifikasi, dan tidak menampilkan angka unlimited atau mengurangi seat pengguna/site berdasarkan asumsi. Jika limit wajib tidak dapat diverifikasi, enrollment ditahan; pembacaan dan pencabutan akses tetap tersedia.

#### UI dan feature gate

Audit semua entrypoint: menu, halaman list/detail, enrollment, private-service create, private-service detail, route BFF, dan mutasi middleware. Saat analisis, machine picker pada form create private service tidak memakai gate yang sama dengan grant list pada detail; aktivasi harus menyamakan perilaku dan menutup mutasi baru saat fitur belum dirilis.

Direct machine grants yang sudah ada tetap dipertahankan ketika mengedit field resource yang tidak berkaitan. Jangan mengirim daftar kosong karena picker disembunyikan atau gagal dimuat. Batas antara role pengelola client dan role jaringan machine harus jelas; fitur `clientOrgRoles` yang diperkenalkan server 1.25 belum menjadi bagian aktivasi awal ini.

Linux memakai config root-only serta `ztarc-client.service`. Container memakai config mount read-only dengan permission host yang sesuai dan image digest; jangan masukkan secret ke argv atau menyalin user-token. Console tidak membuka Integration API atau native machine-create endpoint ke publik.

#### Gate aktivasi dan rollback

Aktifkan `FEATURES.machines` lewat perubahan source yang direview hanya setelah enrollment, grant, revokasi, metadata, dan pengujian tenant lulus. Tahap A1 dapat dirilis setelah tahap A tanpa menunggu SSH/blueprint/AI; kegagalan A1 tidak menahan rilis client pengguna dan Spoke yang sudah lulus.

Acceptance mencakup machine baru tanpa grant tidak bisa mengakses resource; grant hanya membuka tujuan/protokol yang dipilih; pencabutan dan Block memutus akses aktif serta mencegah reconnect; user logout tidak mematikan machine; identitas machine tidak berpindah organisasi; service/container survive reboot tanpa sesi pengguna; dan penggantian credential tidak menyisakan akses dari machine lama.

Rollback penawaran baru tidak boleh menghilangkan kontrol Block/Delete terhadap machine yang telah dipasang. Jika enrollment dinonaktifkan kembali, pertahankan halaman pengelolaan dan revokasi untuk inventory existing, dengan mode tanpa create atau grant baru. Jangan hanya mematikan satu flag yang menyembunyikan seluruh akses pengelolaan sementara workload masih berjalan.

## 11. Migrasi connector existing

Migrasi mempertahankan site ID, credential, tenant, resource assignment, dan perhitungan kuota. Jangan membuat ulang site sebagai prosedur upgrade standar. Perpindahan proses dapat memutus koneksi aktif; jangan menjanjikan zero downtime.

### 11.1 Linux systemd

1. Inventaris binary, unit, config, flag, network, arsitektur, dan sumber credential lokal.
2. Verifikasi artefak CLI dan kompatibilitas; pasang binary berdampingan.
3. Validasi config tanpa membuka tunnel kedua; buat backup root-only dan state migrasi.
4. Ambil lock host; hentikan unit lama dan pastikan proses berhenti.
5. Ubah unit existing ke `ztarc-cli up site --config-file /etc/ztarc/connector.json` dengan atribut operasional yang diperlukan.
6. Reload/start, kemudian periksa identitas site, akses resource, DNS, dan reconnect.
7. Catat berhasil setelah dataplane lulus; simpan binary/unit lama selama masa observasi.

Tool migrasi harus idempotent, dapat melanjutkan atau memulihkan state setelah crash/reboot, dan tidak menjalankan dua connector dengan identitas yang sama. Secret hanya dibaca dari host; console tidak mengambil ulang plaintext secret yang telah hilang.

Rollback menghentikan CLI, memulihkan unit/config lama jika berubah, menjalankan binary lama, lalu menguji resource. Kegagalan rollback melaporkan state terakhir dan langkah recovery tanpa loop pergantian runtime tanpa batas.

### 11.2 Docker

Pre-pull image baru dan simpan definisi deployment lama, digest, restart policy, network, volume, serta sumber secret. Hentikan container lama sebelum pengganti dijalankan. Pertahankan pilihan network saat migrasi agar perubahan runtime tidak bercampur dengan perubahan topologi.

Rollback menghentikan container CLI sebelum deployment lama dihidupkan kembali. Cegah restart policy atau orchestrator menghidupkan dua instance dengan credential site yang sama. Kubernetes, NAS, dan deployment nonstandar memerlukan runbook tersendiri sebelum dinyatakan didukung.

## 12. Pengujian dan acceptance

| Kelompok | Skenario wajib |
| --- | --- |
| Source/build | Pin/dependency; staging berulang; patch drift; audit branding dan protocol marker; version/provenance |
| Credential | Izin sejak pembuatan; atomic write; symlink; redaksi; argv/process inspection; handoff akun pengguna ke root |
| Login | Password/SSO/MFA; expiry; cancel; retry/429; rotasi sesi; org selection; logout; cleanup approval |
| Machines A1 | Headless tanpa sesi pengguna; enrollment idempotent; owner/review/adopsi; direct grants; revoke/Block dan reconnect; tenant inactive; pergantian credential; rollback tanpa kehilangan kontrol revokasi |
| Client | TUN; route/DNS; DNS restoration setelah crash; status/down/logs; approval; suspend/reconnect; reboot |
| Spoke | Linux dan container; HTTP/TCP/UDP; private resources; signal shutdown; reconnect; identitas site |
| Tenant | Lintas user/org; resource tanpa izin; masa aktif; admission; provisioning bypass; API tidak terdaftar |
| SSH tahap B | Sertifikat/principal; JIT polling ownership; expiry; account switch; akses yang ditolak |
| Blueprint tahap C | HTTP dan WS enforcement; mutasi parsial; concurrent quota; retry/idempotency; rekonsiliasi |
| Distribusi | Manifest/installer mismatch; signature salah; mirror offline; salah arsitektur; pin korup; concurrent promotion; rollback |
| Update | Tidak ada replacement upstream; config yang mencoba mengaktifkan update; container tidak self-modify |
| Migrasi | Cutover; rollback; interupsi/crash/reboot; credential tidak tersedia; pencegahan proses ganda |
| Regresi | Windows, Newt/Olm, device authorization, middleware admission, dan flow provisioning existing |

Acceptance tahap A membutuhkan runtime Linux amd64 dan arm64, container yang diiklankan, reboot, gangguan koneksi, serta rollback yang berhasil. Canary minimal 24 jam harus memverifikasi akses resource, bukan hanya heartbeat/online.

Kegagalan isolasi tenant, kebocoran credential, replacement upstream, atau kerusakan dataplane menghentikan promosi. Fitur yang belum diuji tetap `not_tested`; platform tanpa perangkat uji tidak dinyatakan lulus dari hasil cross-build.

Pemeriksaan awal console selama analisis: suite Windows release, device authorization, dan approval journal lulus. Pemeriksaan distribusi terpilih lulus, tetapi kasus kernel locking gagal saat dijalankan terpisah dan penyebab belum ditetapkan. Reproduksi dan penyelesaian kasus ini diperlukan sebelum memakai mekanisme sejenis sebagai dasar promosi CLI.

## 13. Urutan implementasi dan hasil tiap tahap

| Tahap kerja | Proyek utama | Hasil yang dapat direview | Syarat selesai |
| --- | --- | --- | --- |
| 0 Kontrak | CLI dan console | Metadata baseline, inventory command/endpoint, schema distribusi, matriks dukungan | Tidak ada asumsi protokol kritis yang belum dipetakan |
| 1 Fondasi | CLI | Repo/submodule, staging, patch, audit, versi/provenance | Build dan pemeriksaan perubahan lulus |
| 2 Runtime aman | CLI | Namespace, credential storage/handoff, service, updater policy, guard kemampuan | Tidak ada credential otomatis pada argv atau replacement upstream |
| 3 Packaging | CLI dan deployment | Binary dua arsitektur, container, installer, source bundle, checksum/signature | Artefak dapat diverifikasi dan dipasang tanpa mengubah instalasi existing |
| 4 Integrasi tahap A | Console | Native adapter, metadata/pin, installer routes, UI Linux/Spoke, ingress overlay | Contract/regression tests lulus; belum promosi otomatis |
| 5 Canary | Operator dan maintainer | Bukti Linux/container, dataplane, 24 jam, migrasi dan rollback | Semua kemampuan tahap A yang diiklankan lulus |
| 6 Stable tahap A | Deployment dan console | Promosi bytes yang telah diuji; CLI sebagai default Spoke baru yang tervalidasi | Rollback pin dan rollback host tersedia |
| 6a Machines tahap A1 | CLI, console, middleware | Headless config/service/container, enrollment dan direct grants, metadata lifecycle, UI pengelolaan dan revokasi | Gate bagian 10.4 lulus; tanpa role inheritance, expiry otomatis, atau rotasi in-place |
| 7 Tahap B | CLI, console, middleware | SSH/SCP, alias, auth-daemon, exit-node/gateway beserta authorization | Matriks kemampuan tahap B lulus |
| 8 Tahap C | Backend, middleware, CLI | Blueprint dan AI dengan enforcement yang sesuai | Jalur HTTP/WS dan credential scope tervalidasi |
| 9 Tahap D | CLI dan Windows | Companion, macOS, package managers, signed updater | Acceptance platform dan lifecycle masing-masing lulus |

Promosi menggunakan artefak candidate yang telah diuji; jangan rebuild bytes berbeda hanya untuk memberi label stable. Pisahkan metadata channel dari identitas artefak agar pemindahan channel tidak menuntut rebuild.

Rollback UI mengembalikan default instalasi baru ke Newt dan/atau pin CLI ke release sebelumnya. Rollback tersebut tidak menurunkan versi endpoint yang sudah terpasang; rollback host dijalankan terpisah. Route autentikasi runtime yang sudah terpasang tidak boleh dihapus hanya karena penawaran CLI dihentikan.

## 14. Handoff ke proyek CLI

Mulai proyek `ztarc-cli` dengan tahap 0 sampai 3. Salin dokumen ini sebagai konteks desain dan sertakan kontrak integrasi pada dokumentasi CLI, agar implementer tidak memerlukan akses ke percakapan asal.

Sebelum meminta perubahan console, proyek CLI menyerahkan:

- Commit upstream dan ZTARC yang tepat serta inventory patch.
- Metadata versi produk/engine dan capability matrix.
- Binary candidate amd64/arm64 serta image digest yang tersedia.
- Installer, checksum/signature, provenance, dan source bundle.
- Daftar method/path/query/body yang dipakai tiap command dan perlakuan field secret.
- Kontrak machine mode, input credential file yang aman, serta perilaku service/container tanpa sesi pengguna untuk tahap A1.
- Bukti perilaku updater, credential handoff, namespace, service, serta status pengujian runtime.
- Panduan instalasi baru, migrasi, dan rollback yang telah diuji sejauh tersedia.

Console kemudian mengimplementasikan tahap 4 terhadap kontrak tersebut. Pekerjaan tahap B/C dapat disiapkan paralel, tetapi status dukungannya mengikuti gate masing-masing. Desain ini tidak memberikan instruksi untuk langsung mempublikasikan artefak, mengubah pin production, atau memigrasikan host pengguna.
