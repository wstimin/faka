package i18n

func init() {
	for locale, bundle := range map[string]map[string]string{
		LocaleZH: {
			"title": "独角发卡 · 安装向导", "intro": "填写连接信息，自动创建数据库、管理员与安全密钥。",
			"authorization": "安装授权", "token": "安装授权码", "token_hint": "使用启动脚本显示的专属安装链接可自动填写。授权码也保存在安装目录的 .install-token 文件中。",
			"redis": "01　Redis 与发货队列", "host": "Redis 地址", "port": "端口", "redis_password": "Redis 密码（没有则留空）",
			"redis_hint": "填写 1Panel 中宿主机可访问的地址和映射端口。程序会检查缓存库 0 和发货队列库 1，无需创建 MariaDB 数据库。",
			"admin":      "02　管理员", "username": "用户名", "password": "密码", "confirm": "确认密码", "password_hint": "至少 8 个字符，包含大写字母、小写字母和数字；最多 72 字节。",
			"site": "03　网站", "site_url": "网站访问地址", "site_hint": "例如 https://shop.example.com；首次测试也可使用 http://服务器IP:8081。正式收款前请配置域名和 HTTPS。",
			"submit": "确认信息并安装", "database_hint": "使用内置 SQLite 自动建库。管理员密码只保存加密哈希，已有站点不会被覆盖。",
			"working": "正在安装…", "working_hint": "正在验证 Redis 并初始化数据，请等待，不要关闭页面。",
			"success": "安装完成", "starting": "配置已保存，安装入口已锁定。正在等待商城启动，随后自动进入后台。", "enter": "进入管理后台", "startup_wait": "等待商城启动超时。请查看服务日志，修复后点击“进入管理后台”，不要重新安装。",
			"invalid_input": "输入格式不正确或表单过大。", "invalid_redis": "请填写正确的 Redis 地址和端口。", "invalid_admin": "请填写有效的管理员用户名及符合要求的密码，密码首尾不能有空格。", "invalid_url": "网站地址须为 http://IP:端口 或 https://域名，不要添加路径。",
			"redis_failed": "Redis 连接验证失败，请检查运行状态、宿主机地址、端口、密码及数据库 0/1 的访问权限。", "save_failed": "保存安装信息失败，请检查安装目录写入权限和剩余空间。", "database_failed": "初始化数据库失败，未保存安装结果；请检查服务日志后重试。",
			"locked": "已有配置或数据库，安装入口已锁定。", "cross_site": "不允许跨站提交安装信息。", "invalid_token": "安装授权码不正确，请使用启动脚本显示的专属安装链接。", "password_mismatch": "两次输入的管理员密码不一致。",
		},
		LocaleTW: {
			"title": "獨角發卡 · 安裝精靈", "intro": "填寫連線資訊，自動建立資料庫、管理員與安全金鑰。",
			"authorization": "安裝授權", "token": "安裝授權碼", "token_hint": "使用啟動腳本顯示的專屬安裝連結可自動填入。授權碼亦儲存在安裝目錄的 .install-token 檔案。",
			"redis": "01　Redis 與發貨佇列", "host": "Redis 位址", "port": "連接埠", "redis_password": "Redis 密碼（無則留空）",
			"redis_hint": "填寫 1Panel 中主機可存取的位址和對映連接埠。程式會檢查快取資料庫 0 和發貨佇列資料庫 1，無需建立 MariaDB 資料庫。",
			"admin":      "02　管理員", "username": "使用者名稱", "password": "密碼", "confirm": "確認密碼", "password_hint": "至少 8 個字元，包含大寫字母、小寫字母和數字；最多 72 位元組。",
			"site": "03　網站", "site_url": "網站存取位址", "site_hint": "例如 https://shop.example.com；首次測試亦可使用 http://伺服器IP:8081。正式收款前請設定網域和 HTTPS。",
			"submit": "確認資訊並安裝", "database_hint": "使用內建 SQLite 自動建立資料庫。管理員密碼僅儲存加密雜湊，既有網站不會被覆寫。",
			"working": "正在安裝…", "working_hint": "正在驗證 Redis 並初始化資料，請稍候，不要關閉頁面。",
			"success": "安裝完成", "starting": "設定已儲存，安裝入口已鎖定。正在等待商城啟動，隨後自動進入後台。", "enter": "進入管理後台", "startup_wait": "等待商城啟動逾時。請查看服務日誌，修復後點選「進入管理後台」，不要重新安裝。",
			"invalid_input": "輸入格式不正確或表單過大。", "invalid_redis": "請填寫正確的 Redis 位址和連接埠。", "invalid_admin": "請填寫有效的管理員名稱及符合要求的密碼，密碼首尾不能有空白。", "invalid_url": "網站位址須為 http://IP:連接埠 或 https://網域，不要加上路徑。",
			"redis_failed": "Redis 連線驗證失敗，請檢查執行狀態、主機位址、連接埠、密碼及資料庫 0/1 的存取權限。", "save_failed": "儲存安裝資訊失敗，請檢查安裝目錄寫入權限和剩餘空間。", "database_failed": "初始化資料庫失敗，未儲存安裝結果；請檢查服務日誌後重試。",
			"locked": "已有設定或資料庫，安裝入口已鎖定。", "cross_site": "不允許跨站提交安裝資訊。", "invalid_token": "安裝授權碼不正確，請使用啟動腳本顯示的專屬安裝連結。", "password_mismatch": "兩次輸入的管理員密碼不一致。",
		},
		LocaleEN: {
			"title": "Dujiao Next · Setup", "intro": "Connect Redis and create your database, administrator and security keys.",
			"authorization": "Installation authorization", "token": "Installation code", "token_hint": "The private link printed by start.sh fills this automatically. The code is also stored in .install-token in the installation directory.",
			"redis": "01　Redis and delivery queue", "host": "Redis host", "port": "Port", "redis_password": "Redis password (leave empty if none)",
			"redis_hint": "Use the host-accessible address and mapped port shown in 1Panel. Setup checks cache database 0 and delivery queue database 1. A MariaDB database is not required.",
			"admin":      "02　Administrator", "username": "Username", "password": "Password", "confirm": "Confirm password", "password_hint": "At least 8 characters with uppercase, lowercase and numbers; at most 72 bytes.",
			"site": "03　Website", "site_url": "Website URL", "site_hint": "For example https://shop.example.com, or http://SERVER_IP:8081 for initial testing. Configure your domain and HTTPS before accepting payments.",
			"submit": "Install", "database_hint": "The SQLite database is created automatically. Only the administrator password hash is stored. Existing installations are protected.",
			"working": "Installing…", "working_hint": "Checking Redis and initializing data. Please wait and keep this page open.",
			"success": "Installation complete", "starting": "Configuration saved and setup locked. Waiting for the store to start before opening the admin panel.", "enter": "Open admin panel", "startup_wait": "The store has not started yet. Check the service logs, then open the admin panel after fixing the error. Do not reinstall.",
			"invalid_input": "Invalid input or form too large.", "invalid_redis": "Enter a valid Redis host and port.", "invalid_admin": "Enter a valid administrator username and a strong password without leading or trailing whitespace.", "invalid_url": "Use http://IP:PORT or https://DOMAIN without a path.",
			"redis_failed": "Redis verification failed. Check the service, host address, port, password and access to databases 0 and 1.", "save_failed": "Could not save installation data. Check directory write permissions and available disk space.", "database_failed": "Database initialization failed. No installation was saved. Check the service logs and retry.",
			"locked": "Configuration or a database already exists. Setup is locked.", "cross_site": "Cross-site installation submissions are not allowed.", "invalid_token": "Invalid installation code. Use the private link printed by start.sh.", "password_mismatch": "The administrator passwords do not match.",
		},
	} {
		for key, text := range bundle {
			messages[locale]["setup."+key] = text
		}
	}
}
