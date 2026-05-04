import re

with open("Server/v6/backend/main.go", "r", encoding="utf-8") as f:
    content = f.read()

# Fix the variable declarations
content = content.replace("var (\n\tGetConfig()  Config\n\tGetConfig()Mutex sync.RWMutex", "var (\n\tglobalConfig  Config\n\tglobalConfigMutex sync.RWMutex")
content = content.replace("func GetConfig() Config {\n\tGetConfig()Mutex.RLock()\n\tdefer GetConfig()Mutex.RUnlock()\n\treturn GetConfig()\n}\n\nfunc SetConfig(c Config) {\n\tGetConfig()Mutex.Lock()\n\tdefer GetConfig()Mutex.Unlock()\n\tGetConfig() = c\n}", "func GetConfig() Config {\n\tglobalConfigMutex.RLock()\n\tdefer globalConfigMutex.RUnlock()\n\treturn globalConfig\n}\n\nfunc SetConfig(c Config) {\n\tglobalConfigMutex.Lock()\n\tdefer globalConfigMutex.Unlock()\n\tglobalConfig = c\n}")

# Fix assignments
content = content.replace("GetConfig() = c", "SetConfig(c)")
content = content.replace("GetConfig() = config", "SetConfig(config)")

with open("Server/v6/backend/main.go", "w", encoding="utf-8") as f:
    f.write(content)
