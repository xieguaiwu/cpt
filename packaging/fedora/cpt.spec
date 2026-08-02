Name:           cpt
Version:        1.2.0
Release:        1%{?dist}
Summary:        Terminal companion for competitive-companion browser extension

License:        MIT
URL:            https://github.com/xieguaiwu/cpt
Source0:        %{url}/archive/v%{version}.tar.gz#/%{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.22

%description
cpt (Competitive Programming Tool) is a lightweight terminal companion
for the competitive-companion browser extension. It bridges the browser-
to-terminal gap for competitive programming: click a button in your
browser, and cpt saves sample test cases and runs your solution — all
in one seamless flow. Supports 100+ online judges (Luogu, USACO,
Codeforces, AtCoder, and more) through competitive-companion's parsers.

%prep
%setup -q -n cpt-%{version}

%build
export GOFLAGS="-mod=vendor"
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

go build -trimpath -ldflags="-s -w" -o cpt .

%install
rm -rf %{buildroot}
install -Dm755 cpt %{buildroot}%{_bindir}/cpt
install -Dm644 LICENSE %{buildroot}%{_defaultlicensedir}/%{name}/LICENSE
install -Dm644 README.md %{buildroot}%{_defaultdocdir}/%{name}/README.md
install -Dm644 README_zh.md %{buildroot}%{_defaultdocdir}/%{name}/README_zh.md

%files
%license LICENSE
%doc README.md README_zh.md
%{_bindir}/cpt

%changelog
* Sun Aug 02 2026 xgw <xieguaiwu@163.com> - 1.1.1-1
- Align compiler flags with nvim utils.lua: C++ g++ -std=c++17 -O2 -Wall -Wextra -Wshadow, C gcc -std=c11 -O2 -Wall
- Keep compiled binary in source dir (main.cpp -> ./main) for reuse with custom samples; fix exec PATH fallback for relative paths

* Sun Jul 26 2026 xgw <xieguaiwu@163.com> - 1.0.1-1
- Security fixes: localhost-only bind, shared secret auth, rate limiting, body size limit, test count cap, error sanitization, CSRF protection

* Sun Jul 26 2026 xgw <xieguaiwu@163.com> - 1.0.0-1
- Initial Go package

%global debug_package %{nil}
