#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$0")"
package_dir=$(pwd -P)
app_dir=/opt/dujiao-next-vmq
[[ $(id -u) == 0 ]] || { echo '请在 1Panel 终端用 root 执行：bash start.sh' >&2; exit 1; }
command -v systemctl >/dev/null || { echo '需要支持 systemd 的 Linux 系统'; exit 1; }
[[ -f dujiao-next ]] || { echo '缺少 dujiao-next，请使用完整 Linux 安装包'; exit 1; }
if [[ "$package_dir" != "$app_dir" ]]; then
  if [[ -e "$app_dir/config.yml" || -e "$app_dir/config.yaml" || -e "$app_dir/db" ]]; then
    echo '已有商城配置或数据库，禁止覆盖。请从 /opt/dujiao-next-vmq 目录启动原服务。' >&2
    exit 1
  fi
  mkdir -p -- "$app_dir"
  for file in dujiao-next start.sh; do
    install -m 750 -- "$file" "$app_dir/$file.new"
    mv -f -- "$app_dir/$file.new" "$app_dir/$file"
  done
fi
cd -- "$app_dir"
id dujiao-vmq >/dev/null 2>&1 || useradd --system --home-dir "$app_dir" --shell /usr/sbin/nologin dujiao-vmq
chown -R dujiao-vmq:dujiao-vmq "$app_dir"
chmod 750 dujiao-next start.sh
[[ ! -f config.yml ]] || chmod 600 config.yml
[[ ! -f db/config.yml ]] || chmod 600 db/config.yml
cat > /etc/systemd/system/dujiao-next-vmq.service <<EOF
[Unit]
Description=Dujiao Next VMQ Web Installer
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=dujiao-vmq
Group=dujiao-vmq
WorkingDirectory=$app_dir
ExecStart=$app_dir/dujiao-next
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
UMask=0077

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable dujiao-next-vmq.service
systemctl reset-failed dujiao-next-vmq.service
systemctl restart dujiao-next-vmq.service
for ((i=0;i<40;i++)); do
  if [[ -s .install-token || -f config.yml || -f config.yaml || -f db/config.yml ]]; then break; fi
  sleep 0.2
done
if ! systemctl is-active --quiet dujiao-next-vmq.service; then
  journalctl -u dujiao-next-vmq.service -n 40 --no-pager
  exit 1
fi
if [[ -s .install-token && ! -f db/config.yml && ! -f config.yml && ! -f config.yaml ]]; then
  printf '\n浏览器打开（将 SERVER_IP 替换成服务器公网 IP）：\nhttp://SERVER_IP:8081/install?token=%s\n' "$(cat .install-token)"
  echo '首次访问需要放行 TCP 8081；也可在 1Panel 创建反向代理网站，目标 http://127.0.0.1:8081。'
  echo '打开专属链接后，网页填写 Redis、管理员和网站地址即可。'
else
  echo '服务已启动，请访问你的商城域名。'
fi
echo '查看日志：journalctl -u dujiao-next-vmq -n 60 --no-pager'
